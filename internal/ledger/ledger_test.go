package ledger

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/HONG-LOU/entcoin/internal/core"
)

func TestLedgerLifecycleReorgAndImport(t *testing.T) {
	ctx := context.Background()
	ledger, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	if _, err := ledger.DisconnectTip(ctx); !errors.Is(err, ErrGenesisDisconnect) {
		t.Fatalf("disconnect genesis error = %v, want ErrGenesisDisconnect", err)
	}

	alice := newTestWallet(t)
	bob := newTestWallet(t)
	carol := newTestWallet(t)

	block1, genesisTip := mineLedgerCandidate(t, ledger, alice.Address)
	if err := ledger.CommitMinedBlock(ctx, block1, genesisTip); err != nil {
		t.Fatalf("commit block 1: %v", err)
	}
	if err := ledger.CommitMinedBlock(ctx, block1, genesisTip); !errors.Is(err, ErrStaleTip) {
		t.Fatalf("repeat stale commit error = %v, want ErrStaleTip", err)
	}
	tip1, err := ledger.Tip(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tip1.Height != 1 || tip1.Hash != block1.Hash {
		t.Fatalf("tip after block 1 = %d/%s", tip1.Height, tip1.Hash)
	}
	wantWork := new(big.Int).Lsh(big.NewInt(1), uint(block1.Difficulty))
	work, err := ledger.WorkAt(ctx, 1)
	if err != nil || work.Cmp(wantWork) != 0 {
		t.Fatalf("work at block 1 = %v, err %v, want %v", work, err, wantWork)
	}
	// This lifecycle test exercises dependency, undo, and reorg behavior. Mark
	// its funding output non-coinbase in the private test database; maturity is
	// covered independently without publishing a valid pre-mined chain.
	if _, err := ledger.database.ExecContext(ctx, "UPDATE utxos SET coinbase = 0 WHERE tx_id = ?", block1.Transactions[0].ID); err != nil {
		t.Fatal(err)
	}

	aliceUTXO, err := ledger.SpendableUTXO(ctx, alice.Address)
	if err != nil {
		t.Fatal(err)
	}
	firstAmount := core.Subsidy(1) / 2
	firstFee := uint64(1_000)
	first, err := core.BuildTransaction(alice, bob.Address, firstAmount, firstFee, aliceUTXO)
	if err != nil {
		t.Fatalf("build first transaction: %v", err)
	}
	if err := ledger.AddTransaction(ctx, first); err != nil {
		t.Fatalf("add first transaction: %v", err)
	}
	bobUTXO, err := ledger.SpendableUTXO(ctx, bob.Address)
	if err != nil {
		t.Fatal(err)
	}
	secondAmount := firstAmount / 2
	secondFee := uint64(1_000)
	second, err := core.BuildTransaction(bob, carol.Address, secondAmount, secondFee, bobUTXO)
	if err != nil {
		t.Fatalf("build dependent transaction: %v", err)
	}
	if err := ledger.AddTransaction(ctx, second); err != nil {
		t.Fatalf("add dependent transaction: %v", err)
	}

	block2, expectedTip := mineLedgerCandidate(t, ledger, alice.Address)
	if expectedTip.Hash != block1.Hash || len(block2.Transactions) != 3 {
		t.Fatalf("block 2 candidate has tip %s and %d transactions", expectedTip.Hash, len(block2.Transactions))
	}
	if err := ledger.CommitMinedBlock(ctx, block2, expectedTip); err != nil {
		t.Fatalf("commit block 2: %v", err)
	}
	confirmed, spendable, err := ledger.Balances(ctx, carol.Address)
	if err != nil || confirmed != secondAmount || spendable != secondAmount {
		t.Fatalf("Carol balances after confirmation = %d/%d, err %v", confirmed, spendable, err)
	}
	if count, err := ledger.MempoolCount(ctx); err != nil || count != 0 {
		t.Fatalf("mempool after confirmation = %d, err %v", count, err)
	}
	history, err := ledger.TransactionHistory(ctx, carol.Address, 10)
	if err != nil || len(history) != 1 || history[0].ID != second.ID || history[0].Pending ||
		history[0].Confirmations != 1 || history[0].Received != secondAmount || history[0].Sent != 0 {
		t.Fatalf("confirmed history = %#v, err %v", history, err)
	}
	bobHistory, err := ledger.TransactionHistory(ctx, bob.Address, 10)
	if err != nil {
		t.Fatalf("query Bob history: %v", err)
	}
	bobSecond := historyRecordByID(t, bobHistory, second.ID)
	wantBobChange := firstAmount - secondAmount - secondFee
	if bobSecond.Received != wantBobChange || bobSecond.Sent != firstAmount {
		t.Fatalf("Bob change transaction received/sent = %d/%d, want %d/%d", bobSecond.Received, bobSecond.Sent, wantBobChange, firstAmount)
	}
	bobReceived, err := ledger.FilteredTransactionHistory(ctx, bob.Address, 1, TransactionHistoryReceived)
	if err != nil || len(bobReceived) != 1 || bobReceived[0].ID != first.ID {
		t.Fatalf("filtered received history = %#v, err %v; want older transaction %s", bobReceived, err, first.ID)
	}
	bobSent, err := ledger.FilteredTransactionHistory(ctx, bob.Address, 1, TransactionHistorySent)
	if err != nil || len(bobSent) != 1 || bobSent[0].ID != second.ID {
		t.Fatalf("filtered sent history = %#v, err %v; want transaction %s", bobSent, err, second.ID)
	}
	aliceMining, err := ledger.FilteredTransactionHistory(ctx, alice.Address, 1, TransactionHistoryMining)
	if err != nil || len(aliceMining) != 1 || aliceMining[0].ID != block2.Transactions[0].ID {
		t.Fatalf("filtered mining history = %#v, err %v", aliceMining, err)
	}
	if _, err := ledger.FilteredTransactionHistory(ctx, bob.Address, 1, "unknown"); err == nil {
		t.Fatal("unknown transaction history filter was accepted")
	}
	if _, err := ledger.database.ExecContext(ctx, "UPDATE blocks SET data = NULL WHERE height = 2"); err != nil {
		t.Fatalf("simulate pruning block 2: %v", err)
	}
	if _, err := ledger.Block(ctx, 2); !errors.Is(err, ErrBlockPruned) {
		t.Fatalf("pruned block read error = %v, want ErrBlockPruned", err)
	}

	disconnected, err := ledger.DisconnectTip(ctx)
	if err != nil {
		t.Fatalf("disconnect block 2: %v", err)
	}
	if disconnected.Hash != block2.Hash || len(disconnected.Transactions) != 3 {
		t.Fatalf("disconnected block = %s with %d transactions", disconnected.Hash, len(disconnected.Transactions))
	}
	confirmed, spendable, err = ledger.Balances(ctx, carol.Address)
	if err != nil || confirmed != 0 || spendable != secondAmount {
		t.Fatalf("Carol balances after disconnect = %d/%d, err %v", confirmed, spendable, err)
	}
	if count, err := ledger.MempoolCount(ctx); err != nil || count != 2 {
		t.Fatalf("mempool after disconnect = %d, err %v", count, err)
	}
	var disconnectedOutputs int
	if err := ledger.database.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM utxos WHERE tx_id IN (?, ?)
	`, first.ID, second.ID).Scan(&disconnectedOutputs); err != nil || disconnectedOutputs != 0 {
		t.Fatalf("confirmed outputs created in disconnected block = %d, err %v", disconnectedOutputs, err)
	}
	history, err = ledger.TransactionHistory(ctx, carol.Address, 10)
	if err != nil || len(history) != 1 || history[0].ID != second.ID || !history[0].Pending ||
		history[0].Received != secondAmount || history[0].Sent != 0 {
		t.Fatalf("pending history = %#v, err %v", history, err)
	}
	bobHistory, err = ledger.TransactionHistory(ctx, bob.Address, 10)
	if err != nil {
		t.Fatalf("query pending Bob history: %v", err)
	}
	bobSecond = historyRecordByID(t, bobHistory, second.ID)
	if bobSecond.Received != wantBobChange || bobSecond.Sent != firstAmount || !bobSecond.Pending {
		t.Fatalf("pending Bob change received/sent = %d/%d, pending=%v", bobSecond.Received, bobSecond.Sent, bobSecond.Pending)
	}
	bobReceived, err = ledger.FilteredTransactionHistory(ctx, bob.Address, 1, TransactionHistoryReceived)
	if err != nil || len(bobReceived) != 1 || bobReceived[0].ID != first.ID || !bobReceived[0].Pending {
		t.Fatalf("filtered pending received history = %#v, err %v", bobReceived, err)
	}
	bobSent, err = ledger.FilteredTransactionHistory(ctx, bob.Address, 1, TransactionHistorySent)
	if err != nil || len(bobSent) != 1 || bobSent[0].ID != second.ID || !bobSent[0].Pending {
		t.Fatalf("filtered pending sent history = %#v, err %v", bobSent, err)
	}
	if err := ledger.ConnectBlock(ctx, block2); err != nil {
		t.Fatalf("reconnect block 2: %v", err)
	}
	if _, err := ledger.DisconnectTip(ctx); err != nil {
		t.Fatalf("disconnect block 2 before reorg: %v", err)
	}
	legacyWithRejectedPending := core.NewState()
	legacyWithRejectedPending.Blocks = append(legacyWithRejectedPending.Blocks, block1)
	legacyWithRejectedPending.Pending = []core.Transaction{{ID: strings.Repeat("f", 64)}}
	rejectedPendingImport, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := rejectedPendingImport.ImportState(ctx, legacyWithRejectedPending); err != nil {
		t.Fatalf("policy-invalid legacy pending transaction blocked confirmed import: %v", err)
	}
	if count, err := rejectedPendingImport.MempoolCount(ctx); err != nil || count != 0 {
		t.Fatalf("rejected legacy pending transaction count = %d, err %v", count, err)
	}
	if importedTip, err := rejectedPendingImport.Tip(ctx); err != nil || importedTip.Hash != block1.Hash {
		t.Fatalf("confirmed legacy tip was not imported: %#v, err %v", importedTip, err)
	}
	if err := rejectedPendingImport.Close(); err != nil {
		t.Fatalf("close rejected-pending import ledger: %v", err)
	}

	alternative := core.NewState()
	if _, err := alternative.Mine(ctx, bob.Address); err != nil {
		t.Fatalf("mine alternative block 1: %v", err)
	}
	if _, err := alternative.Mine(ctx, bob.Address); err != nil {
		t.Fatalf("mine alternative block 2: %v", err)
	}
	badBlock2 := alternative.Blocks[2]
	badBlock2.Transactions = append([]core.Transaction(nil), badBlock2.Transactions...)
	badBlock2.Transactions[0].Outputs = append([]core.TxOutput(nil), badBlock2.Transactions[0].Outputs...)
	badBlock2.Transactions[0].Outputs[0].Amount++
	if err := ledger.ReplaceFrom(ctx, 0, []core.Block{alternative.Blocks[1], badBlock2}); err == nil {
		t.Fatal("invalid replacement chain was accepted")
	}
	afterFailure, err := ledger.Tip(ctx)
	if err != nil || afterFailure.Hash != block1.Hash || afterFailure.Height != 1 {
		t.Fatalf("tip after failed replacement = %#v, err %v", afterFailure, err)
	}
	if count, err := ledger.MempoolCount(ctx); err != nil || count != 2 {
		t.Fatalf("mempool changed after failed replacement: %d, err %v", count, err)
	}

	if err := ledger.ReplaceFrom(ctx, 0, alternative.Blocks[1:]); err != nil {
		t.Fatalf("replace with stronger chain: %v", err)
	}
	alternativeTip, err := ledger.Tip(ctx)
	if err != nil || alternativeTip.Hash != alternative.Blocks[2].Hash || alternativeTip.Height != 2 {
		t.Fatalf("alternative tip = %#v, err %v", alternativeTip, err)
	}
	if count, err := ledger.MempoolCount(ctx); err != nil || count != 0 {
		t.Fatalf("invalid orphan transactions remained in mempool: %d, err %v", count, err)
	}
	if err := ledger.ReplaceFrom(ctx, 0, alternative.Blocks[1:2]); !errors.Is(err, ErrInsufficientWork) {
		t.Fatalf("weaker replacement error = %v, want ErrInsufficientWork", err)
	}
	afterWeakChain, err := ledger.Tip(ctx)
	if err != nil || afterWeakChain.Hash != alternative.Blocks[2].Hash {
		t.Fatalf("tip changed after weaker replacement = %#v, err %v", afterWeakChain, err)
	}

	imported, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer imported.Close()
	if err := imported.ImportState(ctx, alternative); err != nil {
		t.Fatalf("import legacy state: %v", err)
	}
	importedTip, err := imported.Tip(ctx)
	if err != nil || importedTip.Hash != alternative.Blocks[2].Hash {
		t.Fatalf("imported tip = %#v, err %v", importedTip, err)
	}
	confirmed, spendable, err = imported.Balances(ctx, bob.Address)
	wantBalance := core.Subsidy(1) + core.Subsidy(2)
	if err != nil || confirmed != wantBalance || spendable != 0 {
		t.Fatalf("imported Bob balances = %d/%d, err %v, want %d/0", confirmed, spendable, err, wantBalance)
	}
	if err := imported.ImportState(ctx, alternative); err == nil {
		t.Fatal("legacy state overwrote an initialized ledger")
	}
	prunedThrough, err := ledger.Prune(ctx, 1)
	if err != nil || prunedThrough != 1 {
		t.Fatalf("prune horizon = %d, err %v, want 1", prunedThrough, err)
	}
	if stored, err := ledger.PrunedThrough(ctx); err != nil || stored != 1 {
		t.Fatalf("stored prune horizon = %d, err %v", stored, err)
	}
	if _, err := ledger.Block(ctx, 1); !errors.Is(err, ErrBlockPruned) {
		t.Fatalf("pruned archive block error = %v, want ErrBlockPruned", err)
	}
	prunedHistory, err := ledger.TransactionHistory(ctx, bob.Address, 10)
	if err != nil {
		t.Fatalf("query pruned history: %v", err)
	}
	prunedCoinbase := historyRecordByID(t, prunedHistory, alternative.Blocks[1].Transactions[0].ID)
	if !prunedCoinbase.Pruned || prunedCoinbase.Received != core.Subsidy(1) || prunedCoinbase.Sent != 0 {
		t.Fatalf("pruned coinbase history = %#v", prunedCoinbase)
	}
	if err := ledger.ReplaceFrom(ctx, 0, alternative.Blocks[1:]); !errors.Is(err, ErrReorgBeyondPrune) {
		t.Fatalf("deep pruned reorg error = %v, want ErrReorgBeyondPrune", err)
	}
	if _, err := ledger.DisconnectTip(ctx); err != nil {
		t.Fatalf("disconnect retained block above prune horizon: %v", err)
	}
	if _, err := ledger.DisconnectTip(ctx); !errors.Is(err, ErrReorgBeyondPrune) {
		t.Fatalf("disconnect through prune horizon error = %v, want ErrReorgBeyondPrune", err)
	}
	if err := ledger.ConnectBlock(ctx, alternative.Blocks[2]); err != nil {
		t.Fatalf("reconnect retained block after prune-boundary check: %v", err)
	}
	if err := ledger.quickCheck(ctx); err != nil {
		t.Fatalf("database integrity after lifecycle: %v", err)
	}
}

func newTestWallet(t *testing.T) *core.Wallet {
	t.Helper()
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	return wallet
}

func mineLedgerCandidate(t *testing.T, ledger *Ledger, address string) (core.Block, Tip) {
	t.Helper()
	candidate, tip, err := ledger.BuildMiningCandidate(context.Background(), address)
	if err != nil {
		t.Fatalf("build mining candidate: %v", err)
	}
	mined, err := core.MineBlock(context.Background(), candidate)
	if err != nil {
		t.Fatalf("mine candidate: %v", err)
	}
	return mined, tip
}

func historyRecordByID(t *testing.T, records []TransactionRecord, id string) TransactionRecord {
	t.Helper()
	for _, record := range records {
		if record.ID == id {
			return record
		}
	}
	t.Fatalf("transaction %s is absent from history %#v", id, records)
	return TransactionRecord{}
}
