package vault

import (
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"entropy/internal/core"

	"github.com/tyler-smith/go-bip39"
	"github.com/tyler-smith/go-bip39/wordlists"
)

const mnemonicEntropyBits = 256

var mnemonicMu sync.Mutex

// NewMnemonic creates a wallet backed by a new 24-word English BIP39 phrase.
func NewMnemonic() (*Material, error) {
	mnemonicMu.Lock()
	defer mnemonicMu.Unlock()
	useEnglishWordList()
	entropy, err := bip39.NewEntropy(mnemonicEntropyBits)
	if err != nil {
		return nil, fmt.Errorf("generate recovery entropy: %w", err)
	}
	defer clear(entropy)
	phrase, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, fmt.Errorf("generate recovery phrase: %w", err)
	}
	return restoreMnemonicLocked(phrase)
}

// RestoreMnemonic deterministically restores a P-256 wallet from a 24-word
// English BIP39 phrase using DerivationMnemonicP256V1.
func RestoreMnemonic(phrase string) (*Material, error) {
	mnemonicMu.Lock()
	defer mnemonicMu.Unlock()
	useEnglishWordList()
	return restoreMnemonicLocked(phrase)
}

func restoreMnemonicLocked(phrase string) (*Material, error) {
	normalized := strings.ToLower(strings.Join(strings.Fields(phrase), " "))
	if len(strings.Fields(normalized)) != 24 || !bip39.IsMnemonicValid(normalized) {
		return nil, ErrInvalidMnemonic
	}
	seed, err := bip39.NewSeedWithErrorChecking(normalized, "")
	if err != nil {
		return nil, ErrInvalidMnemonic
	}
	defer clear(seed)
	privateKey, err := deriveP256Scalar(seed)
	if err != nil {
		return nil, err
	}
	defer clear(privateKey)
	wallet, err := walletFromPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	return &Material{
		Wallet:     wallet,
		Mnemonic:   normalized,
		Source:     SourceMnemonic,
		Derivation: DerivationMnemonicP256V1,
	}, nil
}

// FromLegacy validates and wraps an existing P-256 core wallet so it can be
// protected without changing its address. Legacy material has no recovery
// phrase; an encrypted backup is therefore mandatory.
func FromLegacy(wallet *core.Wallet) (*Material, error) {
	if wallet == nil {
		return nil, fmt.Errorf("%w: legacy wallet is nil", ErrInvalidVault)
	}
	if err := wallet.Validate(); err != nil {
		return nil, fmt.Errorf("%w: invalid legacy wallet: %v", ErrInvalidVault, err)
	}
	return &Material{
		Wallet:     *wallet,
		Source:     SourceLegacy,
		Derivation: DerivationLegacyP256V1,
	}, nil
}

func deriveP256Scalar(seed []byte) ([]byte, error) {
	curveOrder := elliptic.P256().Params().N
	for counter := uint32(0); ; counter++ {
		mac := hmac.New(sha256.New, seed)
		_, _ = mac.Write([]byte(DerivationMnemonicP256V1))
		var counterBytes [4]byte
		binary.BigEndian.PutUint32(counterBytes[:], counter)
		_, _ = mac.Write(counterBytes[:])
		candidate := mac.Sum(nil)
		value := new(big.Int).SetBytes(candidate)
		if value.Sign() > 0 && value.Cmp(curveOrder) < 0 {
			return candidate, nil
		}
		clear(candidate)
		if counter == ^uint32(0) {
			return nil, fmt.Errorf("derive P-256 wallet: exhausted derivation counter")
		}
	}
}

func walletFromPrivateKey(privateKey []byte) (core.Wallet, error) {
	if len(privateKey) != 32 {
		return core.Wallet{}, fmt.Errorf("%w: invalid P-256 private key length", ErrInvalidVault)
	}
	value := new(big.Int).SetBytes(privateKey)
	curve := elliptic.P256()
	if value.Sign() <= 0 || value.Cmp(curve.Params().N) >= 0 {
		return core.Wallet{}, fmt.Errorf("%w: invalid P-256 private key", ErrInvalidVault)
	}
	x, y := curve.ScalarBaseMult(privateKey)
	publicKey := elliptic.Marshal(curve, x, y)
	wallet := core.Wallet{
		PrivateKey: hex.EncodeToString(privateKey),
		PublicKey:  hex.EncodeToString(publicKey),
		Address:    core.AddressFromPublicKey(publicKey),
	}
	if err := wallet.Validate(); err != nil {
		return core.Wallet{}, fmt.Errorf("%w: derived wallet is invalid: %v", ErrInvalidVault, err)
	}
	return wallet, nil
}

func walletFromPrivateKeyHex(value string) (core.Wallet, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return core.Wallet{}, fmt.Errorf("%w: invalid encoded private key", ErrInvalidVault)
	}
	defer clear(decoded)
	return walletFromPrivateKey(decoded)
}

func useEnglishWordList() {
	bip39.SetWordList(wordlists.English)
}
