//go:build windows || linux

package node

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWalletProfilesCreateSwitchAndSurviveRestart(t *testing.T) {
	directory := t.TempDir()
	service, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	originalAddress := service.Address()
	originalPhrase, err := service.RecoveryPhrase()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ConfirmWalletRecovery(); err != nil {
		t.Fatal(err)
	}
	secondAddress, err := service.CreateWallet()
	if err != nil {
		t.Fatal(err)
	}
	if secondAddress == originalAddress {
		t.Fatal("new wallet reused the active address")
	}
	profiles, err := service.WalletProfiles()
	if err != nil {
		t.Fatal(err)
	}
	assertWalletProfiles(t, profiles, secondAddress, 2)
	if !profiles[0].NeedsBackup {
		t.Fatal("new wallet was marked secured before recovery confirmation")
	}
	if err := service.ConfirmWalletRecovery(); err != nil {
		t.Fatal(err)
	}
	if err := service.SwitchWallet(originalAddress); err != nil {
		t.Fatal(err)
	}
	if phrase, err := service.RecoveryPhrase(); err != nil || phrase != originalPhrase {
		t.Fatalf("switched recovery phrase mismatch: %v", err)
	}
	closeTestNode(t, service)

	reopened, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { closeTestNode(t, reopened) })
	if reopened.Address() != originalAddress {
		t.Fatalf("reopened address = %s, want %s", reopened.Address(), originalAddress)
	}
	profiles, err = reopened.WalletProfiles()
	if err != nil {
		t.Fatal(err)
	}
	assertWalletProfiles(t, profiles, originalAddress, 2)
	for _, profile := range profiles {
		if profile.NeedsBackup {
			t.Fatalf("confirmed wallet %s lost recovery state", profile.Address)
		}
	}
}

func TestWalletProfilesRequireSecuredCurrentWalletAndProtectActiveProfile(t *testing.T) {
	service, err := New(testConfig(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer closeTestNode(t, service)
	originalAddress := service.Address()
	if _, err := service.CreateWallet(); err == nil {
		t.Fatal("unsecured current wallet was replaced")
	}
	if err := service.ConfirmWalletRecovery(); err != nil {
		t.Fatal(err)
	}
	secondAddress, err := service.CreateWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SwitchWallet(originalAddress); err == nil {
		t.Fatal("unsecured new wallet was left active")
	}
	if err := service.RemoveWallet(secondAddress); err == nil {
		t.Fatal("active wallet was removed")
	}
	if err := service.ConfirmWalletRecovery(); err != nil {
		t.Fatal(err)
	}
	if err := service.SwitchWallet(originalAddress); err != nil {
		t.Fatal(err)
	}
	if err := service.RemoveWallet(originalAddress); err == nil {
		t.Fatal("active original wallet was removed")
	}
	if err := service.RemoveWallet(secondAddress); err != nil {
		t.Fatal(err)
	}
	profiles, err := service.WalletProfiles()
	if err != nil {
		t.Fatal(err)
	}
	assertWalletProfiles(t, profiles, originalAddress, 1)
	if _, err := os.Stat(walletProfileMarkerPath(service.store, secondAddress)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("removed wallet marker remains: %v", err)
	}
}

func TestSeedModeRejectsWalletProfilesDirectory(t *testing.T) {
	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, walletProfilesDirectory), 0o700); err != nil {
		t.Fatal(err)
	}
	config := testConfig(directory)
	config.SeedMode = true
	service, err := New(config)
	if service != nil || err == nil {
		if service != nil {
			closeTestNode(t, service)
		}
		t.Fatalf("seed accepted wallet profile directory: %v", err)
	}
}

func assertWalletProfiles(t *testing.T, profiles []WalletProfile, active string, count int) {
	t.Helper()
	if len(profiles) != count {
		t.Fatalf("wallet profile count = %d, want %d: %+v", len(profiles), count, profiles)
	}
	activeCount := 0
	for _, profile := range profiles {
		if profile.Active {
			activeCount++
			if profile.Address != active {
				t.Fatalf("active wallet = %s, want %s", profile.Address, active)
			}
		}
	}
	if activeCount != 1 {
		t.Fatalf("active wallet count = %d, want 1", activeCount)
	}
}
