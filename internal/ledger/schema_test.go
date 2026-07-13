package ledger

import (
	"context"
	"testing"
)

func TestSchemaV1UpgradeCreatesCaseInsensitiveAddressIndexes(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	chain, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []string{
		"transaction_addresses_address_nocase",
		"utxos_address_nocase",
		"mempool_outputs_address_nocase",
	} {
		if _, err := chain.database.ExecContext(ctx, "DROP INDEX "+index); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := chain.database.ExecContext(ctx, "PRAGMA user_version = 1"); err != nil {
		t.Fatal(err)
	}
	if err := chain.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	var version int
	if err := upgraded.database.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, SchemaVersion)
	}
	for _, index := range []string{
		"transaction_addresses_address_nocase",
		"utxos_address_nocase",
		"mempool_outputs_address_nocase",
	} {
		var found string
		if err := upgraded.database.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type = 'index' AND name = ?", index).Scan(&found); err != nil {
			t.Fatalf("case-insensitive index %s was not migrated: %v", index, err)
		}
	}
}
