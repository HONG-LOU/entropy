//go:build windows

package updater

import (
	"fmt"
	"os/exec"
)

func LaunchInstaller(path string) error {
	if err := exec.Command(path).Start(); err != nil {
		return fmt.Errorf("start Entcoin installer: %w", err)
	}
	return nil
}
