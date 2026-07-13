package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"entropy/internal/core"

	"github.com/gorilla/websocket"
)

func TestTwoNodesPropagateTransactionAndBlockIncrementally(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	nodeA := newTestNode(t)
	nodeB := newTestNode(t)
	startTestNode(t, ctx, nodeA)
	startTestNode(t, ctx, nodeB)
	peerA := "http://" + nodeA.ActualAddress()
	peerB := "http://" + nodeB.ActualAddress()
	if err := nodeA.AddPeer(peerB); err != nil {
		t.Fatal(err)
	}
	if err := nodeB.AddPeer(peerA); err != nil {
		t.Fatal(err)
	}

	if _, err := nodeA.MineOnce(ctx); err != nil {
		t.Fatalf("node A mine: %v", err)
	}
	waitFor(t, 20*time.Second, func() bool {
		dashboard, err := nodeB.Dashboard()
		return err == nil && dashboard.Height == 1
	})

	slowPeer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		time.Sleep(3 * time.Second)
		writer.WriteHeader(http.StatusOK)
	}))
	defer slowPeer.Close()
	if err := nodeA.AddPeer(slowPeer.URL); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	transaction, err := nodeA.Send(nodeB.Address(), "0.02", "0.001")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("local transaction acceptance waited on a slow peer for %s", elapsed)
	}
	waitFor(t, 3*time.Second, func() bool {
		dashboard, err := nodeB.Dashboard()
		return err == nil && dashboard.PendingCount == 1 && dashboard.SpendableBalance == "0.02000000"
	})

	if _, err := nodeB.MineOnce(ctx); err != nil {
		t.Fatalf("node B mine: %v", err)
	}
	waitFor(t, 20*time.Second, func() bool {
		dashboard, err := nodeA.Dashboard()
		return err == nil && dashboard.Height == 2 && strings.HasPrefix(dashboard.ConfirmedBalance, "0.042")
	})
	history, err := nodeB.TransactionHistory(20)
	if err != nil {
		t.Fatal(err)
	}
	if !historyContains(history, transaction.ID, false) {
		t.Fatal("confirmed transaction was missing from wallet history")
	}
}

func TestIncrementalSyncReorgRestoresOrphanedTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	nodeA := newTestNode(t)
	nodeB := newTestNode(t)
	recipient, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	common := core.NewState()
	if _, err := common.Mine(ctx, nodeA.Address()); err != nil {
		t.Fatal(err)
	}
	if err := nodeA.ledger.ImportState(ctx, common); err != nil {
		t.Fatal(err)
	}
	if err := nodeB.ledger.ImportState(ctx, common); err != nil {
		t.Fatal(err)
	}
	utxo, err := nodeA.ledger.SpendableUTXO(ctx, nodeA.Address())
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := core.BuildTransaction(&nodeA.wallet, recipient.Address, core.UnitsPerENT/100, 1_000, utxo)
	if err != nil {
		t.Fatal(err)
	}
	if err := nodeA.ledger.AddTransaction(ctx, transaction); err != nil {
		t.Fatal(err)
	}
	if _, err := nodeA.MineOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := nodeB.MineOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := nodeB.MineOnce(ctx); err != nil {
		t.Fatal(err)
	}

	startTestNode(t, ctx, nodeA)
	startTestNode(t, ctx, nodeB)
	if err := nodeA.AddPeer("http://" + nodeB.ActualAddress()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 30*time.Second, func() bool {
		dashboard, err := nodeA.Dashboard()
		return err == nil && dashboard.Height == 3
	})
	pending, err := nodeA.ledger.MempoolTransactions(ctx, core.MaxPendingTransactions)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != transaction.ID {
		t.Fatal("valid transaction from the orphaned block was not restored")
	}
}

func TestRestartPreservesProtectedWalletAndSQLiteLedger(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	directory := t.TempDir()
	first, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	address := first.Address()
	block, err := first.MineOnce(ctx)
	if err != nil {
		t.Fatal(err)
	}
	closeTestNode(t, first)
	second, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	defer closeTestNode(t, second)
	dashboard, err := second.Dashboard()
	if err != nil {
		t.Fatal(err)
	}
	if second.Address() != address || dashboard.Height != block.Height || dashboard.TipHash != block.Hash {
		t.Fatalf("restart changed node state: address=%s height=%d tip=%s", second.Address(), dashboard.Height, dashboard.TipHash)
	}
}

