package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/HONG-LOU/entcoin/internal/core"
)

const (
	mainnetDataDirectoryName = "mainnet-v1"
	maxLegacyStateBytes      = 256 << 20
	maxLegacyWalletBytes     = 64 << 10
	maxLegacyPeersBytes      = 1 << 20
)

type Store struct {
	directory string
}

func New(directory string) *Store {
	return &Store{directory: directory}
}

func (s *Store) Directory() string {
	return s.directory
}

// LoadLegacyState reads the v1 JSON chain without creating a replacement when
// it is absent. Database migration relies on this distinction.
func (s *Store) LoadLegacyState() (*core.State, bool, error) {
	path := filepath.Join(s.directory, "chain.json")
	var state core.State
	found, err := readStrictJSONFile(path, maxLegacyStateBytes, &state)
	if err != nil {
		return nil, false, fmt.Errorf("read chain state: %w", err)
	}
	if !found {
		return nil, false, nil
	}
	if err := state.ValidateConfirmed(); err != nil {
		return nil, false, fmt.Errorf("legacy chain state is incompatible with %s and will not be imported: %w", core.NetworkID, err)
	}
	if len(state.Pending) > core.MaxPendingTransactions {
		return nil, false, fmt.Errorf("validate chain state: pending transaction limit exceeded")
	}
	return &state, true, nil
}

func (s *Store) SaveLegacyState(state *core.State) error {
	if err := state.Validate(); err != nil {
		return fmt.Errorf("refuse to save invalid chain state: %w", err)
	}
	return writeJSONAtomic(filepath.Join(s.directory, "chain.json"), state, 0o600)
}

// LoadLegacyWallet reads wallet.json without ever generating a new key.
func (s *Store) LoadLegacyWallet() (*core.Wallet, bool, error) {
	path := filepath.Join(s.directory, "wallet.json")
	var wallet core.Wallet
	found, err := readStrictJSONFile(path, maxLegacyWalletBytes, &wallet)
	if err != nil {
		return nil, false, fmt.Errorf("read wallet: %w", err)
	}
	if !found {
		return nil, false, nil
	}
	if err := wallet.Validate(); err != nil {
		return nil, false, fmt.Errorf("validate wallet: %w", err)
	}
	return &wallet, true, nil
}

func (s *Store) SaveLegacyWallet(wallet *core.Wallet) error {
	if err := wallet.Validate(); err != nil {
		return fmt.Errorf("refuse to save invalid wallet: %w", err)
	}
	return writeJSONAtomic(filepath.Join(s.directory, "wallet.json"), wallet, 0o600)
}

func (s *Store) LoadPeers() ([]string, error) {
	peers, _, err := s.LoadLegacyPeers()
	return peers, err
}

// LoadLegacyPeers reads the v1 peer list without creating it.
func (s *Store) LoadLegacyPeers() ([]string, bool, error) {
	path := filepath.Join(s.directory, "peers.json")
	var peers []string
	found, err := readStrictJSONFile(path, maxLegacyPeersBytes, &peers)
	if err != nil {
		return nil, false, fmt.Errorf("read peers: %w", err)
	}
	if !found {
		return []string{}, false, nil
	}
	return peers, true, nil
}

func (s *Store) SaveLegacyPeers(peers []string) error {
	return writeJSONAtomic(filepath.Join(s.directory, "peers.json"), peers, 0o600)
}

// ArchiveLegacy renames a successfully migrated v1 data file and refuses to
// overwrite an earlier backup.
func (s *Store) ArchiveLegacy(name string) error {
	if name != "chain.json" && name != "peers.json" {
		return fmt.Errorf("unsupported legacy file %q", name)
	}
	source := filepath.Join(s.directory, name)
	destination := source + ".migrated.bak"
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect legacy %s: %w", name, err)
	}
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("legacy backup already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect legacy backup %s: %w", name, err)
	}
	if err := os.Rename(source, destination); err != nil {
		return fmt.Errorf("archive migrated %s: %w", name, err)
	}
	return nil
}

// RemoveMigratedWallet deletes the legacy plaintext wallet only after the
// caller has independently verified local and portable encrypted copies.
func (s *Store) RemoveMigratedWallet() error {
	path := filepath.Join(s.directory, "wallet.json")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect migrated wallet: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refuse to remove non-regular legacy wallet")
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove migrated plaintext wallet: %w", err)
	}
	return nil
}

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect temporary file: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", filepath.Base(path), err)
	}
	return nil
}

func readStrictJSONFile(path string, maximum int64, value any) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	if info.Size() <= 0 || info.Size() > maximum {
		return false, fmt.Errorf("%s size is outside the allowed range", filepath.Base(path))
	}
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return false, err
	}
	if int64(len(data)) > maximum {
		return false, fmt.Errorf("%s exceeds the allowed size", filepath.Base(path))
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return false, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return false, fmt.Errorf("%s contains trailing JSON", filepath.Base(path))
		}
		return false, err
	}
	return true, nil
}
