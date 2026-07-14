package store

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"entropy/internal/core"
)

func TestStateAndWalletRoundTrip(t *testing.T) {
	directory := t.TempDir()
	storage := New(directory)
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveLegacyWallet(wallet); err != nil {
		t.Fatal(err)
	}
	state := core.NewState()
	if _, err := state.Mine(context.Background(), wallet.Address); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveLegacyState(state); err != nil {
		t.Fatal(err)
	}
	reloaded, found, err := storage.LoadLegacyState()
	if err != nil || !found {
		t.Fatal(err)
	}
	if len(reloaded.Blocks) != 2 || reloaded.Blocks[1].Hash != state.Blocks[1].Hash {
		t.Fatal("reloaded chain does not match saved chain")
	}

	chainPath := filepath.Join(directory, "chain.json")
	if err := os.WriteFile(chainPath, []byte("{truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := storage.LoadLegacyState(); err == nil {
		t.Fatal("corrupt state was silently accepted")
	}
}

func TestPublishedTestnetChainJSONIsNeverImported(t *testing.T) {
	directory := t.TempDir()
	state := core.NewState()
	state.Blocks[0].Timestamp = 1783900800
	state.Blocks[0].Hash = state.Blocks[0].ComputeHash()
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "chain.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found, err := New(directory).LoadLegacyState(); err == nil || found || !strings.Contains(err.Error(), core.NetworkID) {
		t.Fatalf("testnet chain import = found %v, err %v", found, err)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil || string(unchanged) != string(data) {
		t.Fatalf("rejected testnet chain was modified: %v", err)
	}
}

func TestLegacyProbeDoesNotCreateOrReplaceFiles(t *testing.T) {
	directory := t.TempDir()
	storage := New(directory)
	if state, found, err := storage.LoadLegacyState(); err != nil || found || state != nil {
		t.Fatalf("missing legacy state = (%v, %v, %v)", state, found, err)
	}
	if wallet, found, err := storage.LoadLegacyWallet(); err != nil || found || wallet != nil {
		t.Fatalf("missing legacy wallet = (%v, %v, %v)", wallet, found, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("legacy probe created files: %v", entries)
	}
}

func TestArchiveLegacyPreservesMigratedFile(t *testing.T) {
	directory := t.TempDir()
	storage := New(directory)
	path := filepath.Join(directory, "peers.json")
	if err := os.WriteFile(path, []byte("[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := storage.ArchiveLegacy("peers.json"); err != nil {
		t.Fatal(err)
	}
	backup := path + ".migrated.bak"
	if data, err := os.ReadFile(backup); err != nil || string(data) != "[]\n" {
		t.Fatalf("archived peers = %q, %v", data, err)
	}
	if err := os.WriteFile(path, []byte("[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := storage.ArchiveLegacy("peers.json"); err == nil {
		t.Fatal("archive overwrote an existing migration backup")
	}
	if err := storage.ArchiveLegacy("../outside"); err == nil {
		t.Fatal("archive accepted an unsupported path")
	}
	if err := storage.ArchiveLegacy("wallet.json"); err == nil {
		t.Fatal("archive accepted a plaintext wallet")
	}
}

func TestRemoveMigratedWalletOnlyRemovesPlaintextWallet(t *testing.T) {
	directory := t.TempDir()
	storage := New(directory)
	path := filepath.Join(directory, "wallet.json")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := storage.RemoveMigratedWallet(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("plaintext wallet still exists: %v", err)
	}
	if err := storage.RemoveMigratedWallet(); err != nil {
		t.Fatalf("repeated removal should be harmless: %v", err)
	}
}

func TestLegacyWalletReaderRejectsUnsafeFiles(t *testing.T) {
	directory := t.TempDir()
	storage := New(directory)
	path := filepath.Join(directory, "wallet.json")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := storage.LoadLegacyWallet(); err == nil {
		t.Fatal("wallet directory was treated as a missing wallet")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{} {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := storage.LoadLegacyWallet(); err == nil {
		t.Fatal("wallet with trailing JSON was accepted")
	}
	oversized := make([]byte, maxLegacyWalletBytes+1)
	if err := os.WriteFile(path, oversized, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := storage.LoadLegacyWallet(); err == nil {
		t.Fatal("oversized wallet was accepted")
	}
}
