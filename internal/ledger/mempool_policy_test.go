package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/HONG-LOU/entcoin/internal/core"
)

func TestMempoolResourceBudgetsAndMinimumRelayFee(t *testing.T) {
	if err := validateMempoolResourceBudget(0, MaxMempoolBytes-100, 0, 100, 1); err != nil {
		t.Fatalf("exact byte budget was rejected: %v", err)
	}
	if err := validateMempoolResourceBudget(0, MaxMempoolBytes-100, 0, 101, 1); err == nil {
		t.Fatal("mempool byte budget overflow was accepted")
	}
	if err := validateMempoolResourceBudget(0, 0, MaxMempoolInputs-1, 100, 2); err == nil {
		t.Fatal("mempool input budget overflow was accepted")
	}
	if fee := minimumRelayFee(1); fee != 1_000 {
		t.Fatalf("one-byte relay fee = %d, want 1000", fee)
	}
	if fee := minimumRelayFee(1025); fee != 2_000 {
		t.Fatalf("1025-byte relay fee = %d, want 2000", fee)
	}
	if fee := minimumRelayFee(core.MaxTransactionBytes); fee > 100_000 {
		t.Fatalf("maximum-size transaction relay fee = %d, exceeds the 0.001 ENT UI default", fee)
	}

	ctx := context.Background()
	chain, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Close()
	owner := newTestWallet(t)
	recipient := newTestWallet(t)
	outpoint := core.Outpoint{TxID: strings.Repeat("e", 64), Index: 0}
	const amount uint64 = 1_000_000
	if _, err := chain.database.ExecContext(ctx, `
		INSERT INTO utxos(tx_id, output_index, amount, address, created_height, coinbase)
		VALUES(?, 0, ?, ?, 1, 0)
	`, outpoint.TxID, int64(amount), owner.Address); err != nil {
		t.Fatal(err)
	}
	zeroFee, err := core.BuildTransaction(owner, recipient.Address, amount, 0, core.UTXO{
		outpoint: {Amount: amount, Address: owner.Address},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.AddTransaction(ctx, zeroFee); err == nil {
		t.Fatal("zero-fee transaction entered the relay mempool")
	}
	minimumFee := minimumRelayFee(core.EncodedTransactionSize(zeroFee))
	belowMinimum, err := core.BuildTransaction(owner, recipient.Address, amount-(minimumFee-1), minimumFee-1, core.UTXO{
		outpoint: {Amount: amount, Address: owner.Address},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.AddTransaction(ctx, belowMinimum); err == nil {
		t.Fatal("below-minimum-fee transaction entered the relay mempool")
	}
	withFee, err := core.BuildTransaction(owner, recipient.Address, amount-1, 1, core.UTXO{
		outpoint: {Amount: amount, Address: owner.Address},
	})
	if err != nil {
		t.Fatal(err)
	}
	minimumFee = minimumRelayFee(core.EncodedTransactionSize(withFee))
	withFee, err = core.BuildTransaction(owner, recipient.Address, amount-minimumFee, minimumFee, core.UTXO{
		outpoint: {Amount: amount, Address: owner.Address},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.AddTransaction(ctx, withFee); err != nil {
		t.Fatalf("minimum-fee transaction was rejected: %v", err)
	}
}

type countingMempoolQueryer struct {
	*sql.Tx
	aggregateReads int
}

func (q *countingMempoolQueryer) QueryRowContext(ctx context.Context, query string, arguments ...any) *sql.Row {
	if strings.Contains(query, "SUM(encoded_size)") {
		q.aggregateReads++
	}
	return q.Tx.QueryRowContext(ctx, query, arguments...)
}

func TestRepopulateMempoolUsesIncrementalResourceAccounting(t *testing.T) {
	ctx := context.Background()
	chain, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Close()
	owner := newTestWallet(t)
	recipient := newTestWallet(t)
	databaseTx, err := chain.database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer databaseTx.Rollback()
	pending := buildPolicyTestTransactions(t, databaseTx, owner, recipient, 256)
	queryer := &countingMempoolQueryer{Tx: databaseTx}
	if err := repopulateMempool(ctx, queryer, pending, 1); err != nil {
		t.Fatal(err)
	}
	if queryer.aggregateReads != 1 {
		t.Fatalf("256-transaction repopulation aggregate queries = %d, want 1", queryer.aggregateReads)
	}
	var count, totalInputs int
	if err := databaseTx.QueryRowContext(ctx, `
		SELECT COUNT(*), (SELECT COUNT(*) FROM mempool_inputs) FROM mempool
	`).Scan(&count, &totalInputs); err != nil {
		t.Fatal(err)
	}
	if count != len(pending) || totalInputs != len(pending) {
		t.Fatalf("repopulated mempool count/inputs = %d/%d, want %d/%d", count, totalInputs, len(pending), len(pending))
	}
}

func BenchmarkRepopulateMempoolAtPolicyTransactionLimit(b *testing.B) {
	ctx := context.Background()
	chain, err := Open(ctx, b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = chain.Close() })
	owner := newPolicyTestWallet(b)
	recipient := newPolicyTestWallet(b)
	setupTx, err := chain.database.BeginTx(ctx, nil)
	if err != nil {
		b.Fatal(err)
	}
	pending := buildPolicyTestTransactions(b, setupTx, owner, recipient, core.MaxPendingTransactions)
	if err := setupTx.Commit(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for range b.N {
		databaseTx, err := chain.database.BeginTx(ctx, nil)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := databaseTx.ExecContext(ctx, "DELETE FROM mempool"); err != nil {
			_ = databaseTx.Rollback()
			b.Fatal(err)
		}
		queryer := &countingMempoolQueryer{Tx: databaseTx}
		if err := repopulateMempool(ctx, queryer, pending, 1); err != nil {
			_ = databaseTx.Rollback()
			b.Fatal(err)
		}
		if queryer.aggregateReads != 1 {
			_ = databaseTx.Rollback()
			b.Fatalf("aggregate resource queries = %d, want 1", queryer.aggregateReads)
		}
		var count int
		if err := databaseTx.QueryRowContext(ctx, "SELECT COUNT(*) FROM mempool").Scan(&count); err != nil {
			_ = databaseTx.Rollback()
			b.Fatal(err)
		}
		if count != core.MaxPendingTransactions {
			_ = databaseTx.Rollback()
			b.Fatalf("repopulated transactions = %d, want %d", count, core.MaxPendingTransactions)
		}
		if err := databaseTx.Rollback(); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(core.MaxPendingTransactions), "tx/rebuild")
}

func buildPolicyTestTransactions(
	t testing.TB,
	databaseTx *sql.Tx,
	owner, recipient *core.Wallet,
	count int,
) []pendingMempoolTransaction {
	t.Helper()
	const (
		amount = uint64(1_000_000)
		fee    = uint64(100_000)
	)
	pending := make([]pendingMempoolTransaction, 0, count)
	for index := range count {
		outpoint := core.Outpoint{TxID: fmt.Sprintf("%064x", index+10_000), Index: 0}
		if _, err := databaseTx.Exec(`
			INSERT INTO utxos(tx_id, output_index, amount, address, created_height, coinbase)
			VALUES(?, 0, ?, ?, 0, 0)
		`, outpoint.TxID, int64(amount), owner.Address); err != nil {
			t.Fatal(err)
		}
		transaction, err := core.BuildTransaction(owner, recipient.Address, amount-fee, fee, core.UTXO{
			outpoint: {Amount: amount, Address: owner.Address},
		})
		if err != nil {
			t.Fatal(err)
		}
		pending = append(pending, pendingMempoolTransaction{transaction: transaction, firstSeen: time.Now().Unix()})
	}
	return pending
}

func newPolicyTestWallet(t testing.TB) *core.Wallet {
	t.Helper()
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	return wallet
}
