package core

import (
	"context"
	"crypto/elliptic"
	"encoding/asn1"
	"math"
	"strings"
	"testing"
	"time"
)

func TestEmissionIsExactlyTwoMillionENT(t *testing.T) {
	tests := []struct {
		height uint64
		want   uint64
	}{
		{0, 0},
		{1, BaseSubsidy + 1},
		{BonusSubsidyBlocks, BaseSubsidy + 1},
		{BonusSubsidyBlocks + 1, BaseSubsidy},
		{EmissionBlocks, BaseSubsidy},
		{EmissionBlocks + 1, 0},
	}
	for _, test := range tests {
		if got := Subsidy(test.height); got != test.want {
			t.Fatalf("Subsidy(%d) = %d, want %d", test.height, got, test.want)
		}
	}
	if got := MintedThrough(EmissionBlocks); got != MaxSupply {
		t.Fatalf("total emission = %d, want %d", got, MaxSupply)
	}
	if got := MintedThrough(EmissionBlocks + 10_000); got != MaxSupply {
		t.Fatalf("emission exceeded cap: %d", got)
	}
}

func TestAmountRoundTrip(t *testing.T) {
	values := []string{"0", "0.00000001", "12.5", "2000000"}
	for _, value := range values {
		amount, err := ParseAmount(value)
		if err != nil {
			t.Fatalf("ParseAmount(%q): %v", value, err)
		}
		roundTrip, err := ParseAmount(FormatAmount(amount))
		if err != nil || roundTrip != amount {
			t.Fatalf("amount round trip for %q: got %d, err %v", value, roundTrip, err)
		}
	}
	for _, value := range []string{"", "-1", ".1", "1.000000001", "1e2"} {
		if _, err := ParseAmount(value); err == nil {
			t.Fatalf("ParseAmount(%q) unexpectedly succeeded", value)
		}
	}
}

func TestMineTransferAndRejectTampering(t *testing.T) {
	alice, err := NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	bob, err := NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	state := NewState()
	if _, err := state.Mine(context.Background(), alice.Address); err != nil {
		t.Fatalf("mine first block: %v", err)
	}

	amount := uint64(2 * UnitsPerENT / 100)
	fee := uint64(UnitsPerENT / 1000)
	utxo, err := state.SpendableUTXO()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := BuildTransaction(alice, bob.Address, amount, fee, utxo)
	if err != nil {
		t.Fatalf("build transaction: %v", err)
	}
	highS := tx
	highS.Inputs = append([]TxInput(nil), tx.Inputs...)
	var signature ecdsaSignature
	if _, err := asn1.Unmarshal(highS.Inputs[0].Signature, &signature); err != nil {
		t.Fatal(err)
	}
	signature.S.Sub(elliptic.P256().Params().N, signature.S)
	highS.Inputs[0].Signature, err = asn1.Marshal(signature)
	if err != nil {
		t.Fatal(err)
	}
	highS.ID = highS.ComputeID()
	if err := state.AddPending(highS); err == nil {
		t.Fatal("high-S transaction signature was accepted")
	}
	if err := state.AddPending(tx); err != nil {
		t.Fatalf("add pending transaction: %v", err)
	}
	if balance, err := state.Balance(bob.Address, true); err != nil || balance != amount {
		t.Fatalf("pending Bob balance = %d, err %v", balance, err)
	}
	if _, err := state.Mine(context.Background(), bob.Address); err != nil {
		t.Fatalf("mine confirmation block: %v", err)
	}
	if balance, err := state.Balance(bob.Address, false); err != nil || balance != amount+Subsidy(2)+fee {
		t.Fatalf("confirmed Bob balance = %d, err %v", balance, err)
	}

	tampered := *state
	tampered.Blocks = append([]Block(nil), state.Blocks...)
	tampered.Blocks[2].Transactions = append([]Transaction(nil), state.Blocks[2].Transactions...)
	tampered.Blocks[2].Transactions[1].Outputs = append([]TxOutput(nil), state.Blocks[2].Transactions[1].Outputs...)
	tampered.Blocks[2].Transactions[1].Outputs[0].Amount++
	if err := tampered.Validate(); err == nil {
		t.Fatal("tampered transaction was accepted")
	}
}

