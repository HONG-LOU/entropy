package node

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"entropy/internal/core"
)

func TestTwoNodesPropagateTransactionAndBlock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeA, err := New(Config{DataDirectory: t.TempDir(), ListenAddress: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	nodeB, err := New(Config{DataDirectory: t.TempDir(), ListenAddress: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if err := nodeA.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := nodeB.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = nodeA.Close(shutdown)
		_ = nodeB.Close(shutdown)
	})
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
	waitFor(t, 10*time.Second, func() bool {
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
	if _, err := nodeA.Send(nodeB.Address(), "0.02", "0.001"); err != nil {
		t.Fatalf("send: %v", err)
	}
	waitFor(t, 2*time.Second, func() bool {
		dashboard, err := nodeB.Dashboard()
		return err == nil && dashboard.PendingCount == 1 && dashboard.SpendableBalance == "0.02000000"
	})
	if elapsed := time.Since(started); elapsed >= 2*time.Second {
		t.Fatalf("healthy peer propagation took %s with a slow peer configured", elapsed)
	}

	if _, err := nodeB.MineOnce(ctx); err != nil {
		t.Fatalf("node B mine: %v", err)
	}
	waitFor(t, 10*time.Second, func() bool {
		dashboard, err := nodeA.Dashboard()
		return err == nil && dashboard.Height == 2 && strings.HasPrefix(dashboard.ConfirmedBalance, "0.042")
	})
}

func TestReorgRestoresTransactionsFromOrphanedBlocks(t *testing.T) {
	service, err := New(Config{DataDirectory: t.TempDir(), ListenAddress: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.Close(context.Background()) }()
	recipient, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	common := core.NewState()
	if _, err := common.Mine(context.Background(), service.wallet.Address); err != nil {
		t.Fatal(err)
	}
	local, err := cloneState(common)
	if err != nil {
		t.Fatal(err)
	}
	utxo, err := local.SpendableUTXO()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := core.BuildTransaction(service.wallet, recipient.Address, core.UnitsPerENT/100, 0, utxo)
	if err != nil {
		t.Fatal(err)
	}
	if err := local.AddPending(tx); err != nil {
		t.Fatal(err)
	}
	if _, err := local.Mine(context.Background(), service.wallet.Address); err != nil {
		t.Fatal(err)
	}

	candidate, err := cloneState(common)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := candidate.Mine(context.Background(), recipient.Address); err != nil {
		t.Fatal(err)
	}
	if _, err := candidate.Mine(context.Background(), recipient.Address); err != nil {
		t.Fatal(err)
	}
	service.mu.Lock()
	service.state = local
	if err := service.store.SaveState(local); err != nil {
		service.mu.Unlock()
		t.Fatal(err)
	}
	service.mu.Unlock()
	if err := service.adoptIfBetter(candidate); err != nil {
		t.Fatal(err)
	}
	service.mu.RLock()
	defer service.mu.RUnlock()
	if len(service.state.Pending) != 1 || service.state.Pending[0].ID != tx.ID {
		t.Fatal("valid orphaned transaction was not restored to the pending pool")
	}
}

func TestDataDirectoryLockAndImmediateClose(t *testing.T) {
	directory := t.TempDir()
	first, err := New(Config{DataDirectory: directory, ListenAddress: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(Config{DataDirectory: directory, ListenAddress: "127.0.0.1:0"}); err == nil {
		t.Fatal("second node opened the same data directory")
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := first.Start(ctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
	if err := first.Close(shutdown); err != nil {
		t.Fatal(err)
	}
	stop()

	third, err := New(Config{DataDirectory: directory, ListenAddress: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("data directory remained locked after close: %v", err)
	}
	if err := third.Close(context.Background()); err != nil {
		t.Fatal(err)
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
