package ledger

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/HONG-LOU/entcoin/internal/core"
)

func TestCoinbaseMaturityAcrossMempoolMiningAndHeightRollback(t *testing.T) {
	ctx := context.Background()
	ledger, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	owner := newTestWallet(t)
	recipient := newTestWallet(t)
	outpoint := core.Outpoint{TxID: strings.Repeat("b", 64), Index: 0}
	const sourceAmount uint64 = 100_000
	if _, err := ledger.database.ExecContext(ctx, `
		INSERT INTO utxos(tx_id, output_index, amount, address, created_height, coinbase)
		VALUES(?, 0, ?, ?, 1, 1)
	`, outpoint.TxID, int64(sourceAmount), owner.Address); err != nil {
		t.Fatal(err)
	}
	transaction, err := core.BuildTransaction(owner, recipient.Address, 50_000, 1_000, core.UTXO{
		outpoint: {Amount: sourceAmount, Address: owner.Address},
	})
	if err != nil {
		t.Fatal(err)
	}
	insertSyntheticHeaders(t, ledger, 99)
	confirmed, spendable, err := ledger.Balances(ctx, owner.Address)
	if err != nil || confirmed != sourceAmount || spendable != 0 {
		t.Fatalf("immature balances = %d/%d, err %v", confirmed, spendable, err)
	}
	if err := ledger.AddTransaction(ctx, transaction); err == nil {
		t.Fatal("mempool accepted an immature mainnet coinbase spend")
	}
	if err := validateInputMaturity([]UTXORecord{{
		TxID: outpoint.TxID, OutputIndex: 0, Amount: sourceAmount,
		Address: owner.Address, CreatedHeight: 1, Coinbase: true,
	}}, 100); err == nil {
		t.Fatal("block validation accepted a 99-block-old coinbase")
	}
	insertRawMempoolTransaction(t, ledger, transaction)
	candidate, _, err := ledger.BuildMiningCandidate(ctx, owner.Address)
	if err != nil {
		t.Fatalf("build activation candidate: %v", err)
	}
	if len(candidate.Transactions) != 1 {
		t.Fatalf("mining selected immature coinbase spend: %d transactions", len(candidate.Transactions))
	}
	if _, err := ledger.database.ExecContext(ctx, "DELETE FROM mempool"); err != nil {
		t.Fatal(err)
	}

	insertSyntheticHeaders(t, ledger, 100)
	if err := validateInputMaturity([]UTXORecord{{CreatedHeight: 1, Coinbase: true}}, 101); err != nil {
		t.Fatalf("block validation rejected exactly mature coinbase: %v", err)
	}
	if err := ledger.AddTransaction(ctx, transaction); err != nil {
		t.Fatalf("mempool rejected exactly mature coinbase: %v", err)
	}
	confirmed, spendable, err = ledger.Balances(ctx, owner.Address)
	if err != nil || confirmed != sourceAmount || spendable != sourceAmount-50_000-1_000 {
		t.Fatalf("mature balances = %d/%d, err %v", confirmed, spendable, err)
	}
	if _, err := ledger.database.ExecContext(ctx, "DELETE FROM blocks WHERE height = 100"); err != nil {
		t.Fatal(err)
	}
	rebuildMempoolForTest(t, ledger)
	if count, err := ledger.MempoolCount(ctx); err != nil || count != 0 {
		t.Fatalf("height rollback retained newly immature transaction: %d, err %v", count, err)
	}
}

func insertSyntheticHeaders(t *testing.T, ledger *Ledger, through uint64) {
	t.Helper()
	ctx := context.Background()
	tip, err := ledger.Tip(ctx)
	if err != nil {
		t.Fatal(err)
	}
	previousHash := tip.Hash
	for height := tip.Height + 1; height <= through; height++ {
		hash := fmt.Sprintf("%064x", height+1)
		timestamp := time.Now().Unix() - 1_000 + int64(height)*core.TargetBlockSeconds
		body, err := encodeJSON(core.Block{
			Version: core.StateVersion, Height: height, Timestamp: timestamp,
			PreviousHash: previousHash, MerkleRoot: strings.Repeat("0", 64),
			Difficulty: core.InitialDifficulty, Hash: hash, Transactions: []core.Transaction{},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ledger.database.ExecContext(ctx, `
			INSERT INTO blocks(
				height, hash, previous_hash, version, timestamp, merkle_root,
				difficulty, nonce, cumulative_work, data, encoded_size
			) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, int64(height), hash, previousHash, int64(core.StateVersion),
			timestamp,
			strings.Repeat("0", 64), int64(core.InitialDifficulty), encodeUint64(0),
			encodeWork(new(big.Int).Lsh(big.NewInt(1), uint(height))), body, len(body)); err != nil {
			t.Fatalf("insert synthetic header %d: %v", height, err)
		}
		previousHash = hash
	}
}

func rebuildMempoolForTest(t *testing.T, ledger *Ledger) {
	t.Helper()
	tx, err := ledger.database.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := rebuildMempool(context.Background(), tx, nil); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func insertRawMempoolTransaction(t *testing.T, ledger *Ledger, transaction core.Transaction) {
	t.Helper()
	data, err := encodeJSON(transaction)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.database.ExecContext(context.Background(), `
		INSERT INTO mempool(tx_id, first_seen, fee, encoded_size, data)
		VALUES(?, ?, 1000, ?, ?)
	`, transaction.ID, time.Now().Unix(), core.EncodedTransactionSize(transaction), data); err != nil {
		t.Fatal(err)
	}
	for _, input := range transaction.Inputs {
		if _, err := ledger.database.ExecContext(context.Background(), `
			INSERT INTO mempool_inputs(tx_id, input_tx_id, input_index) VALUES(?, ?, ?)
		`, transaction.ID, input.TxID, int64(input.OutputIndex)); err != nil {
			t.Fatal(err)
		}
	}
	for index, output := range transaction.Outputs {
		if _, err := ledger.database.ExecContext(context.Background(), `
			INSERT INTO mempool_outputs(tx_id, output_index, amount, address) VALUES(?, ?, ?, ?)
		`, transaction.ID, index, int64(output.Amount), output.Address); err != nil {
			t.Fatal(err)
		}
	}
}
