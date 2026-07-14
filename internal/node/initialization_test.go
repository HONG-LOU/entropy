package node

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
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

func TestSeedModeUsesEphemeralIdentityWithoutPersistentWallet(t *testing.T) {
	directory := t.TempDir()
	config := testConfig(directory)
	config.SeedMode = true
	config.InitialPruneDepth = 20_000
	service, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	firstAddress := service.Address()
	dashboard, err := service.Dashboard()
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.WalletNeedsBackup || !dashboard.ArchiveMode || dashboard.PruneDepth != 0 {
		t.Fatalf("seed dashboard = backup %v archive %v depth %d", dashboard.WalletNeedsBackup, dashboard.ArchiveMode, dashboard.PruneDepth)
	}
	if _, err := service.Send(firstAddress, "0.01", "0.001"); !errors.Is(err, ErrSeedModeWalletUnavailable) {
		t.Fatalf("seed send error = %v", err)
	}
	if _, err := service.MineOnce(context.Background()); !errors.Is(err, ErrSeedModeWalletUnavailable) {
		t.Fatalf("seed mine error = %v", err)
	}
	if err := service.StartMining(); !errors.Is(err, ErrSeedModeWalletUnavailable) {
		t.Fatalf("seed start mining error = %v", err)
	}
	if _, err := service.RecoveryPhrase(); !errors.Is(err, ErrSeedModeWalletUnavailable) {
		t.Fatalf("seed recovery error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, walletVaultName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("seed wallet vault stat error = %v", err)
	}
	closeTestNode(t, service)

	reopened, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeTestNode(t, reopened) })
	if reopened.Address() == firstAddress {
		t.Fatal("ephemeral seed identity unexpectedly survived restart")
	}
}

func TestSeedModeRejectsPersistentWalletArtifacts(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, walletVaultName), []byte("must not be ignored"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := testConfig(directory)
	config.SeedMode = true
	service, err := New(config)
	if service != nil || err == nil || !strings.Contains(err.Error(), "refuses persistent wallet artifact") {
		t.Fatalf("seed with wallet artifact returned service=%v err=%v", service, err)
	}
}
