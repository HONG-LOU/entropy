//go:build !linux && !windows

package updater

import (
	"fmt"
	"runtime"
)

func LaunchInstaller(string) error {
	return fmt.Errorf("automatic updates do not support %s", runtime.GOOS)
}
