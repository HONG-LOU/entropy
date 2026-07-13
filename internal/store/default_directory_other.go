//go:build !windows

package store

import (
	"fmt"
	"os"
	"path/filepath"

	"entropy/internal/core"
)

func DefaultDirectory() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(base, core.ChainName), nil
}
