package ledger

import (
	"context"
	"fmt"
	"math"
	"time"

	"entropy/internal/core"
)

// ImportState performs the one-time migration from the legacy replayed state.
// It intentionally refuses to overwrite an initialized SQLite chain.
func (l *Ledger) ImportState(ctx context.Context, state *core.State) error {
	if state == nil {
		return fmt.Errorf("legacy state is nil")
	}
	if err := state.ValidateConfirmed(); err != nil {
		return fmt.Errorf("validate legacy state: %w", err)
	}
	if len(state.Pending) > core.MaxPendingTransactions {
		return fmt.Errorf("validate legacy state: pending transaction limit exceeded")
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()

	tx, err := l.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin legacy state import: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	tip, err := tipFromQuery(ctx, tx)
	if err != nil {
		return err
	}
	if tip.Height != 0 || tip.Hash != core.GenesisBlock().Hash {
		return fmt.Errorf("legacy state import requires a fresh ledger")
	}
	var transactionCount int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM transactions").Scan(&transactionCount); err != nil {
		return fmt.Errorf("check fresh ledger transactions: %w", err)
	}
	if transactionCount != 0 {
		return fmt.Errorf("legacy state import requires an empty transaction index")
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM mempool"); err != nil {
		return fmt.Errorf("clear mempool before legacy import: %w", err)
	}
	for _, block := range state.Blocks[1:] {
		if err := connectBlock(ctx, tx, block); err != nil {
			return fmt.Errorf("import legacy block %d: %w", block.Height, err)
		}
	}
	importedTip, err := tipFromQuery(ctx, tx)
	if err != nil {
		return err
	}
	if importedTip.Height >= math.MaxInt64 {
		return fmt.Errorf("imported chain height is exhausted")
	}
	spendingHeight := importedTip.Height + 1
	firstSeen := time.Now().Unix()
	pending := make([]pendingMempoolTransaction, 0, len(state.Pending))
	for _, transaction := range state.Pending {
		pending = append(pending, pendingMempoolTransaction{transaction: transaction, firstSeen: firstSeen})
	}
	if err := repopulateMempool(ctx, tx, pending, spendingHeight); err != nil {
		return fmt.Errorf("import legacy pending transactions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy state import: %w", err)
	}
	return nil
}
