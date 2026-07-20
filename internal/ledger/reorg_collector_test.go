package ledger

import (
	"strconv"
	"testing"

	"github.com/HONG-LOU/entcoin/internal/core"
)

func TestOrphanedMempoolCollectorPreservesDependencyOrder(t *testing.T) {
	collector := newOrphanedMempoolCollector(4, maxOrphanedMempoolBytes)
	collector.add(testTransactions("n1", "n2"))
	collector.add(testTransactions("m1", "m2"))
	collector.add(testTransactions("o1", "o2"))

	want := []string{"o1", "o2", "m1", "m2"}
	assertTransactionIDs(t, collector.transactions(), want)
}

func TestOrphanedMempoolCollectorEnforcesByteAndCountBudgets(t *testing.T) {
	oldest := testTransactions("o1", "o2")
	byteBudget := core.EncodedTransactionSize(oldest[0]) + core.EncodedTransactionSize(oldest[1])
	collector := newOrphanedMempoolCollector(10, byteBudget)
	collector.add(testTransactions("n1", "n2"))
	collector.add(oldest)
	assertTransactionIDs(t, collector.transactions(), []string{"o1", "o2"})

	transactions := make([]core.Transaction, core.MaxPendingTransactions+5)
	for index := range transactions {
		transactions[index].ID = strconv.Itoa(index)
	}
	collector = newOrphanedMempoolCollector(core.MaxPendingTransactions, maxOrphanedMempoolBytes)
	collector.add(transactions)
	limited := collector.transactions()
	if len(limited) != core.MaxPendingTransactions || limited[0].ID != "0" || limited[len(limited)-1].ID != strconv.Itoa(core.MaxPendingTransactions-1) {
		t.Fatalf("count-limited orphaned transactions = %d, first=%q last=%q", len(limited), limited[0].ID, limited[len(limited)-1].ID)
	}
}

func testTransactions(ids ...string) []core.Transaction {
	transactions := make([]core.Transaction, len(ids))
	for index, id := range ids {
		transactions[index].ID = id
	}
	return transactions
}

func assertTransactionIDs(t *testing.T, transactions []core.Transaction, want []string) {
	t.Helper()
	if len(transactions) != len(want) {
		t.Fatalf("transaction count = %d, want %d", len(transactions), len(want))
	}
	for index, id := range want {
		if transactions[index].ID != id {
			t.Fatalf("transaction %d = %q, want %q", index, transactions[index].ID, id)
		}
	}
}
