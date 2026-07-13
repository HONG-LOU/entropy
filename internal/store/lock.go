package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type DirectoryLock struct {
	file   *os.File
	unlock func() error
	once   sync.Once
	err    error
}

func LockDirectory(directory string) (*DirectoryLock, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	path := filepath.Join(directory, "node.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open data directory lock: %w", err)
	}
	unlock, err := lockFile(file)
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("data directory is already in use: %w", err)
	}
	return &DirectoryLock{file: file, unlock: unlock}, nil
}

func (l *DirectoryLock) Close() error {
	if l == nil {
		return nil
	}
	l.once.Do(func() {
		unlockErr := l.unlock()
		closeErr := l.file.Close()
		if unlockErr != nil {
			l.err = unlockErr
		} else {
			l.err = closeErr
		}
	})
	return l.err
}
