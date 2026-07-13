//go:build windows

package vault

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"entropy/internal/core"
)

func TestLocalDPAPIRoundTripAndNoPlaintextSecret(t *testing.T) {
	if !LocalProtectionAvailable() {
		t.Fatal("DPAPI should be available on Windows")
	}
	material := fixedMaterial(t)
	data, err := EncryptLocal(material)
	if err != nil {
		t.Fatalf("encrypt local vault: %v", err)
	}
	if bytes.Contains(data, []byte(material.Mnemonic)) || bytes.Contains(data, []byte(material.Wallet.PrivateKey)) {
		t.Fatal("DPAPI vault contains plaintext key material")
	}
	opened, err := DecryptLocal(data)
	if err != nil {
		t.Fatalf("decrypt local vault: %v", err)
	}
	if !sameWallet(opened.Wallet, material.Wallet) || opened.Mnemonic != material.Mnemonic {
		t.Fatal("DPAPI round trip changed wallet material")
	}
}

func TestLocalDPAPISupportsMaximumSecretPayload(t *testing.T) {
	plaintext := bytes.Repeat([]byte{0x5a}, maxSecretBytes)
	aad := []byte("maximum-payload-test")
	ciphertext, err := protectLocal(plaintext, aad)
	if err != nil {
		t.Fatalf("protect maximum payload: %v", err)
	}
	if len(ciphertext) > maxDPAPIBlob {
		t.Fatalf("DPAPI blob size = %d, maximum = %d", len(ciphertext), maxDPAPIBlob)
	}
	opened, err := unprotectLocal(ciphertext, aad)
	if err != nil {
		t.Fatalf("unprotect maximum payload: %v", err)
	}
	if !bytes.Equal(opened, plaintext) {
		t.Fatal("maximum DPAPI payload changed during round trip")
	}
	clear(opened)
}

func TestLocalDPAPIRejectsTampering(t *testing.T) {
	data, err := EncryptLocal(fixedMaterial(t))
	if err != nil {
		t.Fatal(err)
	}
	var document envelope
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(document.Ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[len(ciphertext)/2] ^= 1
	document.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	tampered, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptLocal(tampered); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("tampered DPAPI ciphertext error = %v, want ErrAuthentication", err)
	}
}

func TestLocalDPAPIAuthenticatesDescriptor(t *testing.T) {
	data, err := EncryptLocal(fixedMaterial(t))
	if err != nil {
		t.Fatal(err)
	}
	other, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	var document envelope
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	document.Address = other.Address
	document.PublicKey = other.PublicKey
	tampered, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptLocal(tampered); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("tampered DPAPI descriptor error = %v, want ErrAuthentication", err)
	}
}

func TestLocalFileCreateOpenReplaceAndMissingDoesNotCreate(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "wallet.vault")
	first := fixedMaterial(t)
	if err := CreateLocal(path, first); err != nil {
		t.Fatalf("create local vault: %v", err)
	}
	if err := CreateLocal(path, first); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate create error = %v, want ErrAlreadyExists", err)
	}
	opened, err := OpenLocal(path)
	if err != nil {
		t.Fatalf("open local vault: %v", err)
	}
	if !sameWallet(opened.Wallet, first.Wallet) {
		t.Fatal("opened wallet differs")
	}

	second, err := NewMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveLocal(path, second); err != nil {
		t.Fatalf("replace local vault: %v", err)
	}
	replaced, err := OpenLocal(path)
	if err != nil {
		t.Fatal(err)
	}
	if !sameWallet(replaced.Wallet, second.Wallet) {
		t.Fatal("SaveLocal did not replace wallet")
	}

	missing := filepath.Join(directory, "missing.vault")
	if _, err := OpenLocal(missing); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing local vault error = %v, want ErrNotFound", err)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("OpenLocal created missing vault: %v", err)
	}
}

func TestCorruptLocalVaultNeverChangesFileOrCreatesWallet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wallet.vault")
	corrupt := []byte(`{"format":"entropy-wallet","version":1}`)
	if err := os.WriteFile(path, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenLocal(path); err == nil {
		t.Fatal("corrupt vault unexpectedly opened")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, corrupt) {
		t.Fatal("opening corrupt vault modified it")
	}
}

