package ledger

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"entropy/internal/core"
)

var (
	ErrBlockNotFound = errors.New("block not found")
	ErrBlockPruned   = errors.New("block body has been pruned")
)

type rowScanner interface {
	Scan(dest ...any) error
}

func (l *Ledger) Tip(ctx context.Context) (Tip, error) {
	return tipFromQuery(ctx, l.database)
}

func tipFromQuery(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (Tip, error) {
	var (
		height     int64
		hash       string
		timestamp  int64
		difficulty int64
		work       []byte
	)
	err := query.QueryRowContext(ctx, `
        SELECT height, hash, timestamp, difficulty, cumulative_work
        FROM blocks ORDER BY height DESC LIMIT 1
    `).Scan(&height, &hash, &timestamp, &difficulty, &work)
	if err != nil {
		return Tip{}, fmt.Errorf("read chain tip: %w", err)
	}
	if height < 0 || difficulty < 0 || difficulty > math.MaxUint8 {
		return Tip{}, fmt.Errorf("stored chain tip contains invalid numeric values")
	}
	return Tip{Height: uint64(height), Hash: hash, Timestamp: timestamp, Difficulty: uint8(difficulty), Work: decodeWork(work)}, nil
}

func (l *Ledger) Header(ctx context.Context, height uint64) (core.Block, error) {
	if height > math.MaxInt64 {
		return core.Block{}, ErrBlockNotFound
	}
	return scanHeader(l.database.QueryRowContext(ctx, `
        SELECT height, hash, previous_hash, version, timestamp, merkle_root, difficulty, nonce
        FROM blocks WHERE height = ?
    `, int64(height)))
}

func (l *Ledger) Block(ctx context.Context, height uint64) (core.Block, error) {
	if height > math.MaxInt64 {
		return core.Block{}, ErrBlockNotFound
	}
	var data []byte
	err := l.database.QueryRowContext(ctx, "SELECT data FROM blocks WHERE height = ?", int64(height)).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Block{}, ErrBlockNotFound
	}
	if err != nil {
		return core.Block{}, fmt.Errorf("read block %d: %w", height, err)
	}
	if len(data) == 0 {
		return core.Block{}, ErrBlockPruned
	}
	var block core.Block
	if err := decodeJSON(data, &block); err != nil {
		return core.Block{}, fmt.Errorf("decode block %d: %w", height, err)
	}
	return block, nil
}

func (l *Ledger) BlockByHash(ctx context.Context, hash string) (core.Block, error) {
	var data []byte
	err := l.database.QueryRowContext(ctx, "SELECT data FROM blocks WHERE hash = ?", hash).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Block{}, ErrBlockNotFound
	}
	if err != nil {
		return core.Block{}, fmt.Errorf("read block %s: %w", hash, err)
	}
	if len(data) == 0 {
		return core.Block{}, ErrBlockPruned
	}
	var block core.Block
	if err := decodeJSON(data, &block); err != nil {
		return core.Block{}, fmt.Errorf("decode block %s: %w", hash, err)
	}
	return block, nil
}

func (l *Ledger) HashAt(ctx context.Context, height uint64) (string, error) {
	if height > math.MaxInt64 {
		return "", ErrBlockNotFound
	}
	var hash string
	err := l.database.QueryRowContext(ctx, "SELECT hash FROM blocks WHERE height = ?", int64(height)).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrBlockNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read block hash at %d: %w", height, err)
	}
	return hash, nil
}

func (l *Ledger) HeadersFrom(ctx context.Context, from uint64, limit int) ([]core.Block, error) {
	if limit <= 0 || limit > 2_000 {
		return nil, fmt.Errorf("header limit must be between 1 and 2000")
	}
	if from > math.MaxInt64 {
		return []core.Block{}, nil
	}
	rows, err := l.database.QueryContext(ctx, `
        SELECT height, hash, previous_hash, version, timestamp, merkle_root, difficulty, nonce
        FROM blocks WHERE height >= ? ORDER BY height LIMIT ?
    `, int64(from), limit)
	if err != nil {
		return nil, fmt.Errorf("query block headers: %w", err)
	}
	defer rows.Close()
	headers := make([]core.Block, 0, limit)
	for rows.Next() {
		header, err := scanHeader(rows)
		if err != nil {
			return nil, err
		}
		headers = append(headers, header)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate block headers: %w", err)
	}
	return headers, nil
}

