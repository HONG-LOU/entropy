package ledger

import (
	"context"
	"database/sql"
	"fmt"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS meta (
    key TEXT PRIMARY KEY,
    value BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS blocks (
    height INTEGER PRIMARY KEY,
    hash TEXT NOT NULL UNIQUE,
    previous_hash TEXT NOT NULL,
	version INTEGER NOT NULL,
    timestamp INTEGER NOT NULL,
	merkle_root TEXT NOT NULL,
    difficulty INTEGER NOT NULL,
	nonce BLOB NOT NULL CHECK(length(nonce) = 8),
    cumulative_work BLOB NOT NULL,
    data BLOB,
    encoded_size INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS blocks_hash_height ON blocks(hash, height);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    block_height INTEGER NOT NULL,
    tx_index INTEGER NOT NULL,
    coinbase INTEGER NOT NULL,
	data BLOB,
    FOREIGN KEY(block_height) REFERENCES blocks(height) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS transactions_height_index ON transactions(block_height, tx_index);

CREATE TABLE IF NOT EXISTS transaction_addresses (
    tx_id TEXT NOT NULL,
    ordinal INTEGER NOT NULL,
    address TEXT NOT NULL,
    direction INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    PRIMARY KEY(tx_id, ordinal, address, direction),
    FOREIGN KEY(tx_id) REFERENCES transactions(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS transaction_addresses_address ON transaction_addresses(address, tx_id);
CREATE INDEX IF NOT EXISTS transaction_addresses_address_nocase ON transaction_addresses(address COLLATE NOCASE, tx_id);

CREATE TABLE IF NOT EXISTS utxos (
    tx_id TEXT NOT NULL,
    output_index INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    address TEXT NOT NULL,
    created_height INTEGER NOT NULL,
    coinbase INTEGER NOT NULL,
    PRIMARY KEY(tx_id, output_index)
);
CREATE INDEX IF NOT EXISTS utxos_address ON utxos(address, tx_id, output_index);
CREATE INDEX IF NOT EXISTS utxos_address_nocase ON utxos(address COLLATE NOCASE, tx_id, output_index);

CREATE TABLE IF NOT EXISTS block_undo (
    block_height INTEGER PRIMARY KEY,
    block_hash TEXT NOT NULL UNIQUE,
    data BLOB NOT NULL,
    FOREIGN KEY(block_height) REFERENCES blocks(height) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS mempool (
    sequence INTEGER PRIMARY KEY AUTOINCREMENT,
    tx_id TEXT NOT NULL UNIQUE,
    first_seen INTEGER NOT NULL,
    fee INTEGER NOT NULL,
    encoded_size INTEGER NOT NULL,
    data BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS mempool_inputs (
    tx_id TEXT NOT NULL,
    input_tx_id TEXT NOT NULL,
    input_index INTEGER NOT NULL,
    PRIMARY KEY(tx_id, input_tx_id, input_index),
    UNIQUE(input_tx_id, input_index),
    FOREIGN KEY(tx_id) REFERENCES mempool(tx_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS mempool_outputs (
    tx_id TEXT NOT NULL,
    output_index INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    address TEXT NOT NULL,
    PRIMARY KEY(tx_id, output_index),
    FOREIGN KEY(tx_id) REFERENCES mempool(tx_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS mempool_outputs_address ON mempool_outputs(address, tx_id, output_index);
CREATE INDEX IF NOT EXISTS mempool_outputs_address_nocase ON mempool_outputs(address COLLATE NOCASE, tx_id, output_index);

CREATE TABLE IF NOT EXISTS peers (
    url TEXT PRIMARY KEY,
    manual INTEGER NOT NULL DEFAULT 0,
    added_at INTEGER NOT NULL,
    last_seen INTEGER,
    failures INTEGER NOT NULL DEFAULT 0,
    next_attempt INTEGER,
    last_error TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS health_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL,
    severity TEXT NOT NULL,
    message TEXT NOT NULL,
    action TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    resolved INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS health_events_active ON health_events(resolved, created_at DESC);
`

const schemaV2SQL = `
CREATE INDEX IF NOT EXISTS transaction_addresses_address_nocase ON transaction_addresses(address COLLATE NOCASE, tx_id);
CREATE INDEX IF NOT EXISTS utxos_address_nocase ON utxos(address COLLATE NOCASE, tx_id, output_index);
CREATE INDEX IF NOT EXISTS mempool_outputs_address_nocase ON mempool_outputs(address COLLATE NOCASE, tx_id, output_index);
`

func migrateSchema(ctx context.Context, database *sql.DB) error {
	var version int
	if err := database.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read database schema version: %w", err)
	}
	if version > SchemaVersion {
		return fmt.Errorf("database schema %d is newer than supported version %d", version, SchemaVersion)
	}
	if version == SchemaVersion {
		return nil
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if version == 0 {
		if _, err := tx.ExecContext(ctx, schemaSQL); err != nil {
			return fmt.Errorf("create database schema: %w", err)
		}
	}
	if version < 2 {
		if _, err := tx.ExecContext(ctx, schemaV2SQL); err != nil {
			return fmt.Errorf("create case-insensitive address indexes: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", SchemaVersion)); err != nil {
		return fmt.Errorf("record database schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema migration: %w", err)
	}
	return nil
}
