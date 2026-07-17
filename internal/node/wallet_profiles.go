package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"entropy/internal/core"
	"entropy/internal/store"
	"entropy/internal/vault"
)

const (
	walletProfilesDirectory   = "wallets"
	walletProfileVaultSuffix  = ".vault"
	walletProfileMarkerSuffix = ".recovery-confirmed"
)

type WalletProfile struct {
	Address     string `json:"address"`
	Active      bool   `json:"active"`
	NeedsBackup bool   `json:"needs_backup"`
}

func ensureWalletProfile(storage *store.Store, material *vault.Material) error {
	if material == nil {
		return fmt.Errorf("wallet profile material is missing")
	}
	path, err := walletProfileVaultPath(storage, material.Wallet.Address)
	if err != nil {
		return err
	}
	existing, err := vault.OpenLocal(path)
	if errors.Is(err, vault.ErrNotFound) {
		if err := vault.CreateLocal(path, material); err != nil {
			return fmt.Errorf("create wallet profile: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("open wallet profile: %w", err)
	} else {
		defer existing.Clear()
		if existing.Wallet != material.Wallet || existing.Source != material.Source || existing.Derivation != material.Derivation {
			return fmt.Errorf("wallet profile does not match active wallet %s", material.Wallet.Address)
		}
	}
	if legacyWalletRecoveryConfirmed(storage, material.Wallet.Address) &&
		!profileWalletRecoveryConfirmed(storage, material.Wallet.Address) {
		if err := writeWalletRecoveryMarker(walletProfileMarkerPath(storage, material.Wallet.Address), material.Wallet.Address); err != nil {
			return err
		}
	}
	return nil
}

func saveWalletProfile(storage *store.Store, material *vault.Material) error {
	path, err := walletProfileVaultPath(storage, material.Wallet.Address)
	if err != nil {
		return err
	}
	if err := vault.SaveLocal(path, material); err != nil {
		return fmt.Errorf("save wallet profile: %w", err)
	}
	if err := vault.SaveLocal(filepath.Join(storage.Directory(), walletVaultName), material); err != nil {
		return fmt.Errorf("activate wallet profile: %w", err)
	}
	return nil
}

func (s *Service) WalletProfiles() ([]WalletProfile, error) {
	s.walletMutationMu.Lock()
	defer s.walletMutationMu.Unlock()
	s.mu.RLock()
	if s.closing || s.material == nil {
		s.mu.RUnlock()
		return nil, fmt.Errorf("node is closed")
	}
	if s.seedMode {
		s.mu.RUnlock()
		return nil, ErrSeedModeWalletUnavailable
	}
	active := s.wallet.Address
	s.mu.RUnlock()
	return listWalletProfiles(s.store, active)
}

func listWalletProfiles(storage *store.Store, active string) ([]WalletProfile, error) {
	directory, err := ensureWalletProfilesDirectory(storage)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("list wallet profiles: %w", err)
	}
	profiles := make([]WalletProfile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), walletProfileVaultSuffix) {
			continue
		}
		address := strings.TrimSuffix(entry.Name(), walletProfileVaultSuffix)
		if err := core.ValidateAddress(address); err != nil {
			return nil, fmt.Errorf("wallet profile filename is invalid: %s", entry.Name())
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("wallet profile is not a regular file: %s", entry.Name())
		}
		profiles = append(profiles, WalletProfile{
			Address:     address,
			Active:      address == active,
			NeedsBackup: !walletRecoveryConfirmed(storage, address),
		})
	}
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Active != profiles[j].Active {
			return profiles[i].Active
		}
		return profiles[i].Address < profiles[j].Address
	})
	return profiles, nil
}

func walletProfileVaultPath(storage *store.Store, address string) (string, error) {
	if err := core.ValidateAddress(address); err != nil {
		return "", fmt.Errorf("invalid wallet profile address: %w", err)
	}
	directory, err := ensureWalletProfilesDirectory(storage)
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, address+walletProfileVaultSuffix), nil
}

func walletProfileMarkerPath(storage *store.Store, address string) string {
	return filepath.Join(storage.Directory(), walletProfilesDirectory, address+walletProfileMarkerSuffix)
}

func ensureWalletProfilesDirectory(storage *store.Store) (string, error) {
	directory := filepath.Join(storage.Directory(), walletProfilesDirectory)
	info, err := os.Lstat(directory)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(directory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("create wallet profiles directory: %w", err)
		}
		info, err = os.Lstat(directory)
	}
	if err != nil {
		return "", fmt.Errorf("inspect wallet profiles directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("wallet profiles path is not a directory")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return "", fmt.Errorf("protect wallet profiles directory: %w", err)
	}
	return directory, nil
}
