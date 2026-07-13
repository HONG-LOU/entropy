package ledger

import (
	"context"
	"strings"
	"testing"

	"entropy/internal/core"
)

func TestMixedCaseAddressIndexesRemainSpendable(t *testing.T) {
	ctx := context.Background()
	owner := newTestWallet(t)
	recipient := newTestWallet(t)
	mixedCase := recipient.Address[:4] + strings.ToUpper(recipient.Address[4:])
	state := core.NewState()
	if _, err := state.Mine(ctx, owner.Address); err != nil {
		t.Fatal(err)
	}
	utxo, err := state.SpendableUTXO()
	if err != nil {
		t.Fatal(err)
	}
	amount := core.Subsidy(1) / 2
	receive, err := core.BuildTransaction(owner, mixedCase, amount, 0, utxo)
	if err != nil {
		t.Fatal(err)
	}
	if err := state.AddPending(receive); err != nil {
		t.Fatal(err)
	}
	if _, err := state.Mine(ctx, owner.Address); err != nil {
		t.Fatal(err)
	}

	chain, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Close()
	if err := chain.ImportState(ctx, state); err != nil {
		t.Fatal(err)
	}
	confirmed, spendable, err := chain.Balances(ctx, recipient.Address)
	if err != nil || confirmed != amount || spendable != amount {
		t.Fatalf("canonical ledger balances = %d/%d, err %v", confirmed, spendable, err)
	}
	spendableUTXO, err := chain.SpendableUTXO(ctx, recipient.Address)
	if err != nil {
		t.Fatal(err)
	}
	spend, err := core.BuildTransaction(recipient, owner.Address, amount-MinimumRelayFeePerKibiByte, MinimumRelayFeePerKibiByte, spendableUTXO)
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
