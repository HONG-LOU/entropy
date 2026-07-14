//go:build linux

package main

import (
	"os"
	"testing"

	"entropy/internal/vault"
)

func TestMain(m *testing.M) {
	vault.UseMemoryKeyringForTests()
	os.Exit(m.Run())
}
