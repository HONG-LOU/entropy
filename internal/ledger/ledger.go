package ledger

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"entropy/internal/core"
	_ "modernc.org/sqlite"
)

type Ledger struct {
	database *sql.DB
	path     string
	writeMu  sync.Mutex
	closed   bool
}

const shutdownStateMetaKey = "clean_shutdown"

func Open(ctx context.Context, directory string) (*Ledger, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create ledger directory: %w", err)
	}
	path := filepath.Join(directory, DatabaseName)
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("ledger database path must not be a symbolic link")
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("ledger database path is not a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect ledger database path: %w", err)
	}
	dsn := sqliteDSN(path)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open ledger database: %w", err)
	}
	database.SetMaxOpenConns(4)
	database.SetMaxIdleConns(4)
	database.SetConnMaxLifetime(0)
	ledger := &Ledger{database: database, path: path}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = database.Close()
		}
	}()
	if err := database.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect ledger database: %w", err)
	}
	if err := migrateSchema(ctx, database); err != nil {
		return nil, err
	}
	if err := ledger.ensureGenesis(ctx); err != nil {
		return nil, err
	}
	if err := ledger.prepareSession(ctx); err != nil {
		return nil, err
	}
	closeOnError = false
	return ledger, nil
}

func sqliteDSN(path string) string {
	urlPath := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" && !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	fileURL := &url.URL{Scheme: "file", Path: urlPath}
	query := fileURL.Query()
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(FULL)")
	query.Add("_pragma", "foreign_keys(ON)")
	query.Add("_pragma", "temp_store(MEMORY)")
	query.Add("_pragma", "trusted_schema(OFF)")
	fileURL.RawQuery = query.Encode()
	return fileURL.String()
}

func (l *Ledger) Close() error {
	if l == nil || l.database == nil {
		return nil
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	checkpointErr := l.checkpoint()
	if checkpointErr != nil {
		return errors.Join(checkpointErr, l.database.Close())
	}
	cleanErr := l.setShutdownState(context.Background(), true)
	if cleanErr != nil {
		return errors.Join(cleanErr, l.database.Close())
	}
	finalCheckpointErr := l.checkpoint()
	if finalCheckpointErr != nil {
		_ = l.setShutdownState(context.Background(), false)
	}
	return errors.Join(finalCheckpointErr, l.database.Close())
}

func (l *Ledger) Path() string {
	return l.path
}

func (l *Ledger) quickCheck(ctx context.Context) error {
	var result string
	if err := l.database.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result); err != nil {
		return fmt.Errorf("check ledger database: %w", err)
	}
	if !strings.EqualFold(result, "ok") {
		return fmt.Errorf("ledger database integrity check failed: %s", result)
	}
	return nil
}

func (l *Ledger) prepareSession(ctx context.Context) error {
	clean, known, err := l.shutdownState(ctx)
	if err != nil {
		return err
	}
	if !known || !clean {
		if err := l.quickCheck(ctx); err != nil {
			return err
		}
	}
	if err := l.setShutdownState(ctx, false); err != nil {
		return fmt.Errorf("mark ledger session dirty: %w", err)
	}
	return nil
}

func (l *Ledger) shutdownState(ctx context.Context) (bool, bool, error) {
	var value []byte
	err := l.database.QueryRowContext(ctx, "SELECT value FROM meta WHERE key = ?", shutdownStateMetaKey).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("read ledger shutdown state: %w", err)
	}
	if len(value) != 1 || (value[0] != 0 && value[0] != 1) {
		return false, false, fmt.Errorf("stored ledger shutdown state is invalid")
	}
	return value[0] == 1, true, nil
}

func (l *Ledger) setShutdownState(ctx context.Context, clean bool) error {
	value := byte(0)
	if clean {
		value = 1
	}
	_, err := l.database.ExecContext(ctx, `
		INSERT INTO meta(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, shutdownStateMetaKey, []byte{value})
	return err
}

func (l *Ledger) checkpoint() error {
	var busy, logFrames, checkpointedFrames int
	if err := l.database.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointedFrames); err != nil {
		return fmt.Errorf("checkpoint ledger WAL: %w", err)
	}
	if busy != 0 {
		return fmt.Errorf("checkpoint ledger WAL remained busy with %d frames", logFrames-checkpointedFrames)
	}
	return nil
}

func (l *Ledger) ensureGenesis(ctx context.Context) error {
	var count int
	if err := l.database.QueryRowContext(ctx, "SELECT COUNT(*) FROM blocks").Scan(&count); err != nil {
		return fmt.Errorf("count ledger blocks: %w", err)
	}
	if count > 0 {
		var hash string
		if err := l.database.QueryRowContext(ctx, "SELECT hash FROM blocks WHERE height = 0").Scan(&hash); err != nil {
			return fmt.Errorf("read stored genesis: %w", err)
		}
		if hash != core.GenesisBlock().Hash {
			return fmt.Errorf("stored genesis does not match %s", ProtocolName)
		}
		return l.ensureProtocolMetadata(ctx)
	}
	genesis := core.GenesisBlock()
	data, err := encodeJSON(genesis)
	if err != nil {
		return fmt.Errorf("encode genesis block: %w", err)
	}
	tx, err := l.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
        INSERT INTO blocks(height, hash, previous_hash, version, timestamp, merkle_root, difficulty, nonce, cumulative_work, data, encoded_size)
        VALUES(0, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)
	`, genesis.Hash, genesis.PreviousHash, genesis.Version, genesis.Timestamp, genesis.MerkleRoot, encodeUint64(genesis.Nonce), encodeWork(new(big.Int)), data, len(data)); err != nil {
		return fmt.Errorf("insert genesis block: %w", err)
	}
	metadata := map[string][]byte{
		"protocol":     []byte(ProtocolName),
		"genesis_hash": []byte(genesis.Hash),
		"created_at":   encodeUint64(uint64(time.Now().Unix())),
	}
	for key, value := range metadata {
		if _, err := tx.ExecContext(ctx, "INSERT INTO meta(key, value) VALUES(?, ?)", key, value); err != nil {
			return fmt.Errorf("insert ledger metadata %s: %w", key, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit genesis block: %w", err)
	}
	return nil
}

func (l *Ledger) ensureProtocolMetadata(ctx context.Context) error {
	var protocol string
	if err := l.database.QueryRowContext(ctx, "SELECT value FROM meta WHERE key = 'protocol'").Scan(&protocol); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("ledger protocol metadata is missing")
		}
		return fmt.Errorf("read ledger protocol metadata: %w", err)
	}
	if protocol == ProtocolName {
		return nil
	}
	if protocol != "entropy-testnet-v2" {
		return fmt.Errorf("ledger protocol %q is not supported by %s", protocol, ProtocolName)
	}
	return l.upgradeV2Protocol(ctx)
}

func encodeWork(work *big.Int) []byte {
	if work == nil || work.Sign() == 0 {
		return []byte{0}
	}
	return work.Bytes()
}

func decodeWork(value []byte) *big.Int {
	return new(big.Int).SetBytes(value)
}

func blockWork(difficulty uint8) *big.Int {
	return new(big.Int).Lsh(big.NewInt(1), uint(difficulty))
}

func encodeUint64(value uint64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, value)
	return encoded
}
