//go:build !windows

package store

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockFile(file *os.File) (func() error, error) {
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, err
	}
	return func() error {
		return unix.Flock(int(file.Fd()), unix.LOCK_UN)
	}, nil
}
