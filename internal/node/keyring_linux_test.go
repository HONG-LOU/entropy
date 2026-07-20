//go:build linux

package node

import (
	"os"
	"testing"

	"github.com/HONG-LOU/entcoin/internal/vault"
)

func TestMain(m *testing.M) {
	vault.UseMemoryKeyringForTests()
	os.Exit(m.Run())
}