func TestMigrateLegacyLocalPreservesAddressAndRefusesOverwrite(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "wallet.vault")
	migrated, err := MigrateLegacyLocal(path, wallet)
	if err != nil {
		t.Fatalf("migrate legacy wallet: %v", err)
	}
	if !sameWallet(migrated.Wallet, *wallet) || migrated.Mnemonic != "" {
		t.Fatal("legacy migration changed wallet or invented mnemonic")
	}
	opened, err := OpenLocal(path)
	if err != nil {
		t.Fatal(err)
	}
	if !sameWallet(opened.Wallet, *wallet) || opened.Source != SourceLegacy {
		t.Fatal("opened migration differs from legacy wallet")
	}
	other, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateLegacyLocal(path, other); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("overwrite migration error = %v, want ErrAlreadyExists", err)
	}
	stillOriginal, err := OpenLocal(path)
	if err != nil {
		t.Fatal(err)
	}
	if !sameWallet(stillOriginal.Wallet, *wallet) {
		t.Fatal("failed migration replaced existing wallet")
	}
}

func TestMigrateLegacyCreatesLocalAndPortableRecovery(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	localPath := filepath.Join(directory, "wallet.vault")
	backupPath := filepath.Join(directory, "wallet.entwallet")
	migrated, err := MigrateLegacy(localPath, backupPath, wallet, testPassword)
	if err != nil {
		t.Fatalf("migrate legacy wallet and backup: %v", err)
	}
	if !sameWallet(migrated.Wallet, *wallet) {
		t.Fatal("combined migration changed legacy wallet")
	}
	local, err := OpenLocal(localPath)
	if err != nil {
		t.Fatal(err)
	}
	portable, err := ImportBackup(backupPath, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if !sameWallet(local.Wallet, *wallet) || !sameWallet(portable.Wallet, *wallet) {
		t.Fatal("combined migration recovery copies differ")
	}
	retried, err := MigrateLegacy(localPath, backupPath, wallet, testPassword)
	if err != nil {
		t.Fatalf("retry completed migration: %v", err)
	}
	if !sameWallet(retried.Wallet, *wallet) {
		t.Fatal("retry changed migrated wallet")
	}
}

func TestMigrateLegacyResumesAfterPortableBackupOnly(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	localPath := filepath.Join(directory, "wallet.vault")
	backupPath := filepath.Join(directory, "wallet.entwallet")
	backup, err := MigrateLegacyBackup(backupPath, wallet, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	backup.Clear()
	migrated, err := MigrateLegacy(localPath, backupPath, wallet, testPassword)
	if err != nil {
		t.Fatalf("resume after backup: %v", err)
	}
	if !sameWallet(migrated.Wallet, *wallet) {
		t.Fatal("resumed migration changed wallet")
	}
	if _, err := OpenLocal(localPath); err != nil {
		t.Fatalf("resumed migration did not create local vault: %v", err)
	}
}

func TestMigrateLegacyResumesAfterLocalVaultOnly(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	localPath := filepath.Join(directory, "wallet.vault")
	backupPath := filepath.Join(directory, "wallet.entwallet")
	local, err := MigrateLegacyLocal(localPath, wallet)
	if err != nil {
		t.Fatal(err)
	}
	local.Clear()
	migrated, err := MigrateLegacy(localPath, backupPath, wallet, testPassword)
	if err != nil {
		t.Fatalf("resume after local vault: %v", err)
	}
	if !sameWallet(migrated.Wallet, *wallet) {
		t.Fatal("resumed migration changed wallet")
	}
	if _, err := ImportBackup(backupPath, testPassword); err != nil {
		t.Fatalf("resumed migration did not create portable backup: %v", err)
	}
}

func TestMigrateLegacyRejectsMismatchedPartialState(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	other, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	localPath := filepath.Join(directory, "wallet.vault")
	backupPath := filepath.Join(directory, "wallet.entwallet")
	backup, err := MigrateLegacyBackup(backupPath, other, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	backup.Clear()
	if _, err := MigrateLegacy(localPath, backupPath, wallet, testPassword); !errors.Is(err, ErrInvalidVault) {
		t.Fatalf("mismatched partial migration error = %v, want ErrInvalidVault", err)
	}
	if _, err := os.Stat(localPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("mismatched migration created local vault: %v", err)
	}
	if _, err := MigrateLegacy(backupPath, backupPath, wallet, testPassword); !errors.Is(err, ErrInvalidVault) {
		t.Fatalf("same path migration error = %v, want ErrInvalidVault", err)
	}
}
