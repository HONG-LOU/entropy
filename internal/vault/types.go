package vault

import (
	"errors"
	"fmt"

	"github.com/HONG-LOU/entcoin/internal/core"
)

const (
	FormatName     = "entropy-wallet"
	FormatVersion  = uint32(1)
	PayloadVersion = uint32(1)

	NetworkID = "entropy"

	DerivationMnemonicP256V1 = "entropy-p256-bip39-hmac-sha256-v1"
	DerivationLegacyP256V1   = "entropy-p256-direct-v1"
)

type KeySource string

const (
	SourceMnemonic KeySource = "bip39-24"
	SourceLegacy   KeySource = "legacy-p256"
)

var (
	ErrNotFound                   = errors.New("wallet vault not found")
	ErrAlreadyExists              = errors.New("wallet vault already exists")
	ErrInvalidVault               = errors.New("invalid wallet vault")
	ErrUnsupportedVersion         = errors.New("unsupported wallet vault version")
	ErrAuthentication             = errors.New("wallet vault authentication failed")
	ErrWeakPassword               = errors.New("wallet backup password does not meet policy")
	ErrKDFBusy                    = errors.New("wallet password processor is busy")
	ErrInvalidMnemonic            = errors.New("invalid 24-word recovery phrase")
	ErrLocalProtectionUnavailable = errors.New("local wallet protection is unavailable on this platform")
	ErrLocalProtectionFailure     = errors.New("local wallet protection failed")
)

// Material is the sensitive wallet value protected by a vault. Callers should
// retain it only while the wallet is unlocked.
type Material struct {
	Wallet     core.Wallet
	Mnemonic   string
	Source     KeySource
	Derivation string
}

// Clear drops the sensitive values held by Material. Go strings cannot be
// reliably wiped in place, so callers should also avoid making secret copies.
func (m *Material) Clear() {
	zeroMaterial(m)
}

func (m *Material) Validate() error {
	if m == nil {
		return fmt.Errorf("%w: wallet material is nil", ErrInvalidVault)
	}
	if err := m.Wallet.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidVault, err)
	}
	switch m.Source {
	case SourceMnemonic:
		if m.Derivation != DerivationMnemonicP256V1 {
			return fmt.Errorf("%w: mnemonic derivation %q", ErrUnsupportedVersion, m.Derivation)
		}
		restored, err := RestoreMnemonic(m.Mnemonic)
		if err != nil {
			return err
		}
		defer restored.Clear()
		if restored.Wallet != m.Wallet {
			return fmt.Errorf("%w: recovery phrase does not match wallet", ErrInvalidVault)
		}
	case SourceLegacy:
		if m.Derivation != DerivationLegacyP256V1 {
			return fmt.Errorf("%w: legacy derivation %q", ErrUnsupportedVersion, m.Derivation)
		}
		if m.Mnemonic != "" {
			return fmt.Errorf("%w: legacy wallet cannot contain a recovery phrase", ErrInvalidVault)
		}
	default:
		return fmt.Errorf("%w: unknown key source %q", ErrUnsupportedVersion, m.Source)
	}
	return nil
}

func sameWallet(left, right core.Wallet) bool {
	return left.PrivateKey == right.PrivateKey &&
		left.PublicKey == right.PublicKey &&
		left.Address == right.Address
}
