package ledger

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"

	"entropy/internal/core"
)

const (
	prunedThroughMetaKey = "pruned_through"
	pruneDepthMetaKey    = "prune_depth"
	MinimumPruneDepth    = uint64(core.FirstAdjustment)
	MaximumPruneDepth    = core.EmissionBlocks
)

var ErrReorgBeyondPrune = errors.New("reorganization crosses the pruned block horizon")

func (l *Ledger) PrunedThrough(ctx context.Context) (uint64, error) {
	height, err := prunedThrough(ctx, l.database)
	if err != nil {
		return 0, fmt.Errorf("read pruned block horizon: %w", err)
	}
	return height, nil
}

func (l *Ledger) PruneDepth(ctx context.Context) (uint64, error) {
	depth, err := uint64Metadata(ctx, l.database, pruneDepthMetaKey)
	if err != nil {
		return 0, fmt.Errorf("read prune depth: %w", err)
	}
	if err := validatePruneDepth(depth); err != nil {
		return 0, fmt.Errorf("stored prune depth: %w", err)
	}
	return depth, nil
}

func (l *Ledger) SetPruneDepth(ctx context.Context, depth uint64) error {
	if err := validatePruneDepth(depth); err != nil {
		return err
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	if _, err := l.database.ExecContext(ctx, `
		INSERT INTO meta(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, pruneDepthMetaKey, encodeUint64(depth)); err != nil {
		return fmt.Errorf("store prune depth: %w", err)
	}
	return nil
}

func validatePruneDepth(depth uint64) error {
	if depth == 0 {
		return nil
	}
	if depth < MinimumPruneDepth || depth > MaximumPruneDepth {
		return fmt.Errorf("prune depth must be 0 (archive) or between %d and %d blocks", MinimumPruneDepth, MaximumPruneDepth)
	}
	return nil
}

// Prune removes reconstructable bodies and undo data outside the requested
// active-tip retention window. Headers, transaction/address indexes and UTXOs
// remain available. A reorg below the returned height requires a chain resync.
func (l *Ledger) Prune(ctx context.Context, retainRecent uint64) (uint64, error) {
	if retainRecent == 0 {
		return 0, fmt.Errorf("prune retention must keep at least one recent block")
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()

	tx, err := l.database.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin ledger pruning: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	tip, err := tipFromQuery(ctx, tx)
	if err != nil {
		return 0, err
	}
	current, err := prunedThrough(ctx, tx)
	if err != nil {
		return 0, fmt.Errorf("read current prune horizon: %w", err)
	}
	if tip.Height <= retainRecent {
		return current, nil
	}
	cutoff := tip.Height - retainRecent
	if cutoff <= current {
		return current, nil
	}
	if _, err := tx.ExecContext(ctx, "UPDATE blocks SET data = NULL WHERE height BETWEEN 1 AND ?", int64(cutoff)); err != nil {
		return 0, fmt.Errorf("prune block bodies through %d: %w", cutoff, err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE transactions SET data = NULL WHERE block_height BETWEEN 1 AND ?", int64(cutoff)); err != nil {
		return 0, fmt.Errorf("prune transaction bodies through %d: %w", cutoff, err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM block_undo WHERE block_height BETWEEN 1 AND ?", int64(cutoff)); err != nil {
		return 0, fmt.Errorf("prune undo data through %d: %w", cutoff, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO meta(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, prunedThroughMetaKey, encodeUint64(cutoff)); err != nil {
		return 0, fmt.Errorf("record prune horizon %d: %w", cutoff, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit ledger pruning: %w", err)
	}
	return cutoff, nil
}

func prunedThrough(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (uint64, error) {
	return uint64Metadata(ctx, query, prunedThroughMetaKey)
}

func uint64Metadata(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, key string) (uint64, error) {
	var encoded []byte
	err := query.QueryRowContext(ctx, "SELECT value FROM meta WHERE key = ?", key).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if len(encoded) != 8 {
		return 0, fmt.Errorf("stored prune horizon has invalid length")
	}
	return binary.BigEndian.Uint64(encoded), nil
}
