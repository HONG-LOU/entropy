//go:build linux

package main

import (
	"os"
	"testing"

	"github.com/HONG-LOU/entcoin/internal/vault"
)

func TestMain(m *testing.M) {
	vault.UseMemoryKeyringForTests()
	os.Exit(m.Run())
}
