//go:build !windows

package vault

func LocalProtectionAvailable() bool { return false }

func protectLocal(_, _ []byte) ([]byte, error) {
	return nil, ErrLocalProtectionUnavailable
}

func unprotectLocal(_, _ []byte) ([]byte, error) {
	return nil, ErrLocalProtectionUnavailable
}
