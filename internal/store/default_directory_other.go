//go:build !windows

package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HONG-LOU/entcoin/internal/core"
)

func DefaultDirectory() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	current := filepath.Join(base, core.ProductName, mainnetDataDirectoryName)
	legacy := filepath.Join(base, core.ChainName, mainnetDataDirectoryName)
	if _, err := os.Stat(current); err == nil {
		return current, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect application data directory: %w", err)
	}
	if _, err := os.Stat(legacy); err == nil {
		return legacy, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect legacy application data directory: %w", err)
	}
	return current, nil
}