func TestAddressChecksum(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := wallet.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := wallet.Address[:len(wallet.Address)-1] + "0"
	if bad == wallet.Address {
		bad = wallet.Address[:len(wallet.Address)-1] + "1"
	}
	if err := ValidateAddress(bad); err == nil {
		t.Fatal("address with bad checksum was accepted")
	}
}

func TestDifficultyAdjustmentUsesMedianTime(t *testing.T) {
	steady := blocksWithSpacing(10)
	if got := expectedDifficulty(steady, FirstAdjustment); got != InitialDifficulty {
		t.Fatalf("steady difficulty = %d, want %d", got, InitialDifficulty)
	}
	fast := blocksWithSpacing(2)
	if got := expectedDifficulty(fast, FirstAdjustment); got != InitialDifficulty+2 {
		t.Fatalf("fast difficulty = %d, want %d", got, InitialDifficulty+2)
	}
	slow := blocksWithSpacing(50)
	if got := expectedDifficulty(slow, FirstAdjustment); got != InitialDifficulty-2 {
		t.Fatalf("slow difficulty = %d, want %d", got, InitialDifficulty-2)
	}
	steady[len(steady)-1].Timestamp += 10_000
	if got := expectedDifficulty(steady, FirstAdjustment); got != InitialDifficulty {
		t.Fatalf("one timestamp outlier changed difficulty to %d", got)
	}
}

func TestConsensusResourceAndTimestampLimits(t *testing.T) {
	wallet, err := NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	oversized := Transaction{
		Inputs:  make([]TxInput, MaxTransactionInputs+1),
		Outputs: []TxOutput{{Amount: 1, Address: wallet.Address}},
	}
	if _, err := validateRegularTransaction(oversized, UTXO{}); err == nil {
		t.Fatal("transaction input limit was not enforced")
	}

	coinbase, err := NewCoinbase(wallet.Address, 1, Subsidy(1))
	if err != nil {
		t.Fatal(err)
	}
	coinbase.Outputs = append(coinbase.Outputs, TxOutput{Amount: 1, Address: wallet.Address})
	coinbase.ID = coinbase.ComputeID()
	if err := validateCoinbase(coinbase, Subsidy(1)+1); err == nil {
		t.Fatal("coinbase output limit was not enforced")
	}

	genesis := GenesisBlock()
	badTime := Block{
		Version:      StateVersion,
		Height:       1,
		Timestamp:    math.MaxInt64,
		PreviousHash: genesis.Hash,
		Difficulty:   InitialDifficulty,
	}
	if err := validateBlockHeader(badTime, genesis, []Block{genesis}); err == nil || !strings.Contains(err.Error(), "future") {
		t.Fatalf("future timestamp error = %v", err)
	}
	badTime.Timestamp = genesis.Timestamp
	if err := validateBlockHeader(badTime, genesis, []Block{genesis}); err == nil || !strings.Contains(err.Error(), "median") {
		t.Fatalf("MTP timestamp error = %v", err)
	}
	badTime.Timestamp = time.Now().Unix() + MaxFutureSeconds + 1
	if err := validateBlockHeader(badTime, genesis, []Block{genesis}); err == nil || !strings.Contains(err.Error(), "future") {
		t.Fatalf("future drift error = %v", err)
	}

	if got := clampDifficulty(1_000); got != MaximumDifficulty {
		t.Fatalf("maximum difficulty clamp = %d", got)
	}
}

func blocksWithSpacing(seconds int64) []Block {
	blocks := make([]Block, FirstAdjustment)
	for index := range blocks {
		blocks[index] = Block{
			Height:     uint64(index),
			Timestamp:  genesisTimestamp + int64(index)*seconds,
			Difficulty: InitialDifficulty,
		}
	}
	blocks[0].Difficulty = 0
	return blocks
}
