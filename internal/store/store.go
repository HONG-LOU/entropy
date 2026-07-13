package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"entropy/internal/core"
)

type Store struct {
	directory string
}

func New(directory string) *Store {
	return &Store{directory: directory}
}

func DefaultDirectory() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(base, core.ChainName), nil
}

func (s *Store) Directory() string {
	return s.directory
}

func (s *Store) LoadOrCreateState() (*core.State, error) {
	path := filepath.Join(s.directory, "chain.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		state := core.NewState()
		if err := s.SaveState(state); err != nil {
			return nil, err
		}
		return state, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read chain state: %w", err)
	}
	var state core.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode chain state: %w", err)
	}
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("validate chain state: %w", err)
	}
	return &state, nil
}

func (s *Store) SaveState(state *core.State) error {
	if err := state.Validate(); err != nil {
		return fmt.Errorf("refuse to save invalid chain state: %w", err)
	}
	return writeJSONAtomic(filepath.Join(s.directory, "chain.json"), state, 0o600)
}

func (s *Store) LoadOrCreateWallet() (*core.Wallet, error) {
	path := filepath.Join(s.directory, "wallet.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		wallet, err := core.NewWallet()
		if err != nil {
			return nil, err
		}
		if err := s.SaveWallet(wallet); err != nil {
			return nil, err
		}
		return wallet, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read wallet: %w", err)
	}
	var wallet core.Wallet
	if err := json.Unmarshal(data, &wallet); err != nil {
		return nil, fmt.Errorf("decode wallet: %w", err)
	}
	if err := wallet.Validate(); err != nil {
		return nil, fmt.Errorf("validate wallet: %w", err)
	}
	return &wallet, nil
}

func (s *Store) SaveWallet(wallet *core.Wallet) error {
	if err := wallet.Validate(); err != nil {
		return fmt.Errorf("refuse to save invalid wallet: %w", err)
	}
	return writeJSONAtomic(filepath.Join(s.directory, "wallet.json"), wallet, 0o600)
}

func (s *Store) LoadPeers() ([]string, error) {
	path := filepath.Join(s.directory, "peers.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read peers: %w", err)
	}
	var peers []string
	if err := json.Unmarshal(data, &peers); err != nil {
		return nil, fmt.Errorf("decode peers: %w", err)
	}
	return peers, nil
}

func (s *Store) SavePeers(peers []string) error {
	return writeJSONAtomic(filepath.Join(s.directory, "peers.json"), peers, 0o600)
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
