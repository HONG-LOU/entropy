package vault

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"entropy/internal/core"
)

const zeroEntropyMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art"

func TestMnemonicDerivationVector(t *testing.T) {
	material, err := RestoreMnemonic(zeroEntropyMnemonic)
	if err != nil {
		t.Fatalf("restore mnemonic: %v", err)
	}
	if got, want := material.Wallet.PrivateKey, "71e71445d5d88fb1a330c7bee147055fbc152ee4b9ef08573e39a7cd51b562c7"; got != want {
		t.Fatalf("private key = %s, want %s", got, want)
	}
	if got, want := material.Wallet.PublicKey, "047d60220484e7c54a085dc25436362f185752425d4052c607f2d708de9a6309f79933920aec39c11c1801eb0fb8e5f318c5725de7119fd182d13cedc36d7fd020"; got != want {
		t.Fatalf("public key = %s, want %s", got, want)
	}
	if got, want := material.Wallet.Address, "ent1bbeeed3ec25c5427b54354923895522bf5d43299b70c7bc0"; got != want {
		t.Fatalf("address = %s, want %s", got, want)
	}
	if material.Derivation != DerivationMnemonicP256V1 || material.Source != SourceMnemonic {
		t.Fatalf("unexpected derivation metadata: %+v", material)
	}
}

func TestNewMnemonicHasTwentyFourWordsAndRestores(t *testing.T) {
	material, err := NewMnemonic()
	if err != nil {
		t.Fatalf("create mnemonic wallet: %v", err)
	}
	if got := len(strings.Fields(material.Mnemonic)); got != 24 {
		t.Fatalf("word count = %d, want 24", got)
	}
	restored, err := RestoreMnemonic(material.Mnemonic)
	if err != nil {
		t.Fatalf("restore generated mnemonic: %v", err)
	}
	if !sameWallet(material.Wallet, restored.Wallet) {
		t.Fatal("restored wallet differs from generated wallet")
	}
	if err := material.Validate(); err != nil {
		t.Fatalf("validate generated material: %v", err)
	}
}

func TestMnemonicRestoreIsConcurrentSafe(t *testing.T) {
	const workers = 16
	errorsFound := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			material, err := RestoreMnemonic(zeroEntropyMnemonic)
			if err == nil {
				err = material.Validate()
				material.Clear()
			}
			errorsFound <- err
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatalf("concurrent restore: %v", err)
		}
	}
}

func TestRestoreMnemonicNormalizesWhitespaceAndCase(t *testing.T) {
	input := "  " + strings.ToUpper(strings.ReplaceAll(zeroEntropyMnemonic, " ", "  \n")) + "  "
	material, err := RestoreMnemonic(input)
	if err != nil {
		t.Fatalf("restore normalized mnemonic: %v", err)
	}
	if material.Mnemonic != zeroEntropyMnemonic {
		t.Fatalf("normalized mnemonic = %q", material.Mnemonic)
	}
}

func TestRestoreMnemonicRejectsInvalidOrNonTwentyFourWordPhrase(t *testing.T) {
	twelveWords := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	invalidChecksum := strings.TrimSuffix(zeroEntropyMnemonic, "art") + "zoo"
	for _, phrase := range []string{"", twelveWords, invalidChecksum, "not a recovery phrase"} {
		if _, err := RestoreMnemonic(phrase); !errors.Is(err, ErrInvalidMnemonic) {
			t.Fatalf("RestoreMnemonic(%q) error = %v, want ErrInvalidMnemonic", phrase, err)
		}
	}
}

func TestFromLegacyPreservesWalletWithoutInventingMnemonic(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatalf("create legacy wallet: %v", err)
	}
	material, err := FromLegacy(wallet)
	if err != nil {
		t.Fatalf("wrap legacy wallet: %v", err)
	}
	if !sameWallet(*wallet, material.Wallet) {
		t.Fatal("legacy wallet changed during migration")
	}
	if material.Mnemonic != "" || material.Source != SourceLegacy || material.Derivation != DerivationLegacyP256V1 {
		t.Fatalf("unexpected legacy metadata: %+v", material)
	}
	if err := material.Validate(); err != nil {
		t.Fatalf("validate legacy material: %v", err)
	}

	tampered := *wallet
	replacement := "0"
	if strings.HasSuffix(tampered.Address, replacement) {
		replacement = "1"
	}
	tampered.Address = material.Wallet.Address[:len(material.Wallet.Address)-1] + replacement
	if _, err := FromLegacy(&tampered); !errors.Is(err, ErrInvalidVault) {
		t.Fatalf("tampered legacy error = %v, want ErrInvalidVault", err)
	}
	if _, err := FromLegacy(nil); !errors.Is(err, ErrInvalidVault) {
		t.Fatalf("nil legacy error = %v, want ErrInvalidVault", err)
	}
}

func TestMaterialValidationRejectsSourceMismatch(t *testing.T) {
	material, err := RestoreMnemonic(zeroEntropyMnemonic)
	if err != nil {
		t.Fatal(err)
	}
	material.Source = SourceLegacy
	if err := material.Validate(); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("source mismatch error = %v, want ErrUnsupportedVersion", err)
	}
}

func TestMaterialClearDropsSecretValues(t *testing.T) {
	material := fixedMaterial(t)
	publicKey := material.Wallet.PublicKey
	address := material.Wallet.Address
	material.Clear()
	if material.Mnemonic != "" || material.Wallet.PrivateKey != "" {
		t.Fatal("Clear retained secret values")
	}
	if material.Wallet.PublicKey != publicKey || material.Wallet.Address != address {
		t.Fatal("Clear removed the public wallet descriptor")
	}
}
