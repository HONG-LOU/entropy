package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"entropy/internal/ledger"
	"entropy/internal/store"
	"entropy/internal/vault"
)

var ErrWalletChangedDuringBackup = errors.New("wallet changed while backup was being exported")
var ErrProtectedWalletBackupPath = errors.New("wallet backup destination conflicts with node data")

func MigrateLegacyWallet(dataDirectory, backupPath, password string) error {
	if dataDirectory == "" {
		var err error
		dataDirectory, err = store.DefaultDirectory()
		if err != nil {
			return err
		}
	}
	lock, err := store.LockDirectory(dataDirectory)
	if err != nil {
		return err
	}
	defer lock.Close()
	secret := []byte(password)
	defer clear(secret)
	return migrateLegacyWallet(store.New(dataDirectory), backupPath, secret)
}

func (s *Service) RecoveryPhrase() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closing || s.material == nil {
		return "", fmt.Errorf("node is closed")
	}
	if s.seedMode {
		return "", ErrSeedModeWalletUnavailable
	}
	if s.material.Mnemonic == "" {
		return "", fmt.Errorf("this migrated legacy wallet has no recovery phrase; export an encrypted backup")
	}
	return s.material.Mnemonic, nil
}

func (s *Service) ConfirmWalletRecovery() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing || s.material == nil {
		return fmt.Errorf("node is closed")
	}
	if s.seedMode {
		return ErrSeedModeWalletUnavailable
	}
	return s.confirmWalletRecoveryLocked(s.walletGeneration, s.wallet.Address)
}

func (s *Service) ExportWalletBackup(path, password string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("backup path is required")
	}
	if err := validateWalletBackupDestination(path, s.store.Directory()); err != nil {
		return err
	}
	if strings.ToLower(filepath.Ext(path)) != ".entwallet" {
		path += ".entwallet"
	}
	if err := validateWalletBackupDestination(path, s.store.Directory()); err != nil {
		return err
	}
	s.mu.Lock()
	if s.closing || s.material == nil {
		s.mu.Unlock()
		return fmt.Errorf("node is closed")
	}
	if s.seedMode {
		s.mu.Unlock()
		return ErrSeedModeWalletUnavailable
	}
	s.wait.Add(1)
	material := *s.material
	generation := s.walletGeneration
	address := s.wallet.Address
	s.mu.Unlock()
	defer s.wait.Done()
	defer material.Clear()
	secret := []byte(password)
	defer clear(secret)
	if err := vault.ExportBackup(path, &material, secret); err != nil {
		return err
	}
	return s.confirmWalletRecovery(generation, address)
}

func validateWalletBackupDestination(path, dataDirectory string) error {
	destination, err := normalizedWalletPath(path)
	if err != nil {
		return fmt.Errorf("resolve wallet backup destination: %w", err)
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
		protected, err := normalizedWalletPath(filepath.Join(dataDirectory, name))
		if err != nil {
			return fmt.Errorf("resolve protected node path %s: %w", name, err)
		}
		if walletPathsEqual(destination, protected) {
			return fmt.Errorf("%w: %s", ErrProtectedWalletBackupPath, name)
		}
	}
	return nil
}

func normalizedWalletPath(path string) (string, error) {
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}

func walletPathsEqual(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func (s *Service) confirmWalletRecovery(generation uint64, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing || s.material == nil {
		return fmt.Errorf("node is closed")
	}
	if s.seedMode {
		return ErrSeedModeWalletUnavailable
	}
	return s.confirmWalletRecoveryLocked(generation, address)
}

func (s *Service) confirmWalletRecoveryLocked(generation uint64, address string) error {
	if s.walletGeneration != generation || s.wallet.Address != address {
		return fmt.Errorf("%w; export the current wallet again", ErrWalletChangedDuringBackup)
	}
	if err := markWalletRecoveryConfirmed(s.store, address); err != nil {
		return err
	}
	s.walletNeedsBackup = false
	return nil
}

func (s *Service) RestoreWalletBackup(path, password string) (string, error) {
	if err := s.persistentWalletOperationAllowed(); err != nil {
		return "", err
	}
	secret := []byte(password)
	defer clear(secret)
	material, err := vault.ImportBackup(path, secret)
	if err != nil {
		return "", err
	}
	defer func() {
		if material != nil {
			material.Clear()
		}
	}()
	if err := s.replaceWallet(material, true); err != nil {
		return "", err
	}
	address := material.Wallet.Address
	material = nil
	return address, nil
}

func (s *Service) RestoreWalletMnemonic(phrase string) (string, error) {
	if err := s.persistentWalletOperationAllowed(); err != nil {
		return "", err
	}
	material, err := vault.RestoreMnemonic(phrase)
	if err != nil {
		return "", err
	}
	defer func() {
		if material != nil {
			material.Clear()
		}
	}()
	if err := s.replaceWallet(material, false); err != nil {
		return "", err
	}
	address := material.Wallet.Address
	material = nil
	return address, nil
}

func (s *Service) replaceWallet(material *vault.Material, recoveryConfirmed bool) error {
	if material == nil {
		return fmt.Errorf("replacement wallet is missing")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing || s.material == nil {
		return fmt.Errorf("node is closed")
	}
	if s.seedMode {
		return ErrSeedModeWalletUnavailable
	}
	if s.mining || s.miningJobs > 0 {
		return fmt.Errorf("stop mining before restoring a wallet")
	}
	if s.walletNeedsBackup {
		return fmt.Errorf("secure the current wallet recovery phrase or export a backup before replacing it")
	}
	path := filepath.Join(s.store.Directory(), walletVaultName)
	if err := vault.SaveLocal(path, material); err != nil {
		return err
	}
	previous := s.material
	s.material = material
	s.wallet = material.Wallet
	s.walletGeneration++
	s.walletNeedsBackup = !recoveryConfirmed
	if recoveryConfirmed {
		if err := markWalletRecoveryConfirmed(s.store, material.Wallet.Address); err != nil {
			s.walletNeedsBackup = true
		}
	} else {
		_ = removeWalletRecoveryMarker(s.store)
	}
	if previous != nil {
		previous.Clear()
	}
	return nil
}

func (s *Service) persistentWalletOperationAllowed() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closing || s.material == nil {
		return fmt.Errorf("node is closed")
	}
	if s.seedMode {
		return ErrSeedModeWalletUnavailable
	}
	return nil
}

func removeWalletRecoveryMarker(storage *store.Store) error {
	path := filepath.Join(storage.Directory(), walletRecoveryMarker)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
