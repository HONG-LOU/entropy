//go:build !windows && !linux

package vault

import (
	"errors"
	"testing"
)

func TestLocalProtectionUnavailable(t *testing.T) {
	if LocalProtectionAvailable() {
		t.Fatal("local DPAPI protection should not be available")
	}
	if _, err := EncryptLocal(fixedMaterial(t)); !errors.Is(err, ErrLocalProtectionUnavailable) {
		t.Fatalf("EncryptLocal error = %v, want ErrLocalProtectionUnavailable", err)
	}
}
