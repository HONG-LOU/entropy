package node

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestNewContextRejectsCanceledInitialization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service, err := NewContext(ctx, testConfig(t.TempDir()))
	if service != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled initialization returned service=%v err=%v", service, err)
	}
}

func TestListenerFallsBackWhenPreferredPortIsBusy(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	listener, fallback, err := listenNode(occupied.Addr().String(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if !fallback || listener.Addr().String() == occupied.Addr().String() {
		t.Fatalf("listener fallback=%v address=%s occupied=%s", fallback, listener.Addr(), occupied.Addr())
	}
}

func TestInitialPruneDepthAppliesOnlyToFreshLedger(t *testing.T) {
	directory := t.TempDir()
	config := testConfig(directory)
	config.InitialPruneDepth = 20_000
	service, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	depth, err := service.ledger.PruneDepth(context.Background())
	if err != nil || depth != 20_000 {
		t.Fatalf("fresh prune depth = %d, err %v", depth, err)
	}
	closeTestNode(t, service)

	config.InitialPruneDepth = 30_000
	reopened, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeTestNode(t, reopened) })
	depth, err = reopened.ledger.PruneDepth(context.Background())
	if err != nil || depth != 20_000 {
		t.Fatalf("reopened prune depth = %d, err %v, want persisted 20000", depth, err)
	}
}
