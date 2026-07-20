//go:build windows

package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/HONG-LOU/entcoin/internal/core"
)

// DefaultDirectory is network-scoped and keeps existing mainnet installations
// on their original path so a product rename cannot hide a user's wallet.
func DefaultDirectory() (string, error) {
	localBase, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find local application data directory: %w", err)
	}
	current := filepath.Join(localBase, core.ProductName, mainnetDataDirectoryName)
	legacy := filepath.Join(localBase, core.ChainName, mainnetDataDirectoryName)
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
