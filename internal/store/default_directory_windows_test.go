//go:build windows

package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDirectoryAlwaysUsesIsolatedMainnetDirectory(t *testing.T) {
	root := t.TempDir()
	localBase := filepath.Join(root, "local")
	roamingBase := filepath.Join(root, "roaming")
	t.Setenv("LOCALAPPDATA", localBase)
	t.Setenv("APPDATA", roamingBase)

	wantLocal := filepath.Join(localBase, "Entcoin", "mainnet-v1")
	directory, err := DefaultDirectory()
	if err != nil || directory != wantLocal {
		t.Fatalf("clean default directory = %q, err %v, want %q", directory, err, wantLocal)
	}
	wantRoaming := filepath.Join(roamingBase, "Entropy")
	if err := os.MkdirAll(wantRoaming, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wantRoaming, "wallet.vault"), []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	directory, err = DefaultDirectory()
	if err != nil || directory != wantLocal {
		t.Fatalf("legacy data changed mainnet directory = %q, err %v, want %q", directory, err, wantLocal)
	}
	oldLocal := filepath.Join(localBase, "Entropy")
	if err := os.MkdirAll(oldLocal, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldLocal, "entropy.db"), []byte("testnet"), 0o600); err != nil {
		t.Fatal(err)
	}
	directory, err = DefaultDirectory()
	if err != nil || directory != wantLocal {
		t.Fatalf("current default directory = %q, err %v, want %q", directory, err, wantLocal)
	}
	legacyMainnet := filepath.Join(oldLocal, "mainnet-v1")
	if err := os.MkdirAll(legacyMainnet, 0o700); err != nil {
		t.Fatal(err)
	}
	directory, err = DefaultDirectory()
	if err != nil || directory != legacyMainnet {
		t.Fatalf("legacy mainnet directory = %q, err %v, want %q", directory, err, legacyMainnet)
	}
}
