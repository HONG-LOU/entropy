package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"entropy/internal/core"
	"entropy/internal/ledger"
	"entropy/internal/store"
	"entropy/internal/vault"
)

const walletVaultName = "wallet.vault"
const walletRecoveryMarker = "wallet.recovery-confirmed"
const walletRecoveryMarkerVersion = "entropy-wallet-recovery-v2"

var ErrLegacyWalletMigrationRequired = errors.New("legacy wallet requires encrypted migration")
var ErrSeedModeWalletUnavailable = errors.New("wallet transactions and mining are unavailable in seed mode")

type walletLoadState struct {
	Created           bool
	LegacyNeedsBackup bool
}

func loadSeedMaterial(storage *store.Store) (*vault.Material, walletLoadState, error) {
	for _, name := range []string{walletVaultName, walletRecoveryMarker, walletProfilesDirectory, "wallet.json"} {
		_, err := os.Lstat(filepath.Join(storage.Directory(), name))
		if err == nil {
			return nil, walletLoadState{}, fmt.Errorf("seed mode refuses persistent wallet artifact %s", name)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, walletLoadState{}, fmt.Errorf("inspect seed wallet artifact %s: %w", name, err)
		}
	}
	wallet, err := core.NewWallet()
	if err != nil {
		return nil, walletLoadState{}, fmt.Errorf("create ephemeral seed identity: %w", err)
	}
	material, err := vault.FromLegacy(wallet)
	if err != nil {
		return nil, walletLoadState{}, fmt.Errorf("validate ephemeral seed identity: %w", err)
	}
	return material, walletLoadState{}, nil
}

func loadWalletMaterial(storage *store.Store, existingNodeData bool) (*vault.Material, walletLoadState, error) {
	vaultPath := filepath.Join(storage.Directory(), walletVaultName)
	material, err := vault.OpenLocal(vaultPath)
	if err == nil {
		legacy, found, legacyErr := matchingLegacyWallet(storage, material)
		if legacyErr != nil {
			clearWalletMaterial(material)
			return nil, walletLoadState{}, legacyErr
		}
		if found && legacy != nil {
			clearWalletMaterial(material)
			return nil, walletLoadState{LegacyNeedsBackup: true}, ErrLegacyWalletMigrationRequired
		}
		if err := ensureWalletProfile(storage, material); err != nil {
			clearWalletMaterial(material)
			return nil, walletLoadState{}, err
		}
		return material, walletLoadState{}, nil
	}
	if !errors.Is(err, vault.ErrNotFound) {
		return nil, walletLoadState{}, fmt.Errorf("open protected wallet: %w", err)
	}

	_, found, err := storage.LoadLegacyWallet()
	if err != nil {
		return nil, walletLoadState{}, err
	}
	if found {
		return nil, walletLoadState{LegacyNeedsBackup: true}, ErrLegacyWalletMigrationRequired
	}
	if existingNodeData {
		return nil, walletLoadState{}, fmt.Errorf("protected wallet is missing; restore wallet.vault, a .entwallet backup, or the 24-word recovery phrase")
	}
	if !vault.LocalProtectionAvailable() {
		return nil, walletLoadState{}, vault.ErrLocalProtectionUnavailable
	}
	material, err = vault.NewMnemonic()
	if err != nil {
		return nil, walletLoadState{}, err
	}
	if err := vault.CreateLocal(vaultPath, material); err != nil {
		clearWalletMaterial(material)
		return nil, walletLoadState{}, fmt.Errorf("create protected wallet: %w", err)
	}
	if err := ensureWalletProfile(storage, material); err != nil {
		clearWalletMaterial(material)
		return nil, walletLoadState{}, err
	}
	return material, walletLoadState{Created: true}, nil
}

func matchingLegacyWallet(storage *store.Store, material *vault.Material) (*core.Wallet, bool, error) {
	legacy, found, err := storage.LoadLegacyWallet()
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	if legacy.Address != material.Wallet.Address || legacy.PublicKey != material.Wallet.PublicKey {
		return nil, false, fmt.Errorf("protected wallet does not match the remaining legacy wallet")
	}
	return legacy, true, nil
}

func migrateLegacyWallet(storage *store.Store, backupPath string, password []byte) error {
	legacy, found, err := storage.LoadLegacyWallet()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("legacy wallet.json was not found")
	}
	vaultPath := filepath.Join(storage.Directory(), walletVaultName)
	material, err := vault.MigrateLegacy(vaultPath, backupPath, legacy, password)
	if err != nil {
		return fmt.Errorf("migrate legacy wallet: %w", err)
	}
	defer material.Clear()
	// Write the identity-bound marker before removing plaintext. A crash before
	// removal remains fail-closed and can resume through MigrateLegacy; a crash
	// after removal leaves a verified recovery marker for this exact wallet.
	if err := markWalletRecoveryConfirmed(storage, legacy.Address); err != nil {
		return err
	}
	if err := storage.RemoveMigratedWallet(); err != nil {
		return err
	}
	return nil
}

func nodeDataExists(directory string) (bool, error) {
	for _, name := range []string{ledger.DatabaseName, "chain.json"} {
		info, err := os.Lstat(filepath.Join(directory, name))
		if err == nil {
			if !info.Mode().IsRegular() {
				return false, fmt.Errorf("existing node data %s is not a regular file", name)
			}
			return true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("inspect existing node data %s: %w", name, err)
		}
	}
	return false, nil
}

func clearWalletMaterial(material *vault.Material) {
	if material == nil {
		return
	}
	material.Wallet.PrivateKey = ""
	material.Mnemonic = ""
}

func walletRecoveryConfirmed(storage *store.Store, address string) bool {
	return profileWalletRecoveryConfirmed(storage, address) || legacyWalletRecoveryConfirmed(storage, address)
}

func profileWalletRecoveryConfirmed(storage *store.Store, address string) bool {
	return walletRecoveryMarkerMatches(walletProfileMarkerPath(storage, address), address)
}

func legacyWalletRecoveryConfirmed(storage *store.Store, address string) bool {
	return walletRecoveryMarkerMatches(filepath.Join(storage.Directory(), walletRecoveryMarker), address)
}

func walletRecoveryMarkerMatches(path, address string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 128 {
		return false
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return string(contents) == walletRecoveryMarkerContents(address)
}

func markWalletRecoveryConfirmed(storage *store.Store, address string) error {
	if err := core.ValidateAddress(address); err != nil {
		return fmt.Errorf("record wallet recovery confirmation: %w", err)
	}
	if _, err := ensureWalletProfilesDirectory(storage); err != nil {
		return err
	}
	if err := writeWalletRecoveryMarker(walletProfileMarkerPath(storage, address), address); err != nil {
		return err
	}
	return writeWalletRecoveryMarker(filepath.Join(storage.Directory(), walletRecoveryMarker), address)
}

func writeWalletRecoveryMarker(path, address string) error {
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("wallet recovery marker is not a regular file")
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect wallet recovery marker: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("record wallet recovery confirmation: %w", err)
	}
	if _, err := file.WriteString(walletRecoveryMarkerContents(address)); err != nil {
		_ = file.Close()
		return fmt.Errorf("write wallet recovery confirmation: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync wallet recovery confirmation: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close wallet recovery confirmation: %w", err)
	}
	return nil
}

func walletRecoveryMarkerContents(address string) string {
	return walletRecoveryMarkerVersion + " " + address + "\n"
}
