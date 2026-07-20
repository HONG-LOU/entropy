//go:build windows

package node

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HONG-LOU/entcoin/internal/core"
	"github.com/HONG-LOU/entcoin/internal/ledger"
	"github.com/HONG-LOU/entcoin/internal/store"
	"github.com/HONG-LOU/entcoin/internal/vault"
)

func TestProtectedWalletCreatedOnceAndReopened(t *testing.T) {
	directory := t.TempDir()
	storage := store.New(directory)
	material, state, err := loadWalletMaterial(storage, false)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Created || material.Mnemonic == "" {
		t.Fatal("fresh node did not create a recoverable wallet")
	}
	address := material.Wallet.Address
	clearWalletMaterial(material)

	reopened, state, err := loadWalletMaterial(storage, true)
	if err != nil {
		t.Fatal(err)
	}
	defer clearWalletMaterial(reopened)
	if state.Created || reopened.Wallet.Address != address {
		t.Fatalf("reopened wallet address = %s, created=%v", reopened.Wallet.Address, state.Created)
	}
}

func TestMissingProtectedWalletIsNotSilentlyReplaced(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ledger.DatabaseName), []byte("existing-ledger"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadWalletMaterial(store.New(directory), true); err == nil {
		t.Fatal("missing wallet was silently replaced for an existing node")
	}
	if _, err := os.Stat(filepath.Join(directory, walletVaultName)); !os.IsNotExist(err) {
		t.Fatalf("missing wallet refusal created a vault: %v", err)
	}
}

func TestLegacyWalletMigrationRequiresPortableBackup(t *testing.T) {
	directory := t.TempDir()
	storage := store.New(directory)
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveLegacyWallet(wallet); err != nil {
		t.Fatal(err)
	}
	if _, state, err := loadWalletMaterial(storage, true); !errors.Is(err, ErrLegacyWalletMigrationRequired) || !state.LegacyNeedsBackup {
		t.Fatalf("legacy load state = %+v, %v", state, err)
	}
	backupPath := filepath.Join(directory, "legacy.entwallet")
	password := []byte("correct horse battery staple")
	if err := migrateLegacyWallet(storage, backupPath, password); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "wallet.json")); !os.IsNotExist(err) {
		t.Fatalf("plaintext wallet remains after verified migration: %v", err)
	}
	material, state, err := loadWalletMaterial(storage, true)
	if err != nil {
		t.Fatal(err)
	}
	defer material.Clear()
	if state.LegacyNeedsBackup || material.Wallet.Address != wallet.Address {
		t.Fatalf("migrated wallet state = %+v, address %s", state, material.Wallet.Address)
	}
	restored, err := vault.ImportBackup(backupPath, password)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Clear()
	if restored.Wallet.Address != wallet.Address {
		t.Fatal("portable backup restored a different wallet")
	}
}

