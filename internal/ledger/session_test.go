package ledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"entropy/internal/core"
)

func TestPublishedTestnetLedgerIsRejectedWithoutMutation(t *testing.T) {
	directory := t.TempDir()
	createPublishedTestnetLedger(t, directory)
	before := snapshotDirectory(t, directory)

	if opened, err := Open(context.Background(), directory); err == nil {
		_ = opened.Close()
		t.Fatal("published testnet ledger was opened as mainnet")
	} else if !strings.Contains(err.Error(), "entropy-testnet-v3") {
		t.Fatalf("testnet rejection error = %v", err)
	}
	if after := snapshotDirectory(t, directory); !reflect.DeepEqual(after, before) {
		t.Fatalf("rejected testnet directory changed:\nbefore %#v\nafter  %#v", before, after)
	}
}

func TestMainnetProtocolCannotHideForeignGenesis(t *testing.T) {
	directory := t.TempDir()
	path := createPublishedTestnetLedger(t, directory)
	updateStoredProtocol(t, path, ProtocolName)
	before := snapshotDirectory(t, directory)

	if opened, err := Open(context.Background(), directory); err == nil {
		_ = opened.Close()
		t.Fatal("testnet genesis with forged mainnet protocol was accepted")
	} else if !strings.Contains(err.Error(), "genesis") {
		t.Fatalf("foreign genesis rejection error = %v", err)
	}
	if after := snapshotDirectory(t, directory); !reflect.DeepEqual(after, before) {
		t.Fatalf("foreign-genesis database changed:\nbefore %#v\nafter  %#v", before, after)
	}
}

func TestNonMainnetProtocolsAreRejectedWithoutUpgrade(t *testing.T) {
	for _, protocol := range []string{"entropy-testnet-v2", "foreign-chain"} {
		t.Run(protocol, func(t *testing.T) {
			directory := t.TempDir()
			chain, err := Open(context.Background(), directory)
			if err != nil {
				t.Fatal(err)
			}
			path := chain.Path()
			if err := chain.Close(); err != nil {
				t.Fatal(err)
			}
			updateStoredProtocol(t, path, protocol)
			before := snapshotDirectory(t, directory)
			if opened, err := Open(context.Background(), directory); err == nil {
				_ = opened.Close()
				t.Fatalf("protocol %q was upgraded in place", protocol)
			}
			if after := snapshotDirectory(t, directory); !reflect.DeepEqual(after, before) {
				t.Fatalf("protocol %q rejection modified the database", protocol)
			}
		})
	}
}

func TestLegacyChainJSONIsRejectedBeforeDatabaseCreation(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "chain.json")
	contents := []byte(`{"version":1,"name":"Entropy","symbol":"ENT","blocks":[],"pending":[]}`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	before := snapshotDirectory(t, directory)
	if opened, err := Open(context.Background(), directory); err == nil {
		_ = opened.Close()
		t.Fatal("legacy chain.json was accepted by mainnet")
	} else if !strings.Contains(err.Error(), "chain.json") {
		t.Fatalf("legacy chain rejection error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, DatabaseName)); !os.IsNotExist(err) {
		t.Fatalf("mainnet database was created beside legacy chain: %v", err)
	}
	if after := snapshotDirectory(t, directory); !reflect.DeepEqual(after, before) {
		t.Fatalf("legacy directory changed:\nbefore %#v\nafter  %#v", before, after)
	}
}

func TestDirtySessionRecoversAndCleanClosePersists(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	crashed, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	assertShutdownState(t, crashed.database, 0)
	eventID, err := crashed.AddHealthEvent(ctx, HealthEvent{
		Code: "test.crash", Severity: "warning", Message: "committed before simulated crash",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Closing the raw pool skips Ledger.Close's clean marker and models an
	// abruptly terminated process while preserving SQLite's committed WAL.
	if err := crashed.database.Close(); err != nil {
		t.Fatal(err)
	}

	recovered, err := Open(ctx, directory)
	if err != nil {
		t.Fatalf("recover dirty ledger: %v", err)
	}
	assertShutdownState(t, recovered.database, 0)
	events, err := recovered.HealthEvents(ctx, false, 10)
	if err != nil || len(events) != 1 || events[0].ID != eventID {
		t.Fatalf("WAL event after recovery = %#v, err %v", events, err)
	}
	if err := recovered.Close(); err != nil {
		t.Fatalf("clean close: %v", err)
	}

	raw, err := sql.Open("sqlite", sqliteDSN(recovered.Path()))
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	assertShutdownState(t, raw, 1)
}

func createPublishedTestnetLedger(t *testing.T, directory string) string {
	t.Helper()
	chain, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	path := chain.Path()
	if err := chain.Close(); err != nil {
		t.Fatal(err)
	}

	genesis := core.Block{
		Version: core.StateVersion, Height: 0, Timestamp: 1783900800,
		PreviousHash: strings.Repeat("0", 64), MerkleRoot: core.MerkleRoot(nil),
		Difficulty: 0, Transactions: []core.Transaction{},
	}
	genesis.Hash = genesis.ComputeHash()
	data, err := json.Marshal(genesis)
	if err != nil {
		t.Fatal(err)
	}
	raw := openWritableTestDatabase(t, path)
	tx, err := raw.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		UPDATE blocks SET hash = ?, timestamp = ?, data = ?, encoded_size = ? WHERE height = 0
	`, genesis.Hash, genesis.Timestamp, data, len(data)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("UPDATE meta SET value = ? WHERE key = 'protocol'", "entropy-testnet-v3"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("UPDATE meta SET value = ? WHERE key = 'genesis_hash'", genesis.Hash); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	checkpointAndCloseTestDatabase(t, raw)
	return path
}

func updateStoredProtocol(t *testing.T, path, protocol string) {
	t.Helper()
	raw := openWritableTestDatabase(t, path)
	if _, err := raw.Exec("UPDATE meta SET value = ? WHERE key = 'protocol'", protocol); err != nil {
		t.Fatal(err)
	}
	checkpointAndCloseTestDatabase(t, raw)
}

func openWritableTestDatabase(t *testing.T, path string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Ping(); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	return database
}

func checkpointAndCloseTestDatabase(t *testing.T, database *sql.DB) {
	t.Helper()
	var busy, logFrames, checkpointedFrames int
	if err := database.QueryRow("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointedFrames); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	if busy != 0 {
		_ = database.Close()
		t.Fatalf("test checkpoint remained busy: %d/%d", checkpointedFrames, logFrames)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
}

func snapshotDirectory(t *testing.T, directory string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := make(map[string]string, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			snapshot[entry.Name()] = entry.Type().String()
			continue
		}
		contents, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(contents)
		snapshot[entry.Name()] = hex.EncodeToString(digest[:])
	}
	return snapshot
}

func assertShutdownState(t *testing.T, database interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, want byte) {
	t.Helper()
	var state []byte
	if err := database.QueryRowContext(context.Background(), "SELECT value FROM meta WHERE key = ?", shutdownStateMetaKey).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if len(state) != 1 || state[0] != want {
		t.Fatalf("shutdown state = %v, want [%d]", state, want)
	}
}
