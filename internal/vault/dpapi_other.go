//go:build !windows && !linux

package vault

func LocalProtectionAvailable() bool { return false }

func newLocalProtection() (string, cipherDescriptor, error) {
	return "", cipherDescriptor{}, ErrLocalProtectionUnavailable
}

func validateLocalProtection(envelope) error {
	return ErrLocalProtectionUnavailable
}

func protectLocal(_, _ []byte, _ cipherDescriptor) ([]byte, error) {
	return nil, ErrLocalProtectionUnavailable
}

func unprotectLocal(_, _ []byte, _ cipherDescriptor) ([]byte, error) {
	return nil, ErrLocalProtectionUnavailable
}
