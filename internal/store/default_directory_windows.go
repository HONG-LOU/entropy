//go:build windows

package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"entropy/internal/core"
)

var defaultNodeDataNames = []string{
	"wallet.vault",
	"wallet.json",
	"wallet.recovery-confirmed",
	"entropy.db",
	"chain.json",
}

// DefaultDirectory keeps existing v0.1/v0.2 data in the roaming location but
// places clean installations under LocalAppData, where a growing live SQLite
// database will not be replicated as profile configuration.
func DefaultDirectory() (string, error) {
	localBase, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find local application data directory: %w", err)
	}
	roamingBase, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find roaming application data directory: %w", err)
	}
	local := filepath.Join(localBase, core.ChainName)
	roaming := filepath.Join(roamingBase, core.ChainName)
	localHasData, err := directoryContainsNodeData(local)
	if err != nil {
		return "", err
	}
	if localHasData || strings.EqualFold(local, roaming) {
		return local, nil
	}
	roamingHasData, err := directoryContainsNodeData(roaming)
	if err != nil {
		return "", err
	}
	if roamingHasData {
		return roaming, nil
	}
	return local, nil
}

func directoryContainsNodeData(directory string) (bool, error) {
	for _, name := range defaultNodeDataNames {
		path := filepath.Join(directory, name)
		info, err := os.Lstat(path)
		if err == nil {
			if !info.Mode().IsRegular() {
				return false, fmt.Errorf("default node data %s is not a regular file", path)
			}
			return true, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("inspect default node data %s: %w", path, err)
		}
	}
	return false, nil
}
