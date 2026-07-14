//go:build windows

package store

import (
	"fmt"
	"os"
	"path/filepath"

	"entropy/internal/core"
)

// DefaultDirectory is network-scoped. Testnet data under either historical
// Entropy root is deliberately never selected for a mainnet process.
func DefaultDirectory() (string, error) {
	localBase, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find local application data directory: %w", err)
	}
	return filepath.Join(localBase, core.ChainName, mainnetDataDirectoryName), nil
}
