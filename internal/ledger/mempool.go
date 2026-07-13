package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"entropy/internal/core"
)

var ErrTransactionAlreadyKnown = errors.New("transaction is already known")

const (
	MaxMempoolBytes              int64  = 32 << 20
	MaxMempoolInputs                    = 20_000
	MinimumRelayFeePerKibiByte   uint64 = 1_000
	mempoolRelayFeeKibiByteUnits        = 1 << 10
)

type mempoolResourceUsage struct {
	count  int64
	bytes  int64
	inputs int64
}

type pendingMempoolTransaction struct {
	transaction core.Transaction
	firstSeen   int64
}

type sqlQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type mempoolRejection struct {
	err error
}

func (e mempoolRejection) Error() string { return e.err.Error() }
func (e mempoolRejection) Unwrap() error { return e.err }

func rejectMempool(err error) error {
	return mempoolRejection{err: err}
}

func rejectMempoolf(format string, arguments ...any) error {
	return rejectMempool(fmt.Errorf(format, arguments...))
}

func isMempoolRejection(err error) bool {
	var rejection mempoolRejection
	return errors.As(err, &rejection)
}

func (l *Ledger) AddTransaction(ctx context.Context, transaction core.Transaction) error {
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	tx, err := l.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin mempool transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	tip, err := tipFromQuery(ctx, tx)
	if err != nil {
		return err
	}
	if tip.Height >= math.MaxInt64 {
		return fmt.Errorf("chain height is exhausted")
	}
	if err := addMempoolTransaction(ctx, tx, transaction, time.Now().Unix(), tip.Height+1); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit mempool transaction: %w", err)
	}
	return nil
}

func addMempoolTransaction(
	ctx context.Context,
	query sqlQueryer,
	transaction core.Transaction,
	firstSeen int64,
	spendingHeight uint64,
) error {
	return addMempoolTransactionWithUsage(ctx, query, transaction, firstSeen, spendingHeight, nil)
}

func addMempoolTransactionWithUsage(
	ctx context.Context,
	query sqlQueryer,
	transaction core.Transaction,
	firstSeen int64,
	spendingHeight uint64,
	usage *mempoolResourceUsage,
) error {
	if transaction.Coinbase {
		return rejectMempoolf("coinbase transaction cannot enter mempool")
	}
	var exists int
	err := query.QueryRowContext(ctx, `
        SELECT 1 FROM transactions WHERE id = ?
        UNION ALL SELECT 1 FROM mempool WHERE tx_id = ? LIMIT 1
    `, transaction.ID, transaction.ID).Scan(&exists)
	if err == nil {
		return rejectMempool(fmt.Errorf("%w: %s", ErrTransactionAlreadyKnown, transaction.ID))
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check existing transaction: %w", err)
	}
	if usage == nil {
		loaded, err := readMempoolResourceUsage(ctx, query)
		if err != nil {
			return err
		}
		usage = &loaded
	}
	encodedSize := core.EncodedTransactionSize(transaction)
	if err := validateMempoolResourceBudget(usage.count, usage.bytes, usage.inputs, encodedSize, len(transaction.Inputs)); err != nil {
		return rejectMempool(err)
	}

	utxo := make(core.UTXO, len(transaction.Inputs))
	for _, input := range transaction.Inputs {
		outpoint := core.Outpoint{TxID: input.TxID, Index: input.OutputIndex}
		output, err := lookupMempoolSpendable(ctx, query, outpoint, spendingHeight)
		if err != nil {
			return err
		}
		utxo[outpoint] = output
	}
	fee, err := core.ValidateRegularTransaction(transaction, utxo)
	if err != nil {
		return rejectMempool(err)
	}
	minimumFee := minimumRelayFee(encodedSize)
	if fee < minimumFee {
		return rejectMempoolf("transaction fee %d is below relay minimum %d", fee, minimumFee)
	}
	if fee > math.MaxInt64 {
		return rejectMempoolf("transaction fee exceeds storage range")
	}
	for index, output := range transaction.Outputs {
		if output.Amount > math.MaxInt64 {
			return rejectMempoolf("transaction output %d exceeds storage range", index)
		}
	}
	data, err := encodeJSON(transaction)
	if err != nil {
		return fmt.Errorf("encode mempool transaction: %w", err)
	}
	if _, err := query.ExecContext(ctx, `
        INSERT INTO mempool(tx_id, first_seen, fee, encoded_size, data)
        VALUES(?, ?, ?, ?, ?)
	`, transaction.ID, firstSeen, int64(fee), encodedSize, data); err != nil {
		return fmt.Errorf("insert mempool transaction: %w", err)
	}
	for _, input := range transaction.Inputs {
		if _, err := query.ExecContext(ctx, `
            INSERT INTO mempool_inputs(tx_id, input_tx_id, input_index) VALUES(?, ?, ?)
        `, transaction.ID, input.TxID, int64(input.OutputIndex)); err != nil {
			return fmt.Errorf("index mempool input: %w", err)
		}
	}
	for index, output := range transaction.Outputs {
		if _, err := query.ExecContext(ctx, `
            INSERT INTO mempool_outputs(tx_id, output_index, amount, address) VALUES(?, ?, ?, ?)
        `, transaction.ID, index, int64(output.Amount), output.Address); err != nil {
			return fmt.Errorf("index mempool output: %w", err)
		}
	}
	usage.count++
	usage.bytes += int64(encodedSize)
	usage.inputs += int64(len(transaction.Inputs))
	return nil
}

