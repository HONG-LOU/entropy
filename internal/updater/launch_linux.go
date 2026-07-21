//go:build linux

package updater

import (
	"fmt"
	"os/exec"
)

func LaunchInstaller(path string) error {
	if err := exec.Command("xdg-open", path).Start(); err != nil {
		return fmt.Errorf("open system package installer: %w", err)
	}
	return nil
}
