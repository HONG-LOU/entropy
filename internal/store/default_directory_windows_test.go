//go:build windows

package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDirectoryPrefersLocalButPreservesRoamingData(t *testing.T) {
	root := t.TempDir()
	localBase := filepath.Join(root, "local")
	roamingBase := filepath.Join(root, "roaming")
	t.Setenv("LOCALAPPDATA", localBase)
	t.Setenv("APPDATA", roamingBase)

	wantLocal := filepath.Join(localBase, "Entropy")
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
	if err != nil || directory != wantRoaming {
		t.Fatalf("legacy default directory = %q, err %v, want %q", directory, err, wantRoaming)
	}
	if err := os.MkdirAll(wantLocal, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wantLocal, "entropy.db"), []byte("current"), 0o600); err != nil {
		t.Fatal(err)
	}
	directory, err = DefaultDirectory()
	if err != nil || directory != wantLocal {
		t.Fatalf("current default directory = %q, err %v, want %q", directory, err, wantLocal)
	}
}
