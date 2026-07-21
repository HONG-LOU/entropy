package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/HONG-LOU/entcoin/internal/core"
)

type TransactionHistoryFilter string

const (
	TransactionHistoryAll      TransactionHistoryFilter = "all"
	TransactionHistoryReceived TransactionHistoryFilter = "received"
	TransactionHistorySent     TransactionHistoryFilter = "sent"
	TransactionHistoryMining   TransactionHistoryFilter = "mining"
)

func (l *Ledger) TransactionHistory(ctx context.Context, address string, limit int) ([]TransactionRecord, error) {
	return l.FilteredTransactionHistory(ctx, address, limit, TransactionHistoryAll)
}

func (l *Ledger) FilteredTransactionHistory(
	ctx context.Context,
	address string,
	limit int,
	filter TransactionHistoryFilter,
) ([]TransactionRecord, error) {
	if err := core.ValidateAddress(address); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1_000 {
		return nil, fmt.Errorf("transaction history limit must be between 1 and 1000")
	}
	if !filter.valid() {
		return nil, fmt.Errorf("unknown transaction history filter %q", filter)
	}
	tx, err := l.database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin transaction history snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	tip, err := tipFromQuery(ctx, tx)
	if err != nil {
		return nil, err
	}
	records, err := pendingHistory(ctx, tx, address, limit, filter)
	if err != nil {
		return nil, err
	}
	if len(records) == limit {
		return records, nil
	}
	confirmed, err := confirmedHistory(ctx, tx, address, limit-len(records), tip.Height, filter)
	if err != nil {
		return nil, err
	}
	return append(records, confirmed...), nil
}

func (filter TransactionHistoryFilter) valid() bool {
	switch filter {
	case TransactionHistoryAll, TransactionHistoryReceived, TransactionHistorySent, TransactionHistoryMining:
		return true
	default:
		return false
	}
}

