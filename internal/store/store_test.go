package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStateAndWalletRoundTrip(t *testing.T) {
	directory := t.TempDir()
	storage := New(directory)
	wallet, err := storage.LoadOrCreateWallet()
	if err != nil {
		t.Fatal(err)
	}
	state, err := storage.LoadOrCreateState()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.Mine(context.Background(), wallet.Address); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveState(state); err != nil {
		t.Fatal(err)
	}
	reloaded, err := storage.LoadOrCreateState()
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Blocks) != 2 || reloaded.Blocks[1].Hash != state.Blocks[1].Hash {
		t.Fatal("reloaded chain does not match saved chain")
	}

	chainPath := filepath.Join(directory, "chain.json")
	if err := os.WriteFile(chainPath, []byte("{truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.LoadOrCreateState(); err == nil {
		t.Fatal("corrupt state was silently accepted")
	}
}
