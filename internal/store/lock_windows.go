//go:build windows

package store

import (
	"os"

	"golang.org/x/sys/windows"
)

func lockFile(file *os.File) (func() error, error) {
	handle := windows.Handle(file.Fd())
	overlapped := new(windows.Overlapped)
	if err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlapped); err != nil {
		return nil, err
	}
	return func() error {
		return windows.UnlockFileEx(handle, 0, 1, 0, overlapped)
	}, nil
}