func readMempoolResourceUsage(ctx context.Context, query sqlQueryer) (mempoolResourceUsage, error) {
	var usage mempoolResourceUsage
	if err := query.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(encoded_size), 0),
		       (SELECT COUNT(*) FROM mempool_inputs)
		FROM mempool
	`).Scan(&usage.count, &usage.bytes, &usage.inputs); err != nil {
		return mempoolResourceUsage{}, fmt.Errorf("read mempool resource usage: %w", err)
	}
	if usage.count < 0 || usage.bytes < 0 || usage.inputs < 0 {
		return mempoolResourceUsage{}, fmt.Errorf("mempool resource accounting contains invalid values")
	}
	return usage, nil
}

func validateMempoolResourceBudget(count, totalBytes, totalInputs int64, encodedSize, inputs int) error {
	if count < 0 || totalBytes < 0 || totalInputs < 0 || encodedSize <= 0 || inputs <= 0 {
		return fmt.Errorf("mempool resource accounting contains invalid values")
	}
	if count >= core.MaxPendingTransactions {
		return fmt.Errorf("pending transaction limit reached")
	}
	if int64(encodedSize) > MaxMempoolBytes-totalBytes {
		return fmt.Errorf("pending transaction byte budget reached")
	}
	if int64(inputs) > int64(MaxMempoolInputs)-totalInputs {
		return fmt.Errorf("pending transaction input budget reached")
	}
	return nil
}

func minimumRelayFee(encodedSize int) uint64 {
	if encodedSize <= 0 {
		return 0
	}
	kibibytes := (uint64(encodedSize) + mempoolRelayFeeKibiByteUnits - 1) / mempoolRelayFeeKibiByteUnits
	return kibibytes * MinimumRelayFeePerKibiByte
}

func lookupMempoolSpendable(
	ctx context.Context,
	query sqlQueryer,
	outpoint core.Outpoint,
	spendingHeight uint64,
) (core.TxOutput, error) {
	var spent int
	err := query.QueryRowContext(ctx, `
        SELECT 1 FROM mempool_inputs WHERE input_tx_id = ? AND input_index = ? LIMIT 1
    `, outpoint.TxID, int64(outpoint.Index)).Scan(&spent)
	if err == nil {
		return core.TxOutput{}, rejectMempoolf("input %s:%d is already spent by a pending transaction", outpoint.TxID, outpoint.Index)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return core.TxOutput{}, fmt.Errorf("check pending spend: %w", err)
	}
	var amount int64
	var address string
	var createdHeight int64
	var coinbase int
	confirmed := true
	err = query.QueryRowContext(ctx, `
		SELECT amount, address, created_height, coinbase
		FROM utxos WHERE tx_id = ? AND output_index = ?
	`, outpoint.TxID, int64(outpoint.Index)).Scan(&amount, &address, &createdHeight, &coinbase)
	if errors.Is(err, sql.ErrNoRows) {
		confirmed = false
		err = query.QueryRowContext(ctx, `
			SELECT amount, address FROM mempool_outputs WHERE tx_id = ? AND output_index = ?
        `, outpoint.TxID, int64(outpoint.Index)).Scan(&amount, &address)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return core.TxOutput{}, rejectMempoolf("input %s:%d is missing or already spent", outpoint.TxID, outpoint.Index)
	}
	if err != nil {
		return core.TxOutput{}, fmt.Errorf("read transaction input: %w", err)
	}
	if amount <= 0 || (confirmed && (createdHeight < 0 || (coinbase != 0 && coinbase != 1))) {
		return core.TxOutput{}, fmt.Errorf("stored transaction output has invalid amount")
	}
	if confirmed && coinbase == 1 && !core.IsCoinbaseMature(uint64(createdHeight), spendingHeight) {
		return core.TxOutput{}, rejectMempoolf("coinbase input %s:%d is immature", outpoint.TxID, outpoint.Index)
	}
	return core.TxOutput{Amount: uint64(amount), Address: address}, nil
}

func (l *Ledger) SpendableUTXO(ctx context.Context, address string) (core.UTXO, error) {
	if err := core.ValidateAddress(address); err != nil {
		return nil, err
	}
	tx, err := l.database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin spendable output snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	tip, err := tipFromQuery(ctx, tx)
	if err != nil {
		return nil, err
	}
	if tip.Height >= math.MaxInt64 {
		return nil, fmt.Errorf("chain height is exhausted")
	}
	return spendableUTXO(ctx, tx, address, tip.Height+1)
}

func spendableUTXO(ctx context.Context, query sqlQueryer, address string, spendingHeight uint64) (core.UTXO, error) {
	enforceMaturity := spendingHeight >= core.CoinbaseMaturityActivationHeight
	matureThrough := uint64(0)
	if enforceMaturity {
		matureThrough = spendingHeight - core.CoinbaseMaturity
	}
	rows, err := query.QueryContext(ctx, `
        SELECT u.tx_id, u.output_index, u.amount, u.address
		FROM utxos u
		WHERE u.address COLLATE NOCASE = ?
		  AND u.created_height <= ?
		  AND (? = 0 OR u.coinbase = 0 OR u.created_height <= ?)
          AND NOT EXISTS (
              SELECT 1 FROM mempool_inputs mi
              WHERE mi.input_tx_id = u.tx_id AND mi.input_index = u.output_index
          )
        UNION ALL
        SELECT o.tx_id, o.output_index, o.amount, o.address
        FROM mempool_outputs o
		WHERE o.address COLLATE NOCASE = ?
          AND NOT EXISTS (
              SELECT 1 FROM mempool_inputs mi
              WHERE mi.input_tx_id = o.tx_id AND mi.input_index = o.output_index
          )
	`, address, int64(spendingHeight), enforceMaturity, int64(matureThrough), address)
	if err != nil {
		return nil, fmt.Errorf("query spendable outputs: %w", err)
	}
	defer rows.Close()
	utxo := make(core.UTXO)
	for rows.Next() {
		var txID string
		var index int64
		var amount int64
		var outputAddress string
		if err := rows.Scan(&txID, &index, &amount, &outputAddress); err != nil {
			return nil, fmt.Errorf("scan spendable output: %w", err)
		}
		if index < 0 || index > math.MaxUint32 || amount <= 0 {
			return nil, fmt.Errorf("stored spendable output contains invalid values")
		}
		utxo[core.Outpoint{TxID: txID, Index: uint32(index)}] = core.TxOutput{Amount: uint64(amount), Address: outputAddress}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spendable outputs: %w", err)
	}
	return utxo, nil
}

func (l *Ledger) Balances(ctx context.Context, address string) (uint64, uint64, error) {
	if err := core.ValidateAddress(address); err != nil {
		return 0, 0, err
	}
	tx, err := l.database.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return 0, 0, fmt.Errorf("begin balance snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var confirmed sql.NullInt64
	if err := tx.QueryRowContext(ctx, "SELECT SUM(amount) FROM utxos WHERE address COLLATE NOCASE = ?", address).Scan(&confirmed); err != nil {
		return 0, 0, fmt.Errorf("query confirmed balance: %w", err)
	}
	confirmedAmount := uint64(0)
	if confirmed.Valid {
		if confirmed.Int64 < 0 {
			return 0, 0, fmt.Errorf("stored confirmed balance is negative")
		}
		confirmedAmount = uint64(confirmed.Int64)
	}
	tip, err := tipFromQuery(ctx, tx)
	if err != nil {
		return 0, 0, err
	}
	if tip.Height >= math.MaxInt64 {
		return 0, 0, fmt.Errorf("chain height is exhausted")
	}
	spendableUTXO, err := spendableUTXO(ctx, tx, address, tip.Height+1)
	if err != nil {
		return 0, 0, err
	}
	spendable := uint64(0)
	for _, output := range spendableUTXO {
		if math.MaxUint64-spendable < output.Amount {
			return 0, 0, fmt.Errorf("spendable balance overflows")
		}
		spendable += output.Amount
	}
	return confirmedAmount, spendable, nil
}

func (l *Ledger) MempoolTransactions(ctx context.Context, limit int) ([]core.Transaction, error) {
	return l.MempoolTransactionsPage(ctx, limit, 0)
}

func (l *Ledger) MempoolTransactionsPage(ctx context.Context, limit, offset int) ([]core.Transaction, error) {
	if limit <= 0 || limit > core.MaxPendingTransactions {
		return nil, fmt.Errorf("mempool limit is invalid")
	}
	if offset < 0 || offset > core.MaxPendingTransactions {
		return nil, fmt.Errorf("mempool offset is invalid")
	}
	rows, err := l.database.QueryContext(ctx, "SELECT data FROM mempool ORDER BY sequence LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query mempool: %w", err)
	}
	defer rows.Close()
	transactions := make([]core.Transaction, 0)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scan mempool transaction: %w", err)
		}
		var transaction core.Transaction
		if err := decodeJSON(data, &transaction); err != nil {
			return nil, fmt.Errorf("decode mempool transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mempool: %w", err)
	}
	return transactions, nil
}

func (l *Ledger) MempoolCount(ctx context.Context) (int, error) {
	var count int
	if err := l.database.QueryRowContext(ctx, "SELECT COUNT(*) FROM mempool").Scan(&count); err != nil {
		return 0, fmt.Errorf("count mempool: %w", err)
	}
	return count, nil
}

func rebuildMempool(ctx context.Context, tx *sql.Tx, extra []core.Transaction) error {
	rows, err := tx.QueryContext(ctx, "SELECT data, first_seen FROM mempool ORDER BY sequence")
	if err != nil {
		return err
	}
	pendingTransactions := make([]pendingMempoolTransaction, 0)
	for rows.Next() {
		var data []byte
		var firstSeen int64
		if err := rows.Scan(&data, &firstSeen); err != nil {
			_ = rows.Close()
			return err
		}
		var transaction core.Transaction
		if err := decodeJSON(data, &transaction); err != nil {
			_ = rows.Close()
			return err
		}
		pendingTransactions = append(pendingTransactions, pendingMempoolTransaction{transaction: transaction, firstSeen: firstSeen})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	tip, err := tipFromQuery(ctx, tx)
	if err != nil {
		return err
	}
	if tip.Height >= math.MaxInt64 {
		return fmt.Errorf("chain height is exhausted")
	}
	spendingHeight := tip.Height + 1
	if _, err := tx.ExecContext(ctx, "DELETE FROM mempool"); err != nil {
		return err
	}
	now := time.Now().Unix()
	rebuilt := make([]pendingMempoolTransaction, 0, len(extra)+len(pendingTransactions))
	for _, transaction := range extra {
		rebuilt = append(rebuilt, pendingMempoolTransaction{transaction: transaction, firstSeen: now})
	}
	rebuilt = append(rebuilt, pendingTransactions...)
	return repopulateMempool(ctx, tx, rebuilt, spendingHeight)
}

func repopulateMempool(
	ctx context.Context,
	query sqlQueryer,
	pending []pendingMempoolTransaction,
	spendingHeight uint64,
) error {
	usage, err := readMempoolResourceUsage(ctx, query)
	if err != nil {
		return err
	}
	for _, item := range pending {
		if err := addMempoolTransactionWithUsage(ctx, query, item.transaction, item.firstSeen, spendingHeight, &usage); err != nil {
			if isMempoolRejection(err) {
				continue
			}
			return err
		}
	}
	return nil
}
