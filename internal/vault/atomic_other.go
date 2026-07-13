//go:build !windows

package vault

import "os"

func installAtomic(temporaryPath, destination string, overwrite bool) error {
	if overwrite {
		return os.Rename(temporaryPath, destination)
	}
	if err := os.Link(temporaryPath, destination); err != nil {
		return err
	}
	return os.Remove(temporaryPath)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
