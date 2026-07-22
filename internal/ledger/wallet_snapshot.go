package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/HONG-LOU/entcoin/internal/core"
)

// WalletSnapshot is a bounded, read-only view suitable for untrusted wallet clients.
type WalletSnapshot struct {
	Tip              Tip
	ConfirmedBalance uint64
	SpendableBalance uint64
	UTXO             core.UTXO
	UTXOTruncated    bool
	History          []TransactionRecord
}

// ReadWalletSnapshot reads wallet state from one database snapshot. UTXO and
// history limits protect public nodes from unbounded address queries.
func (l *Ledger) ReadWalletSnapshot(ctx context.Context, address string, utxoLimit, historyLimit int) (WalletSnapshot, error) {
	if err := core.ValidateAddress(address); err != nil {
		return WalletSnapshot{}, err
	}
	if utxoLimit <= 0 || utxoLimit > core.MaxTransactionInputs {
		return WalletSnapshot{}, fmt.Errorf("wallet UTXO limit must be between 1 and %d", core.MaxTransactionInputs)
	}
	if historyLimit <= 0 || historyLimit > 50 {
		return WalletSnapshot{}, fmt.Errorf("wallet history limit must be between 1 and 50")
	}
	tx, err := l.database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return WalletSnapshot{}, fmt.Errorf("begin wallet snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	tip, err := tipFromQuery(ctx, tx)
	if err != nil {
		return WalletSnapshot{}, err
	}
	if tip.Height >= math.MaxInt64 {
		return WalletSnapshot{}, fmt.Errorf("chain height is exhausted")
	}
	confirmed, err := confirmedBalance(ctx, tx, address)
	if err != nil {
		return WalletSnapshot{}, err
	}
	spendable, outputs, truncated, err := boundedSpendableUTXO(ctx, tx, address, tip.Height+1, utxoLimit)
	if err != nil {
		return WalletSnapshot{}, err
	}
	history, err := boundedHistory(ctx, tx, address, historyLimit, tip.Height)
	if err != nil {
		return WalletSnapshot{}, err
	}
	return WalletSnapshot{
		Tip: tip, ConfirmedBalance: confirmed, SpendableBalance: spendable,
		UTXO: outputs, UTXOTruncated: truncated, History: history,
	}, nil
}

func confirmedBalance(ctx context.Context, query sqlQueryer, address string) (uint64, error) {
	var amount sql.NullInt64
	if err := query.QueryRowContext(ctx, `SELECT SUM(amount) FROM utxos WHERE address COLLATE NOCASE = ?`, address).Scan(&amount); err != nil {
		return 0, fmt.Errorf("query confirmed balance: %w", err)
	}
	if !amount.Valid {
		return 0, nil
	}
	if amount.Int64 < 0 {
		return 0, fmt.Errorf("stored confirmed balance is negative")
	}
	return uint64(amount.Int64), nil
}

func boundedSpendableUTXO(ctx context.Context, query sqlQueryer, address string, spendingHeight uint64, limit int) (uint64, core.UTXO, bool, error) {
	enforceMaturity := spendingHeight >= core.CoinbaseMaturityActivationHeight
	matureThrough := uint64(0)
	if enforceMaturity && spendingHeight >= core.CoinbaseMaturity {
		matureThrough = spendingHeight - core.CoinbaseMaturity
	}
	const spendableQuery = `
		SELECT u.tx_id, u.output_index, u.amount, u.address
		FROM utxos u
		WHERE u.address COLLATE NOCASE = ? AND u.created_height <= ?
		  AND (? = 0 OR u.coinbase = 0 OR u.created_height <= ?)
		  AND NOT EXISTS (SELECT 1 FROM mempool_inputs mi WHERE mi.input_tx_id = u.tx_id AND mi.input_index = u.output_index)
		UNION ALL
		SELECT o.tx_id, o.output_index, o.amount, o.address
		FROM mempool_outputs o
		WHERE o.address COLLATE NOCASE = ?
		  AND NOT EXISTS (SELECT 1 FROM mempool_inputs mi WHERE mi.input_tx_id = o.tx_id AND mi.input_index = o.output_index)`
	args := []any{address, int64(spendingHeight), enforceMaturity, int64(matureThrough), address}
	var total sql.NullInt64
	if err := query.QueryRowContext(ctx, `SELECT SUM(amount) FROM (`+spendableQuery+`)`, args...).Scan(&total); err != nil {
		return 0, nil, false, fmt.Errorf("sum spendable outputs: %w", err)
	}
	if total.Valid && total.Int64 < 0 {
		return 0, nil, false, fmt.Errorf("stored spendable balance is negative")
	}
	rows, err := query.QueryContext(ctx, `SELECT tx_id, output_index, amount, address FROM (`+spendableQuery+`) ORDER BY tx_id, output_index LIMIT ?`, append(args, limit+1)...)
	if err != nil {
		return 0, nil, false, fmt.Errorf("query bounded spendable outputs: %w", err)
	}
	defer rows.Close()
	outputs := make(core.UTXO, limit)
	truncated := false
	for rows.Next() {
		var txID, outputAddress string
		var index, amount int64
		if err := rows.Scan(&txID, &index, &amount, &outputAddress); err != nil {
			return 0, nil, false, fmt.Errorf("scan bounded spendable output: %w", err)
		}
		if len(outputs) == limit {
			truncated = true
			continue
		}
		if index < 0 || index > math.MaxUint32 || amount <= 0 {
			return 0, nil, false, fmt.Errorf("stored spendable output contains invalid values")
		}
		outputs[core.Outpoint{TxID: txID, Index: uint32(index)}] = core.TxOutput{Amount: uint64(amount), Address: outputAddress}
	}
	if err := rows.Err(); err != nil {
		return 0, nil, false, fmt.Errorf("iterate bounded spendable outputs: %w", err)
	}
	spendable := uint64(0)
	if total.Valid {
		spendable = uint64(total.Int64)
	}
	return spendable, outputs, truncated, nil
}

func boundedHistory(ctx context.Context, query sqlQueryer, address string, limit int, tipHeight uint64) ([]TransactionRecord, error) {
	records, err := pendingHistory(ctx, query, address, limit, TransactionHistoryAll)
	if err != nil || len(records) == limit {
		return records, err
	}
	confirmed, err := confirmedHistory(ctx, query, address, limit-len(records), tipHeight, TransactionHistoryAll)
	if err != nil {
		return nil, err
	}
	return append(records, confirmed...), nil
}