func (l *Ledger) BlocksFrom(ctx context.Context, from uint64, limit int) ([]core.Block, error) {
	if limit <= 0 || limit > 128 {
		return nil, fmt.Errorf("block limit must be between 1 and 128")
	}
	if from > math.MaxInt64 {
		return []core.Block{}, nil
	}
	rows, err := l.database.QueryContext(ctx, `
        SELECT height, data FROM blocks WHERE height >= ? ORDER BY height LIMIT ?
    `, int64(from), limit)
	if err != nil {
		return nil, fmt.Errorf("query blocks: %w", err)
	}
	defer rows.Close()
	blocks := make([]core.Block, 0, limit)
	for rows.Next() {
		var height int64
		var data []byte
		if err := rows.Scan(&height, &data); err != nil {
			return nil, fmt.Errorf("scan block: %w", err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("block %d: %w", height, ErrBlockPruned)
		}
		var block core.Block
		if err := decodeJSON(data, &block); err != nil {
			return nil, fmt.Errorf("decode block %d: %w", height, err)
		}
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blocks: %w", err)
	}
	return blocks, nil
}

func (l *Ledger) RecentBlocks(ctx context.Context, limit int) ([]core.Block, error) {
	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("recent block limit must be between 1 and 100")
	}
	rows, err := l.database.QueryContext(ctx, `
		SELECT b.height, b.hash, b.previous_hash, b.version, b.timestamp, b.merkle_root,
		       b.difficulty, b.nonce,
		       (SELECT COUNT(*) FROM transactions t WHERE t.block_height = b.height)
		FROM blocks b ORDER BY b.height DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent blocks: %w", err)
	}
	defer rows.Close()
	blocks := make([]core.Block, 0, limit)
	for rows.Next() {
		block, transactionCount, err := scanHeaderWithTransactionCount(rows)
		if err != nil {
			return nil, err
		}
		block.Transactions = make([]core.Transaction, transactionCount)
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent blocks: %w", err)
	}
	return blocks, nil
}

func scanHeaderWithTransactionCount(scanner rowScanner) (core.Block, int, error) {
	var (
		height           int64
		hash             string
		previousHash     string
		version          int64
		timestamp        int64
		merkleRoot       string
		difficulty       int64
		nonce            []byte
		transactionCount int
	)
	if err := scanner.Scan(
		&height, &hash, &previousHash, &version, &timestamp, &merkleRoot,
		&difficulty, &nonce, &transactionCount,
	); err != nil {
		return core.Block{}, 0, fmt.Errorf("scan recent block: %w", err)
	}
	if height < 0 || version < 0 || version > math.MaxUint32 || difficulty < 0 || difficulty > math.MaxUint8 ||
		len(nonce) != 8 || transactionCount < 0 || transactionCount > core.MaxBlockTransactions {
		return core.Block{}, 0, fmt.Errorf("stored recent block contains invalid values")
	}
	return core.Block{
		Version:      uint32(version),
		Height:       uint64(height),
		Timestamp:    timestamp,
		PreviousHash: previousHash,
		MerkleRoot:   merkleRoot,
		Difficulty:   uint8(difficulty),
		Nonce:        binary.BigEndian.Uint64(nonce),
		Hash:         hash,
	}, transactionCount, nil
}

func (l *Ledger) HeaderWindow(ctx context.Context, throughHeight uint64) ([]core.Block, error) {
	const window = core.FirstAdjustment
	start := uint64(0)
	if throughHeight+1 > window {
		start = throughHeight + 1 - window
	}
	limit := int(throughHeight-start) + 1
	return l.HeadersFrom(ctx, start, limit)
}

func (l *Ledger) BlockLocator(ctx context.Context) ([]string, error) {
	tip, err := l.Tip(ctx)
	if err != nil {
		return nil, err
	}
	locator := make([]string, 0, 64)
	height := tip.Height
	step := uint64(1)
	for {
		hash, err := l.HashAt(ctx, height)
		if err != nil {
			return nil, err
		}
		locator = append(locator, hash)
		if height == 0 || len(locator) >= 64 {
			break
		}
		if len(locator) > 10 {
			step *= 2
		}
		if step > height {
			height = 0
		} else {
			height -= step
		}
	}
	if locator[len(locator)-1] != core.GenesisBlock().Hash {
		locator = append(locator, core.GenesisBlock().Hash)
	}
	return locator, nil
}

func (l *Ledger) FindLocator(ctx context.Context, hashes []string) (uint64, string, error) {
	if len(hashes) == 0 || len(hashes) > 64 {
		return 0, "", fmt.Errorf("locator must contain between 1 and 64 hashes")
	}
	for _, hash := range hashes {
		var height int64
		err := l.database.QueryRowContext(ctx, "SELECT height FROM blocks WHERE hash = ?", hash).Scan(&height)
		if err == nil {
			return uint64(height), hash, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, "", fmt.Errorf("find locator hash: %w", err)
		}
	}
	return 0, "", fmt.Errorf("locator does not share the Entropy genesis")
}

func (l *Ledger) WorkAt(ctx context.Context, height uint64) (*big.Int, error) {
	work, err := workAt(ctx, l.database, height)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBlockNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read cumulative work at %d: %w", height, err)
	}
	return new(big.Int).Set(work), nil
}

func scanHeader(scanner rowScanner) (core.Block, error) {
	var (
		height       int64
		hash         string
		previousHash string
		version      int64
		timestamp    int64
		merkleRoot   string
		difficulty   int64
		nonce        []byte
	)
	if err := scanner.Scan(&height, &hash, &previousHash, &version, &timestamp, &merkleRoot, &difficulty, &nonce); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.Block{}, ErrBlockNotFound
		}
		return core.Block{}, fmt.Errorf("scan block header: %w", err)
	}
	if height < 0 || version < 0 || version > math.MaxUint32 || difficulty < 0 || difficulty > math.MaxUint8 || len(nonce) != 8 {
		return core.Block{}, fmt.Errorf("stored block header contains invalid numeric values")
	}
	return core.Block{
		Version:      uint32(version),
		Height:       uint64(height),
		Timestamp:    timestamp,
		PreviousHash: previousHash,
		MerkleRoot:   merkleRoot,
		Difficulty:   uint8(difficulty),
		Nonce:        binary.BigEndian.Uint64(nonce),
		Hash:         hash,
	}, nil
}

func workAt(ctx context.Context, query interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, height uint64) (*big.Int, error) {
	if height > math.MaxInt64 {
		return nil, ErrBlockNotFound
	}
	var work []byte
	if err := query.QueryRowContext(ctx, "SELECT cumulative_work FROM blocks WHERE height = ?", int64(height)).Scan(&work); err != nil {
		return nil, err
	}
	return decodeWork(work), nil
}

func unixTime(value sql.NullInt64) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return time.Unix(value.Int64, 0)
}
