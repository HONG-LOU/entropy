//go:build windows

package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

func LaunchInstaller(path string) error {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return fmt.Errorf("find the Windows local application directory")
	}
	target := filepath.Join(localAppData, "Programs", "Entcoin", "Entcoin.exe")
	const installScript = `
$oldPid = [int]$args[0]
$installer = $args[1]
$target = $args[2]
Wait-Process -Id $oldPid -ErrorAction SilentlyContinue
$result = Start-Process -FilePath $installer -ArgumentList '/S' -Wait -PassThru
if ($result.ExitCode -eq 0 -and (Test-Path -LiteralPath $target -PathType Leaf)) {
    Start-Process -FilePath $target
}
`
	command := exec.Command(
		"powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden",
		"-Command", installScript, strconv.Itoa(os.Getpid()), path, target,
	)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if err := command.Start(); err != nil {
		return fmt.Errorf("schedule Entcoin update: %w", err)
	}
	return nil
}
