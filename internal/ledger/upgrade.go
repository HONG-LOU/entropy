package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"entropy/internal/core"
)

func (l *Ledger) upgradeV2Protocol(ctx context.Context) error {
	tip, err := l.Tip(ctx)
	if err != nil {
		return err
	}
	if tip.Height >= core.CoinbaseMaturityActivationHeight {
		prunedHeight, err := l.PrunedThrough(ctx)
		if err != nil {
			return err
		}
		if prunedHeight > 0 {
			return fmt.Errorf("v2 ledger is pruned through height %d; resync is required for the v3 consensus upgrade", prunedHeight)
		}
		if err := l.quickCheck(ctx); err != nil {
			return err
		}
		if err := l.validateStoredChainForV3(ctx, tip); err != nil {
			return fmt.Errorf("v2 ledger cannot be upgraded safely; resync is required: %w", err)
		}
	}

	tx, err := l.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin v3 protocol upgrade: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, "UPDATE meta SET value = ? WHERE key = 'protocol' AND value = ?", ProtocolName, "entropy-testnet-v2")
	if err != nil {
		return fmt.Errorf("upgrade ledger protocol metadata: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check ledger protocol upgrade: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("ledger protocol changed while upgrading metadata")
	}
	if mempoolMaturityRulesActive(tip.Height) {
		if err := rebuildMempool(ctx, tx, nil); err != nil {
			return fmt.Errorf("revalidate mempool for v3 protocol: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit v3 protocol upgrade: %w", err)
	}
	return nil
}

func mempoolMaturityRulesActive(tipHeight uint64) bool {
	return tipHeight < math.MaxUint64 && tipHeight+1 >= core.CoinbaseMaturityActivationHeight
}

func (l *Ledger) validateStoredChainForV3(ctx context.Context, tip Tip) error {
	rows, err := l.database.QueryContext(ctx, "SELECT height, data FROM blocks ORDER BY height")
	if err != nil {
		return fmt.Errorf("query archived blocks: %w", err)
	}
	blocks := make([]core.Block, 0)
	for rows.Next() {
		var height int64
		var data []byte
		if err := rows.Scan(&height, &data); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan archived block: %w", err)
		}
		if height < 0 || len(data) == 0 {
			_ = rows.Close()
			return fmt.Errorf("block %d body is not retained", height)
		}
		if uint64(height) != uint64(len(blocks)) {
			_ = rows.Close()
			return fmt.Errorf("archived block heights are not contiguous")
		}
		var block core.Block
		if err := decodeJSON(data, &block); err != nil {
			_ = rows.Close()
			return fmt.Errorf("decode archived block %d: %w", height, err)
		}
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate archived blocks: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close archived block query: %w", err)
	}
	if len(blocks) == 0 || uint64(len(blocks)-1) != tip.Height || blocks[len(blocks)-1].Hash != tip.Hash {
		return fmt.Errorf("archived blocks do not match the active tip")
	}
	state := &core.State{
		Version: core.StateVersion,
		Name:    core.ChainName,
		Symbol:  core.ChainSymbol,
		Blocks:  blocks,
		Pending: []core.Transaction{},
	}
	if err := state.Validate(); err != nil {
		return fmt.Errorf("replay archived chain under v3 rules: %w", err)
	}
	if state.CumulativeWork().Cmp(tip.Work) != 0 {
		return fmt.Errorf("stored cumulative work does not match replayed chain")
	}
	expected := replayedUTXORecords(blocks)
	stored, err := loadAllUTXORecords(ctx, l.database)
	if err != nil {
		return err
	}
	if len(stored) != len(expected) {
		return fmt.Errorf("stored UTXO count does not match replayed chain")
	}
	for outpoint, want := range expected {
		got, exists := stored[outpoint]
		if !exists || got != want {
			return fmt.Errorf("stored UTXO %s:%d does not match replayed chain", outpoint.TxID, outpoint.Index)
		}
	}
	return nil
}

func replayedUTXORecords(blocks []core.Block) map[core.Outpoint]UTXORecord {
	utxo := make(map[core.Outpoint]UTXORecord)
	for _, block := range blocks[1:] {
		for _, transaction := range block.Transactions[1:] {
			for _, input := range transaction.Inputs {
				delete(utxo, core.Outpoint{TxID: input.TxID, Index: input.OutputIndex})
			}
			addReplayedOutputs(utxo, transaction, block.Height, false)
		}
		addReplayedOutputs(utxo, block.Transactions[0], block.Height, true)
	}
	return utxo
}

func addReplayedOutputs(utxo map[core.Outpoint]UTXORecord, transaction core.Transaction, height uint64, coinbase bool) {
	for index, output := range transaction.Outputs {
		outpoint := core.Outpoint{TxID: transaction.ID, Index: uint32(index)}
		utxo[outpoint] = UTXORecord{
			TxID: transaction.ID, OutputIndex: uint32(index), Amount: output.Amount,
			Address: output.Address, CreatedHeight: height, Coinbase: coinbase,
		}
	}
}

func loadAllUTXORecords(ctx context.Context, database *sql.DB) (map[core.Outpoint]UTXORecord, error) {
	rows, err := database.QueryContext(ctx, `
		SELECT tx_id, output_index, amount, address, created_height, coinbase FROM utxos
	`)
	if err != nil {
		return nil, fmt.Errorf("query stored UTXOs: %w", err)
	}
	defer rows.Close()
	records := make(map[core.Outpoint]UTXORecord)
	for rows.Next() {
		var record UTXORecord
		var index, amount, height, coinbase int64
		if err := rows.Scan(&record.TxID, &index, &amount, &record.Address, &height, &coinbase); err != nil {
			return nil, fmt.Errorf("scan stored UTXO: %w", err)
		}
		if index < 0 || index > math.MaxUint32 || amount <= 0 || height < 0 || (coinbase != 0 && coinbase != 1) {
			return nil, fmt.Errorf("stored UTXO contains invalid values")
		}
		record.OutputIndex = uint32(index)
		record.Amount = uint64(amount)
		record.CreatedHeight = uint64(height)
		record.Coinbase = coinbase == 1
		outpoint := core.Outpoint{TxID: record.TxID, Index: record.OutputIndex}
		if _, exists := records[outpoint]; exists {
			return nil, fmt.Errorf("stored UTXO index is duplicated")
		}
		records[outpoint] = record
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stored UTXOs: %w", err)
	}
	return records, nil
}