func pendingHistory(
	ctx context.Context,
	query sqlQueryer,
	address string,
	limit int,
	filter TransactionHistoryFilter,
) ([]TransactionRecord, error) {
	if filter == TransactionHistoryMining {
		return []TransactionRecord{}, nil
	}
	rows, err := query.QueryContext(ctx, `
		SELECT m.tx_id, m.first_seen, m.data,
		       COALESCE((
		           SELECT SUM(output.amount) FROM mempool_outputs output
		           WHERE output.tx_id = m.tx_id AND output.address COLLATE NOCASE = ?
		       ), 0) AS received,
		       COALESCE((
		           SELECT SUM(COALESCE(confirmed.amount, pending.amount))
		           FROM mempool_inputs input
		           LEFT JOIN utxos confirmed
		             ON confirmed.tx_id = input.input_tx_id AND confirmed.output_index = input.input_index
		           LEFT JOIN mempool_outputs pending
		             ON pending.tx_id = input.input_tx_id AND pending.output_index = input.input_index
		           WHERE input.tx_id = m.tx_id
		             AND (confirmed.address COLLATE NOCASE = ? OR pending.address COLLATE NOCASE = ?)
		       ), 0) AS sent
		FROM mempool m
		WHERE (EXISTS (
			SELECT 1 FROM mempool_outputs output
			WHERE output.tx_id = m.tx_id AND output.address COLLATE NOCASE = ?
		) OR EXISTS (
			SELECT 1
			FROM mempool_inputs input
			LEFT JOIN utxos confirmed
			  ON confirmed.tx_id = input.input_tx_id AND confirmed.output_index = input.input_index
			LEFT JOIN mempool_outputs pending
			  ON pending.tx_id = input.input_tx_id AND pending.output_index = input.input_index
			WHERE input.tx_id = m.tx_id
			  AND (confirmed.address COLLATE NOCASE = ? OR pending.address COLLATE NOCASE = ?)
		))
		AND (
			? = 'all'
			OR (? = 'sent' AND EXISTS (
				SELECT 1
				FROM mempool_inputs filter_input
				LEFT JOIN utxos filter_confirmed
				  ON filter_confirmed.tx_id = filter_input.input_tx_id AND filter_confirmed.output_index = filter_input.input_index
				LEFT JOIN mempool_outputs filter_pending
				  ON filter_pending.tx_id = filter_input.input_tx_id AND filter_pending.output_index = filter_input.input_index
				WHERE filter_input.tx_id = m.tx_id
				  AND (filter_confirmed.address COLLATE NOCASE = ? OR filter_pending.address COLLATE NOCASE = ?)
			))
			OR (? = 'received' AND NOT EXISTS (
				SELECT 1
				FROM mempool_inputs filter_input
				LEFT JOIN utxos filter_confirmed
				  ON filter_confirmed.tx_id = filter_input.input_tx_id AND filter_confirmed.output_index = filter_input.input_index
				LEFT JOIN mempool_outputs filter_pending
				  ON filter_pending.tx_id = filter_input.input_tx_id AND filter_pending.output_index = filter_input.input_index
				WHERE filter_input.tx_id = m.tx_id
				  AND (filter_confirmed.address COLLATE NOCASE = ? OR filter_pending.address COLLATE NOCASE = ?)
			))
		)
		ORDER BY m.sequence DESC
		LIMIT ?
	`, address, address, address, address, address, address,
		filter, filter, address, address, filter, address, address, limit)
	if err != nil {
		return nil, fmt.Errorf("query pending transaction history: %w", err)
	}
	defer rows.Close()
	records := make([]TransactionRecord, 0)
	for rows.Next() {
		var id string
		var firstSeen int64
		var data []byte
		var received, sent int64
		if err := rows.Scan(&id, &firstSeen, &data, &received, &sent); err != nil {
			return nil, fmt.Errorf("scan pending transaction history: %w", err)
		}
		if received < 0 || sent < 0 {
			return nil, fmt.Errorf("pending transaction %s has invalid indexed amounts", id)
		}
		var transaction core.Transaction
		if err := decodeJSON(data, &transaction); err != nil {
			return nil, fmt.Errorf("decode pending transaction %s: %w", id, err)
		}
		if transaction.ID != id {
			return nil, fmt.Errorf("pending transaction %s data does not match its index", id)
		}
		records = append(records, TransactionRecord{
			ID:          id,
			Pending:     true,
			Timestamp:   firstSeen,
			Received:    uint64(received),
			Sent:        uint64(sent),
			Transaction: transaction,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending transaction history: %w", err)
	}
	return records, nil
}

func confirmedHistory(
	ctx context.Context,
	query sqlQueryer,
	address string,
	limit int,
	tipHeight uint64,
	filter TransactionHistoryFilter,
) ([]TransactionRecord, error) {
	rows, err := query.QueryContext(ctx, `
		SELECT t.id, t.block_height, t.tx_index, t.coinbase, t.data, b.hash, b.timestamp,
		       COALESCE((
		           SELECT SUM(received.amount) FROM transaction_addresses received
		           WHERE received.tx_id = t.id AND received.address COLLATE NOCASE = ? AND received.direction = ?
		       ), 0) AS received,
		       COALESCE((
		           SELECT SUM(sent.amount) FROM transaction_addresses sent
		           WHERE sent.tx_id = t.id AND sent.address COLLATE NOCASE = ? AND sent.direction = ?
		       ), 0) AS sent
		FROM transactions t
		JOIN blocks b ON b.height = t.block_height
		WHERE EXISTS (
			SELECT 1 FROM transaction_addresses ta
			WHERE ta.tx_id = t.id AND ta.address COLLATE NOCASE = ?
		)
		AND (
			? = 'all'
			OR (? = 'mining' AND t.coinbase = 1)
			OR (? = 'sent' AND t.coinbase = 0 AND EXISTS (
				SELECT 1 FROM transaction_addresses filter_sent
				WHERE filter_sent.tx_id = t.id AND filter_sent.address COLLATE NOCASE = ? AND filter_sent.direction = ?
			))
			OR (? = 'received' AND t.coinbase = 0 AND NOT EXISTS (
				SELECT 1 FROM transaction_addresses filter_sent
				WHERE filter_sent.tx_id = t.id AND filter_sent.address COLLATE NOCASE = ? AND filter_sent.direction = ?
			))
		)
		ORDER BY t.block_height DESC, t.tx_index DESC
		LIMIT ?
	`, address, addressDirectionOutput, address, addressDirectionInput, address,
		filter, filter, filter, address, addressDirectionInput, filter, address, addressDirectionInput, limit)
	if err != nil {
		return nil, fmt.Errorf("query confirmed transaction history: %w", err)
	}
	defer rows.Close()
	records := make([]TransactionRecord, 0, limit)
	for rows.Next() {
		var id, blockHash string
		var blockHeight, position, coinbase, timestamp, received, sent int64
		var data []byte
		if err := rows.Scan(&id, &blockHeight, &position, &coinbase, &data, &blockHash, &timestamp, &received, &sent); err != nil {
			return nil, fmt.Errorf("scan confirmed transaction history: %w", err)
		}
		if blockHeight < 0 || uint64(blockHeight) > tipHeight || position < 0 || position > math.MaxInt ||
			(coinbase != 0 && coinbase != 1) || received < 0 || sent < 0 {
			return nil, fmt.Errorf("confirmed transaction %s index contains invalid values", id)
		}
		var transaction core.Transaction
		pruned := len(data) == 0
		if pruned {
			transaction.ID = id
			transaction.Coinbase = coinbase == 1
		} else {
			if err := decodeJSON(data, &transaction); err != nil {
				return nil, fmt.Errorf("decode confirmed transaction %s: %w", id, err)
			}
			if transaction.ID != id || transaction.Coinbase != (coinbase == 1) {
				return nil, fmt.Errorf("confirmed transaction %s data does not match its index", id)
			}
		}
		height := uint64(blockHeight)
		records = append(records, TransactionRecord{
			ID:            id,
			BlockHeight:   &height,
			BlockHash:     blockHash,
			Position:      int(position),
			Coinbase:      coinbase == 1,
			Pruned:        pruned,
			Confirmations: tipHeight - height + 1,
			Timestamp:     timestamp,
			Received:      uint64(received),
			Sent:          uint64(sent),
			Transaction:   transaction,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate confirmed transaction history: %w", err)
	}
	return records, nil
}