func TestDataDirectoryLockAndImmediateClose(t *testing.T) {
	directory := t.TempDir()
	first, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(testConfig(directory)); err == nil {
		t.Fatal("second node opened the same data directory")
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := first.Start(ctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	closeTestNode(t, first)
	third, err := New(testConfig(directory))
	if err != nil {
		t.Fatalf("data directory remained locked after close: %v", err)
	}
	closeTestNode(t, third)
}

func TestPrunePolicyPersistsAcrossRestart(t *testing.T) {
	directory := t.TempDir()
	first, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.PruneLedger(1); err == nil {
		t.Fatal("unsafe prune depth was accepted")
	}
	if _, err := first.PruneLedger(120); err != nil {
		t.Fatal(err)
	}
	closeTestNode(t, first)
	second, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	defer closeTestNode(t, second)
	dashboard, err := second.Dashboard()
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.PruneDepth != 120 || dashboard.ArchiveMode {
		t.Fatalf("restored prune policy = depth %d archive %v", dashboard.PruneDepth, dashboard.ArchiveMode)
	}
	if _, err := second.PruneLedger(0); err != nil {
		t.Fatal(err)
	}
}

func TestV1FullStateEndpointIsRemoved(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := newTestNode(t)
	startTestNode(t, ctx, service)
	response, err := http.Get("http://" + service.ActualAddress() + "/v1/state")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("legacy full-state endpoint returned HTTP %d", response.StatusCode)
	}
}

func TestRejectsUnsafePeerURLs(t *testing.T) {
	unsafe := []string{
		"http://<svg&Tab;onload=alert(1)>",
		"http://user:pass@localhost:47821",
		"http://localhost:0",
		"http://localhost:70000",
		"http://localhost:47821/path",
	}
	for _, peer := range unsafe {
		if _, err := normalizePeer(peer); err == nil {
			t.Fatalf("unsafe peer URL was accepted: %s", peer)
		}
	}
	for _, peer := range []string{"http://localhost:47821", "http://127.0.0.1:47821", "http://[::1]:47821"} {
		if _, err := normalizePeer(peer); err != nil {
			t.Fatalf("valid peer URL %s rejected: %v", peer, err)
		}
	}
}

func TestProtocolJSONRejectsDuplicateUnknownAndDeepFields(t *testing.T) {
	tests := []string{
		`{"locator":[],"limit":1,"limit":2}`,
		`{"locator":[],"limit":1,"unexpected":true}`,
		`{"locator":[],"limit":1} {}`,
		strings.Repeat("[", 130) + strings.Repeat("]", 130),
	}
	for _, body := range tests {
		var request headersRequest
		if err := decodeLimitedJSON(strings.NewReader(body), 16<<10, &request); err == nil {
			t.Fatalf("unsafe JSON was accepted: %.80s", body)
		}
	}
	var valid headersRequest
	if err := decodeLimitedJSON(strings.NewReader(`{"locator":["hash"],"limit":1}`), 16<<10, &valid); err != nil {
		t.Fatalf("valid JSON was rejected: %v", err)
	}
}

func TestWebSocketPerIPConnectionLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := newTestNode(t)
	startTestNode(t, ctx, service)
	endpoint := "ws://" + service.ActualAddress() + "/v2/p2p"
	connections := make([]*websocket.Conn, 0, 4)
	defer func() {
		for _, connection := range connections {
			_ = connection.Close()
		}
	}()
	for range 4 {
		connection, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
		if err != nil {
			t.Fatal(err)
		}
		connections = append(connections, connection)
	}
	connection, response, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if connection != nil {
		_ = connection.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("fifth websocket = response %v, err %v", response, err)
	}
	_ = response.Body.Close()
}

func newTestNode(t *testing.T) *Service {
	t.Helper()
	service, err := New(testConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeTestNode(t, service) })
	return service
}

func testConfig(directory string) Config {
	return Config{
		DataDirectory:    directory,
		ListenAddress:    "127.0.0.1:0",
		SyncInterval:     100 * time.Millisecond,
		DisableDiscovery: true,
	}
}

func startTestNode(t *testing.T, ctx context.Context, service *Service) {
	t.Helper()
	if err := service.Start(ctx); err != nil {
		t.Fatal(err)
	}
}

func closeTestNode(t *testing.T, service *Service) {
	t.Helper()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := service.Close(shutdown); err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("close node: %v", err)
	}
}

func historyContains(history []TransactionSummary, id string, pending bool) bool {
	for _, transaction := range history {
		if transaction.ID == id && transaction.Pending == pending {
			return true
		}
	}
	return false
}

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal(fmt.Errorf("condition not met within %s", timeout))
}
