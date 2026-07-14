package vault

import (
	"encoding/base64"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"entropy/internal/core"
)

// EncryptLocal returns a wallet document protected by the current operating
// system user credential store. It is intentionally not portable between user
// accounts; use an encrypted backup or recovery phrase for portability.
func EncryptLocal(material *Material) ([]byte, error) {
	payload, err := encodePayload(material)
	if err != nil {
		return nil, err
	}
	defer clear(payload)
	protection, cipher, err := newLocalProtection()
	if err != nil {
		return nil, err
	}
	e, err := newEnvelope(material, protection)
	if err != nil {
		return nil, err
	}
	e.Cipher = cipher
	aad, err := e.associatedData()
	if err != nil {
		return nil, err
	}
	ciphertext, err := protectLocal(payload, aad, e.Cipher)
	if err != nil {
		return nil, err
	}
	e.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	clear(ciphertext)
	return marshalEnvelope(e)
}

// DecryptLocal automatically opens a wallet document for the operating-system
// user that created it. It never creates replacement key material on failure.
func DecryptLocal(data []byte) (*Material, error) {
	e, err := unmarshalEnvelope(data)
	if err != nil {
		return nil, err
	}
	if err := validateLocalProtection(e); err != nil {
		return nil, err
	}
	ciphertext, err := decodeCiphertext(e.Ciphertext, maxDPAPIBlob)
	if err != nil {
		return nil, err
	}
	defer clear(ciphertext)
	aad, err := e.associatedData()
	if err != nil {
		return nil, err
	}
	plaintext, err := unprotectLocal(ciphertext, aad, e.Cipher)
	if err != nil {
		return nil, err
	}
	defer clear(plaintext)
	material, err := decodePayload(plaintext)
	if err != nil {
		return nil, err
	}
	if err := verifyDescriptor(e, material); err != nil {
		zeroMaterial(material)
		return nil, err
	}
	return material, nil
}

// CreateLocal writes a new operating-system-protected vault and refuses to
// overwrite any existing path. Use this for first-run wallet creation.
func CreateLocal(path string, material *Material) error {
	data, err := EncryptLocal(material)
	if err != nil {
		return err
	}
	defer clear(data)
	return writeEnvelopeFile(path, data, false)
}

// SaveLocal atomically replaces a local vault after successful encryption.
func SaveLocal(path string, material *Material) error {
	data, err := EncryptLocal(material)
	if err != nil {
		return err
	}
	defer clear(data)
	return writeEnvelopeFile(path, data, true)
}

// OpenLocal opens an existing local vault. Missing, corrupt, or inaccessible
// files return an error and never trigger wallet generation.
func OpenLocal(path string) (*Material, error) {
	data, err := readEnvelopeFile(path)
	if err != nil {
		return nil, err
	}
	defer clear(data)
	return DecryptLocal(data)
}

// MigrateLegacyLocal protects a validated legacy wallet without changing its
// address and refuses to replace an existing vault.
func MigrateLegacyLocal(path string, wallet *core.Wallet) (*Material, error) {
	material, err := FromLegacy(wallet)
	if err != nil {
		return nil, err
	}
	if err := CreateLocal(path, material); err != nil {
		zeroMaterial(material)
		return nil, err
	}
	verified, err := OpenLocal(path)
	if err != nil {
		zeroMaterial(material)
		return nil, rollbackNewVault(path, fmt.Errorf("verify migrated wallet vault: %w", err))
	}
	defer zeroMaterial(verified)
	if !sameWallet(verified.Wallet, material.Wallet) || verified.Source != SourceLegacy {
		zeroMaterial(material)
		return nil, rollbackNewVault(path, fmt.Errorf("%w: migrated wallet vault mismatch", ErrInvalidVault))
	}
	return material, nil
}

// MigrateLegacy creates and verifies both a portable password backup and a
// local operating-system-protected vault. The caller must retain the legacy
// file unless this method succeeds. If local protection fails after backup
// creation, the verified portable backup is deliberately retained at backupPath.
func MigrateLegacy(localPath, backupPath string, wallet *core.Wallet, password []byte) (*Material, error) {
	if !LocalProtectionAvailable() {
		return nil, ErrLocalProtectionUnavailable
	}
	if strings.EqualFold(filepath.Clean(localPath), filepath.Clean(backupPath)) {
		return nil, fmt.Errorf("%w: local vault and portable backup paths must differ", ErrInvalidVault)
	}
	expected, err := FromLegacy(wallet)
	if err != nil {
		return nil, err
	}
	backupMaterial, err := ensureLegacyBackup(backupPath, expected, password)
	if err != nil {
		expected.Clear()
		return nil, err
	}
	backupMaterial.Clear()
	localMaterial, err := ensureLegacyLocal(localPath, expected)
	if err != nil {
		expected.Clear()
		return nil, fmt.Errorf("create local vault after verified portable backup: %w", err)
	}
	localMaterial.Clear()
	return expected, nil
}

func ensureLegacyBackup(path string, expected *Material, password []byte) (*Material, error) {
	// Existing files are opened and matched, never replaced. This makes the
	// combined migration restartable after a crash between its two writes.
	material, err := ImportBackup(path, password)
	if errors.Is(err, ErrNotFound) {
		material, err = MigrateLegacyBackup(path, &expected.Wallet, password)
	}
	if err != nil {
		return nil, err
	}
	if err := verifyLegacyMigration(material, expected); err != nil {
		material.Clear()
		return nil, err
	}
	return material, nil
}

func ensureLegacyLocal(path string, expected *Material) (*Material, error) {
	material, err := OpenLocal(path)
	if errors.Is(err, ErrNotFound) {
		material, err = MigrateLegacyLocal(path, &expected.Wallet)
	}
	if err != nil {
		return nil, err
	}
	if err := verifyLegacyMigration(material, expected); err != nil {
		material.Clear()
		return nil, err
	}
	return material, nil
}

func verifyLegacyMigration(actual, expected *Material) error {
	if actual == nil || expected == nil || actual.Source != SourceLegacy ||
		actual.Mnemonic != "" || !sameWallet(actual.Wallet, expected.Wallet) {
		return fmt.Errorf("%w: migrated legacy wallet does not match", ErrInvalidVault)
	}
	if actual.Derivation != DerivationLegacyP256V1 {
		return fmt.Errorf("%w: migrated legacy derivation %q", ErrUnsupportedVersion, actual.Derivation)
	}
	return nil
}
