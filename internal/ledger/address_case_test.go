package ledger

import (
	"context"
	"strings"
	"testing"

	"github.com/HONG-LOU/entcoin/internal/core"
)

func TestMixedCaseAddressIndexesRemainSpendable(t *testing.T) {
	ctx := context.Background()
	owner := newTestWallet(t)
	recipient := newTestWallet(t)
	mixedCase := recipient.Address[:4] + strings.ToUpper(recipient.Address[4:])
	chain, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Close()
	fundingID := strings.Repeat("e", 64)
	const fundingAmount uint64 = 100_000
	if _, err := chain.database.ExecContext(ctx, `
		INSERT INTO utxos(tx_id, output_index, amount, address, created_height, coinbase)
		VALUES(?, 0, ?, ?, 0, 0)
	`, fundingID, int64(fundingAmount), owner.Address); err != nil {
		t.Fatal(err)
	}
	receiveAmount := uint64(50_000)
	receive, err := core.BuildTransaction(owner, mixedCase, receiveAmount, MinimumRelayFeePerKibiByte, core.UTXO{
		{TxID: fundingID, Index: 0}: {Amount: fundingAmount, Address: owner.Address},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.AddTransaction(ctx, receive); err != nil {
		t.Fatal(err)
	}
	block, tip := mineLedgerCandidate(t, chain, owner.Address)
	if err := chain.CommitMinedBlock(ctx, block, tip); err != nil {
		t.Fatal(err)
	}
	confirmed, spendable, err := chain.Balances(ctx, recipient.Address)
	if err != nil || confirmed != receiveAmount || spendable != receiveAmount {
		t.Fatalf("canonical ledger balances = %d/%d, err %v", confirmed, spendable, err)
	}
	spendableUTXO, err := chain.SpendableUTXO(ctx, recipient.Address)
	if err != nil {
		t.Fatal(err)
	}
	spend, err := core.BuildTransaction(recipient, owner.Address, receiveAmount-MinimumRelayFeePerKibiByte, MinimumRelayFeePerKibiByte, spendableUTXO)
	if err != nil {
		t.Fatalf("build spend from mixed-case indexed output: %v", err)
	}
	if err := chain.AddTransaction(ctx, spend); err != nil {
		t.Fatalf("add spend from mixed-case indexed output: %v", err)
	}
	history, err := chain.TransactionHistory(ctx, recipient.Address, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("mixed-case address history rows = %d, want 2", len(history))
	}
}
