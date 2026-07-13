package ledger

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"entropy/internal/core"
)

func TestProtocolMetadataUpgradePreservesLedgerState(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	ledger, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	wallet := newTestWallet(t)
	insertSyntheticHeaders(t, ledger, 1)
	if _, err := ledger.database.ExecContext(ctx, `
		INSERT INTO utxos(tx_id, output_index, amount, address, created_height, coinbase)
		VALUES(?, 0, 12345, ?, 1, 1)
	`, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", wallet.Address); err != nil {
		t.Fatal(err)
	}
	beforeTip, err := ledger.Tip(ctx)
	if err != nil {
		t.Fatal(err)
	}
	beforeConfirmed, beforeSpendable, err := ledger.Balances(ctx, wallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.database.ExecContext(ctx, "UPDATE meta SET value = ? WHERE key = 'protocol'", "entropy-testnet-v2"); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := Open(ctx, directory)
	if err != nil {
		t.Fatalf("open v2 ledger for known upgrade: %v", err)
	}
	defer upgraded.Close()
	var protocol string
	if err := upgraded.database.QueryRowContext(ctx, "SELECT value FROM meta WHERE key = 'protocol'").Scan(&protocol); err != nil {
		t.Fatal(err)
	}
	if protocol != ProtocolName {
		t.Fatalf("upgraded protocol = %q, want %q", protocol, ProtocolName)
	}
	afterTip, err := upgraded.Tip(ctx)
	if err != nil || afterTip.Height != beforeTip.Height || afterTip.Hash != beforeTip.Hash || afterTip.Work.Cmp(beforeTip.Work) != 0 {
		t.Fatalf("tip changed across protocol upgrade: before %#v after %#v err %v", beforeTip, afterTip, err)
	}
	afterConfirmed, afterSpendable, err := upgraded.Balances(ctx, wallet.Address)
	if err != nil || afterConfirmed != beforeConfirmed || afterSpendable != beforeSpendable {
		t.Fatalf("balances changed across protocol upgrade: before %d/%d after %d/%d err %v",
			beforeConfirmed, beforeSpendable, afterConfirmed, afterSpendable, err)
	}
}

func TestProtocolUpgradeRevalidatesMempoolForNextHeight(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	chain, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	owner := newTestWallet(t)
	recipient := newTestWallet(t)
	insertSyntheticHeaders(t, chain, core.CoinbaseMaturityActivationHeight-1)
	outpoint := core.Outpoint{TxID: strings.Repeat("d", 64), Index: 0}
	amount := uint64(10_000)
	if _, err := chain.database.ExecContext(ctx, `
		INSERT INTO utxos(tx_id, output_index, amount, address, created_height, coinbase)
		VALUES(?, 0, ?, ?, ?, 1)
	`, outpoint.TxID, int64(amount), owner.Address, int64(core.CoinbaseMaturityActivationHeight-1)); err != nil {
		t.Fatal(err)
	}
	transaction, err := core.BuildTransaction(owner, recipient.Address, amount, 0, core.UTXO{
		outpoint: {Amount: amount, Address: owner.Address},
	})
	if err != nil {
		t.Fatal(err)
	}
	insertRawMempoolTransaction(t, chain, transaction)
	if _, err := chain.database.ExecContext(ctx, "UPDATE meta SET value = ? WHERE key = 'protocol'", "entropy-testnet-v2"); err != nil {
		t.Fatal(err)
	}
	if err := chain.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := Open(ctx, directory)
	if err != nil {
		t.Fatalf("open height-99 v2 ledger: %v", err)
	}
	defer upgraded.Close()
	if count, err := upgraded.MempoolCount(ctx); err != nil || count != 0 {
		t.Fatalf("upgrade retained next-height immature spend: count=%d err=%v", count, err)
	}
	if !mempoolMaturityRulesActive(core.CoinbaseMaturityActivationHeight - 1) {
		t.Fatal("next-height maturity activation was not detected")
	}
}

func TestUnknownProtocolMetadataIsRejected(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	ledger, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.database.ExecContext(ctx, "UPDATE meta SET value = 'foreign-chain' WHERE key = 'protocol'"); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(ctx, directory); err == nil {
		t.Fatal("unknown ledger protocol was accepted")
	}
}

func TestV2HighChainRequiresSafeReplayOrResync(t *testing.T) {
	ctx := context.Background()
	archiveDirectory := t.TempDir()
	archive, err := Open(ctx, archiveDirectory)
	if err != nil {
		t.Fatal(err)
	}
	insertSyntheticHeaders(t, archive, core.CoinbaseMaturityActivationHeight)
	if _, err := archive.database.ExecContext(ctx, "UPDATE meta SET value = 'entropy-testnet-v2' WHERE key = 'protocol'"); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(ctx, archiveDirectory); err == nil || !strings.Contains(err.Error(), "resync") {
		t.Fatalf("invalid archived v2 chain upgrade error = %v", err)
	}

	prunedDirectory := t.TempDir()
	pruned, err := Open(ctx, prunedDirectory)
	if err != nil {
		t.Fatal(err)
	}
	insertSyntheticHeaders(t, pruned, core.CoinbaseMaturityActivationHeight)
	if _, err := pruned.Prune(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := pruned.database.ExecContext(ctx, "UPDATE meta SET value = 'entropy-testnet-v2' WHERE key = 'protocol'"); err != nil {
		t.Fatal(err)
	}
	if err := pruned.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(ctx, prunedDirectory); err == nil || !strings.Contains(err.Error(), "resync") {
		t.Fatalf("pruned v2 chain upgrade error = %v", err)
	}
}

func TestV3ReplayAuditDoesNotTrustStoredUTXO(t *testing.T) {
	ctx := context.Background()
	ledger, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	wallet := newTestWallet(t)
	block, expectedTip := mineLedgerCandidate(t, ledger, wallet.Address)
	if err := ledger.CommitMinedBlock(ctx, block, expectedTip); err != nil {
		t.Fatal(err)
	}
	tip, err := ledger.Tip(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.validateStoredChainForV3(ctx, tip); err != nil {
		t.Fatalf("valid archive replay audit: %v", err)
	}
	if _, err := ledger.database.ExecContext(ctx, "UPDATE utxos SET amount = amount + 1"); err != nil {
		t.Fatal(err)
	}
	if err := ledger.validateStoredChainForV3(ctx, tip); err == nil {
		t.Fatal("replay audit trusted a corrupted stored UTXO")
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
