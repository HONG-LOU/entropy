package ledger

import (
	"context"
	"fmt"
	"testing"

	"entropy/internal/core"
)

func TestMiningCandidatePrioritizesHigherFeeRate(t *testing.T) {
	ctx := context.Background()
	chain, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Close()
	lowFeeOwner := newTestWallet(t)
	highFeeOwner := newTestWallet(t)
	recipient := newTestWallet(t)
	miner := newTestWallet(t)
	const amount uint64 = 1_000_000
	owners := []*core.Wallet{lowFeeOwner, highFeeOwner}
	fees := []uint64{1_000, 10_000}
	transactions := make([]core.Transaction, 0, len(owners))
	for index, owner := range owners {
		outpoint := core.Outpoint{TxID: fmt.Sprintf("%064x", index+1_000), Index: 0}
		if _, err := chain.database.ExecContext(ctx, `
			INSERT INTO utxos(tx_id, output_index, amount, address, created_height, coinbase)
			VALUES(?, 0, ?, ?, 0, 0)
		`, outpoint.TxID, int64(amount), owner.Address); err != nil {
			t.Fatal(err)
		}
		transaction, err := core.BuildTransaction(owner, recipient.Address, amount-fees[index], fees[index], core.UTXO{
			outpoint: {Amount: amount, Address: owner.Address},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := chain.AddTransaction(ctx, transaction); err != nil {
			t.Fatal(err)
		}
		transactions = append(transactions, transaction)
	}

	candidate, _, err := chain.BuildMiningCandidate(ctx, miner.Address)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Transactions) != 3 || candidate.Transactions[1].ID != transactions[1].ID || candidate.Transactions[2].ID != transactions[0].ID {
		t.Fatalf("candidate transaction order = %#v", candidate.Transactions)
	}
}

func TestMiningPriorityUsesFeeRateWithoutBreakingDependencies(t *testing.T) {
	items := []miningMempoolItem{
		{transaction: core.Transaction{ID: "parent"}, fee: 1_000, encodedSize: 1_000, sequence: 1},
		{
			transaction: core.Transaction{ID: "child", Inputs: []core.TxInput{{TxID: "parent"}}},
			fee:         9_000, encodedSize: 1_000, sequence: 2,
		},
		{transaction: core.Transaction{ID: "medium"}, fee: 5_000, encodedSize: 1_000, sequence: 3},
		{transaction: core.Transaction{ID: "highest"}, fee: 6_000, encodedSize: 1_000, sequence: 4},
		{transaction: core.Transaction{ID: "lowest"}, fee: 500, encodedSize: 1_000, sequence: 5},
	}

	ordered, err := prioritizeMempoolForMining(items)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"highest", "medium", "parent", "child", "lowest"}
	if len(ordered) != len(want) {
		t.Fatalf("ordered transaction count = %d, want %d", len(ordered), len(want))
	}
	for index, id := range want {
		if ordered[index].ID != id {
			t.Fatalf("ordered transaction %d = %q, want %q", index, ordered[index].ID, id)
		}
	}
}

func TestMiningPriorityRejectsDependencyCycle(t *testing.T) {
	items := []miningMempoolItem{
		{
			transaction: core.Transaction{ID: "left", Inputs: []core.TxInput{{TxID: "right"}}},
			fee:         1, encodedSize: 1, sequence: 1,
		},
		{
			transaction: core.Transaction{ID: "right", Inputs: []core.TxInput{{TxID: "left"}}},
			fee:         1, encodedSize: 1, sequence: 2,
		},
	}
	if _, err := prioritizeMempoolForMining(items); err == nil {
		t.Fatal("cyclic mining mempool was accepted")
	}
}