func TestLegacyMigrationResumesWithVaultBackupAndPlaintextCoexisting(t *testing.T) {
	directory := t.TempDir()
	storage := store.New(directory)
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveLegacyWallet(wallet); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(directory, "legacy.entwallet")
	password := []byte("correct horse battery staple")
	migrated, err := vault.MigrateLegacy(
		filepath.Join(directory, walletVaultName), backupPath, wallet, password,
	)
	if err != nil {
		t.Fatal(err)
	}
	migrated.Clear()

	// Simulate a crash after both encrypted copies were verified but before
	// wallet.json was removed. Startup must not silently accept this state.
	if _, state, err := loadWalletMaterial(storage, true); !errors.Is(err, ErrLegacyWalletMigrationRequired) || !state.LegacyNeedsBackup {
		t.Fatalf("coexisting migration load state = %+v, %v", state, err)
	}
	if err := migrateLegacyWallet(storage, backupPath, password); err != nil {
		t.Fatalf("resume completed migration: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "wallet.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resumed migration left plaintext wallet: %v", err)
	}
	reopened, state, err := loadWalletMaterial(storage, true)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Clear()
	if state.LegacyNeedsBackup || reopened.Wallet.Address != wallet.Address {
		t.Fatalf("resumed wallet state = %+v, address %s", state, reopened.Wallet.Address)
	}
	if !walletRecoveryConfirmed(storage, wallet.Address) {
		t.Fatal("resumed migration did not bind recovery confirmation to the migrated wallet")
	}
}

func TestRecoveryMarkerIsBoundToWalletAddress(t *testing.T) {
	storage := store.New(t.TempDir())
	first, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	second, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if err := markWalletRecoveryConfirmed(storage, first.Address); err != nil {
		t.Fatal(err)
	}
	if !walletRecoveryConfirmed(storage, first.Address) {
		t.Fatal("marker did not confirm its wallet")
	}
	if walletRecoveryConfirmed(storage, second.Address) {
		t.Fatal("marker for the previous wallet confirmed a replacement wallet")
	}
	if err := os.Remove(walletProfileMarkerPath(storage, first.Address)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storage.Directory(), walletRecoveryMarker), []byte("confirmed-v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if walletRecoveryConfirmed(storage, first.Address) {
		t.Fatal("unbound v1 marker was accepted")
	}
}

func TestServiceWalletBackupAndRestoreRoundTrip(t *testing.T) {
	directory := t.TempDir()
	service, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	defer closeTestNode(t, service)
	originalAddress := service.Address()
	phrase, err := service.RecoveryPhrase()
	if err != nil || phrase == "" {
		t.Fatalf("recovery phrase = %q, %v", phrase, err)
	}
	backup := filepath.Join(t.TempDir(), "wallet.entwallet")
	password := "correct horse battery staple"
	if err := service.ExportWalletBackup(backup, password); err != nil {
		t.Fatal(err)
	}
	dashboard, err := service.Dashboard()
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.WalletNeedsBackup {
		t.Fatal("successful encrypted backup did not clear the recovery reminder")
	}
	replacement, err := vault.NewMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	replacementPhrase := replacement.Mnemonic
	replacementAddress := replacement.Wallet.Address
	replacement.Clear()
	if restored, err := service.RestoreWalletMnemonic(replacementPhrase); err != nil || restored != replacementAddress {
		t.Fatalf("restore mnemonic = %s, %v", restored, err)
	}
	if err := service.ConfirmWalletRecovery(); err != nil {
		t.Fatal(err)
	}
	if restored, err := service.RestoreWalletBackup(backup, password); err != nil || restored != originalAddress {
		t.Fatalf("restore backup = %s, %v", restored, err)
	}
}

func TestStaleBackupSnapshotCannotConfirmReplacementWallet(t *testing.T) {
	directory := t.TempDir()
	service, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	defer closeTestNode(t, service)
	if err := service.ConfirmWalletRecovery(); err != nil {
		t.Fatal(err)
	}
	service.mu.RLock()
	staleGeneration := service.walletGeneration
	staleAddress := service.wallet.Address
	service.mu.RUnlock()

	replacement, err := vault.NewMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	replacementPhrase := replacement.Mnemonic
	replacementAddress := replacement.Wallet.Address
	replacement.Clear()
	if _, err := service.RestoreWalletMnemonic(replacementPhrase); err != nil {
		t.Fatal(err)
	}
	if err := service.confirmWalletRecovery(staleGeneration, staleAddress); !errors.Is(err, ErrWalletChangedDuringBackup) {
		t.Fatalf("stale backup confirmation error = %v, want ErrWalletChangedDuringBackup", err)
	}
	dashboard, err := service.Dashboard()
	if err != nil {
		t.Fatal(err)
	}
	if !dashboard.WalletNeedsBackup {
		t.Fatal("stale backup snapshot cleared the replacement wallet reminder")
	}
	if walletRecoveryConfirmed(store.New(directory), replacementAddress) {
		t.Fatal("stale backup snapshot confirmed the replacement wallet marker")
	}
}

func TestExportWalletBackupRejectsProtectedNodePaths(t *testing.T) {
	directory := t.TempDir()
	service, err := New(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	defer closeTestNode(t, service)
	vaultPath := filepath.Join(directory, walletVaultName)
	vaultBefore, err := os.ReadFile(vaultPath)
	if err != nil {
		t.Fatal(err)
	}
	protectedNames := []string{
		walletVaultName,
		walletRecoveryMarker,
		ledger.DatabaseName,
		ledger.DatabaseName + "-wal",
		ledger.DatabaseName + "-shm",
		"node.lock",
		"wallet.json",
		"chain.json",
		"peers.json",
		"chain.json.migrated.bak",
		"peers.json.migrated.bak",
	}
	for _, name := range protectedNames {
		name := name
		t.Run(name, func(t *testing.T) {
			destination := filepath.Join(directory, "unused", "..", strings.ToUpper(name))
			err := service.ExportWalletBackup(destination, "correct horse battery staple")
			if !errors.Is(err, ErrProtectedWalletBackupPath) {
				t.Fatalf("protected destination error = %v, want ErrProtectedWalletBackupPath", err)
			}
		})
	}
	vaultAfter, err := os.ReadFile(vaultPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(vaultAfter, vaultBefore) {
		t.Fatal("rejected backup destination modified wallet.vault")
	}
}

func TestLegacyJSONChainIsRejectedWhileWalletRemainsRecoverable(t *testing.T) {
	directory := t.TempDir()
	storage := store.New(directory)
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
	chainPath := filepath.Join(directory, "chain.json")
	chainBefore, err := os.ReadFile(chainPath)
	if err != nil {
		t.Fatal(err)
	}
	peer := "http://127.0.0.1:49001"
	if err := storage.SaveLegacyPeers([]string{peer}); err != nil {
		t.Fatal(err)
	}
	password := []byte("correct horse battery staple")
	if err := migrateLegacyWallet(storage, filepath.Join(directory, "legacy.entwallet"), password); err != nil {
		t.Fatal(err)
	}
	service, err := New(testConfig(directory))
	if service != nil {
		closeTestNode(t, service)
		t.Fatal("legacy chain unexpectedly started as mainnet")
	}
	if err == nil || !strings.Contains(err.Error(), "chain.json") || !strings.Contains(err.Error(), ledger.ProtocolName) {
		t.Fatalf("legacy chain rejection error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, ledger.DatabaseName)); !os.IsNotExist(err) {
		t.Fatalf("mainnet database was created beside legacy chain: %v", err)
	}
	chainAfter, err := os.ReadFile(chainPath)
	if err != nil || !bytes.Equal(chainAfter, chainBefore) {
		t.Fatalf("rejected legacy chain changed: %v", err)
	}
	for _, name := range []string{"chain.json.migrated.bak", "peers.json.migrated.bak"} {
		if _, err := os.Stat(filepath.Join(directory, name)); !os.IsNotExist(err) {
			t.Fatalf("legacy chain artifact %s was created: %v", name, err)
		}
	}
	recovered, err := vault.OpenLocal(filepath.Join(directory, walletVaultName))
	if err != nil {
		t.Fatalf("open independently migrated wallet: %v", err)
	}
	defer recovered.Clear()
	if recovered.Wallet.Address != wallet.Address {
		t.Fatalf("recovered wallet address = %s, want %s", recovered.Wallet.Address, wallet.Address)
	}
	peers, found, err := storage.LoadLegacyPeers()
	if err != nil || !found || len(peers) != 1 || peers[0] != peer {
		t.Fatalf("legacy peers were altered: peers %v found %v err %v", peers, found, err)
	}
}
