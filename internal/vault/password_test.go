package vault

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"entropy/internal/core"
)

var testPassword = []byte("correct horse battery staple")

func fixedMaterial(t *testing.T) *Material {
	t.Helper()
	material, err := RestoreMnemonic(zeroEntropyMnemonic)
	if err != nil {
		t.Fatalf("restore fixed mnemonic: %v", err)
	}
	return material
}

func fastEncryptBackup(t *testing.T, material *Material, password []byte) []byte {
	t.Helper()
	data, err := encryptBackupWithParams(material, password, argonParams{
		time: 2, memoryKiB: 32 * 1024, threads: 1,
	})
	if err != nil {
		t.Fatalf("encrypt backup: %v", err)
	}
	return data
}

func TestPasswordBackupRoundTripAndContainsNoPlaintextSecret(t *testing.T) {
	material := fixedMaterial(t)
	data := fastEncryptBackup(t, material, testPassword)
	if bytes.Contains(data, []byte(material.Mnemonic)) || bytes.Contains(data, []byte(material.Wallet.PrivateKey)) {
		t.Fatal("encrypted backup contains plaintext key material")
	}
	opened, err := DecryptBackup(data, testPassword)
	if err != nil {
		t.Fatalf("decrypt backup: %v", err)
	}
	if !sameWallet(opened.Wallet, material.Wallet) || opened.Mnemonic != material.Mnemonic {
		t.Fatal("decrypted material differs from original")
	}
}

func TestPasswordBackupRejectsWrongPasswordAndCiphertextTampering(t *testing.T) {
	data := fastEncryptBackup(t, fixedMaterial(t), testPassword)
	if _, err := DecryptBackup(data, []byte("this password is wrong")); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("wrong password error = %v, want ErrAuthentication", err)
	}

	var document envelope
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(document.Ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext[len(ciphertext)-1] ^= 0x80
	document.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	tampered, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptBackup(tampered, testPassword); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("tampered ciphertext error = %v, want ErrAuthentication", err)
	}
}

func TestPasswordBackupAuthenticatesPublicDescriptor(t *testing.T) {
	data := fastEncryptBackup(t, fixedMaterial(t), testPassword)
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
	if _, err := DecryptBackup(tampered, testPassword); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("tampered descriptor error = %v, want ErrAuthentication", err)
	}
}

func TestPasswordBackupRejectsHostileKDFBeforeAllocation(t *testing.T) {
	data := fastEncryptBackup(t, fixedMaterial(t), testPassword)
	var document envelope
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	document.KDF.MemoryKiB = 4 * 1024 * 1024
	document.KDF.Time = 1_000_000
	hostile, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptBackup(hostile, testPassword); !errors.Is(err, ErrInvalidVault) {
		t.Fatalf("hostile KDF error = %v, want ErrInvalidVault", err)
	}
}

func TestPasswordBackupRejectsMalformedEnvelope(t *testing.T) {
	data := fastEncryptBackup(t, fixedMaterial(t), testPassword)
	unknownField := bytes.Replace(data, []byte("{\n"), []byte("{\n  \"unknown\": true,\n"), 1)
	duplicateField := bytes.Replace(data, []byte("\"version\": 1,"), []byte("\"version\": 1, \"version\": 1,"), 1)
	for _, candidate := range [][]byte{
		nil,
		[]byte("not json"),
		append(append([]byte(nil), data...), []byte(` {"extra":true}`)...),
		unknownField,
		duplicateField,
		bytes.Repeat([]byte("x"), maxEnvelopeBytes+1),
	} {
		if _, err := DecryptBackup(candidate, testPassword); !errors.Is(err, ErrInvalidVault) {
			t.Fatalf("malformed envelope error = %v, want ErrInvalidVault", err)
		}
	}
}

func TestPasswordPolicy(t *testing.T) {
	material := fixedMaterial(t)
	if _, err := EncryptBackup(material, []byte("too short")); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("short password error = %v, want ErrWeakPassword", err)
	}
	if _, err := DecryptBackup([]byte("broken"), []byte("too short")); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("short decrypt password error = %v, want ErrWeakPassword", err)
	}
	if _, err := encryptBackupWithParams(material, testPassword, argonParams{time: 100, memoryKiB: 32 * 1024, threads: 1}); !errors.Is(err, ErrInvalidVault) {
		t.Fatalf("invalid Argon params error = %v, want ErrInvalidVault", err)
	}
}

func TestPasswordKDFHasSingleConcurrentMemoryBudget(t *testing.T) {
	passwordKDFSlot <- struct{}{}
	defer func() { <-passwordKDFSlot }()
	if _, err := EncryptBackup(fixedMaterial(t), testPassword); !errors.Is(err, ErrKDFBusy) {
		t.Fatalf("busy KDF error = %v, want ErrKDFBusy", err)
	}
}

func TestBackupFileRoundTripAndFailedExportPreservesExistingFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "wallet.entwallet")
	material := fixedMaterial(t)
	if err := ExportBackup(path, material, testPassword); err != nil {
		t.Fatalf("export backup: %v", err)
	}
	opened, err := ImportBackup(path, testPassword)
	if err != nil {
		t.Fatalf("import backup: %v", err)
	}
	if !sameWallet(opened.Wallet, material.Wallet) {
		t.Fatal("imported wallet differs")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ExportBackup(path, material, []byte("short")); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("failed export error = %v, want ErrWeakPassword", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed export changed existing backup")
	}
	missing := filepath.Join(directory, "missing.entwallet")
	if _, err := ImportBackup(missing, testPassword); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing import error = %v, want ErrNotFound", err)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing import created a file: %v", err)
	}
}

func TestLegacyMaterialPortableBackupRoundTrip(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	material, err := FromLegacy(wallet)
	if err != nil {
		t.Fatal(err)
	}
	data := fastEncryptBackup(t, material, testPassword)
	if strings.Contains(string(data), wallet.PrivateKey) {
		t.Fatal("legacy backup contains plaintext private key")
	}
	opened, err := DecryptBackup(data, testPassword)
	if err != nil {
		t.Fatalf("decrypt legacy backup: %v", err)
	}
	if !sameWallet(opened.Wallet, *wallet) || opened.Mnemonic != "" || opened.Source != SourceLegacy {
		t.Fatal("legacy backup did not preserve wallet and source")
	}
}

func TestMigrateLegacyBackupPreservesAddressAndRefusesOverwrite(t *testing.T) {
	wallet, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "legacy.entwallet")
	migrated, err := MigrateLegacyBackup(path, wallet, testPassword)
	if err != nil {
		t.Fatalf("migrate legacy backup: %v", err)
	}
	if !sameWallet(migrated.Wallet, *wallet) || migrated.Mnemonic != "" {
		t.Fatal("legacy backup migration changed wallet or invented mnemonic")
	}
	other, err := core.NewWallet()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateLegacyBackup(path, other, testPassword); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate backup migration error = %v, want ErrAlreadyExists", err)
	}
	opened, err := ImportBackup(path, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	if !sameWallet(opened.Wallet, *wallet) {
		t.Fatal("failed migration replaced the original backup")
	}
}

func FuzzDecryptBackupNeverPanics(f *testing.F) {
	f.Add([]byte("not a wallet"), []byte("long enough fuzz password"))
	f.Add([]byte("{}"), []byte("long enough fuzz password"))
	f.Fuzz(func(t *testing.T, data, password []byte) {
		if len(data) > maxEnvelopeBytes+1 || len(password) > maxPasswordBytes+1 {
			t.Skip()
		}
		_, _ = DecryptBackup(data, password)
	})
}
