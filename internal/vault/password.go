package vault

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"entropy/internal/core"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	argon2Version = uint32(0x13)
	argon2Name    = "argon2id"

	passwordKeyBytes      = uint32(32)
	passwordSaltBytes     = 16
	minPasswordBytes      = 12
	maxPasswordBytes      = 1024
	maxPasswordCiphertext = maxSecretBytes + chacha20poly1305.Overhead
)

type argonParams struct {
	time      uint32
	memoryKiB uint32
	threads   uint8
}

var defaultArgonParams = argonParams{time: 3, memoryKiB: 64 * 1024, threads: 2}

var passwordKDFSlot = make(chan struct{}, 1)

// EncryptBackup returns a portable password-protected .entwallet document.
func EncryptBackup(material *Material, password []byte) ([]byte, error) {
	return encryptBackupWithParams(material, password, defaultArgonParams)
}

func encryptBackupWithParams(material *Material, password []byte, params argonParams) ([]byte, error) {
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	if err := validateArgonParams(params); err != nil {
		return nil, err
	}
	payload, err := encodePayload(material)
	if err != nil {
		return nil, err
	}
	defer clear(payload)

	e, err := newEnvelope(material, protectionPassword)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate wallet backup salt: %w", err)
	}
	defer clear(salt)
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate wallet backup nonce: %w", err)
	}
	e.KDF = &kdfDescriptor{
		Algorithm: argon2Name,
		Version:   argon2Version,
		Time:      params.time,
		MemoryKiB: params.memoryKiB,
		Threads:   params.threads,
		KeyBytes:  passwordKeyBytes,
		Salt:      base64.StdEncoding.EncodeToString(salt),
	}
	e.Cipher = cipherDescriptor{
		Algorithm: passwordCipher,
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
	}
	aad, err := e.associatedData()
	if err != nil {
		return nil, err
	}
	key, err := derivePasswordKey(password, salt, params)
	if err != nil {
		return nil, err
	}
	defer clear(key)
	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("initialize wallet backup cipher: %w", err)
	}
	ciphertext := cipher.Seal(nil, nonce, payload, aad)
	e.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	clear(ciphertext)
	return marshalEnvelope(e)
}

// DecryptBackup opens a portable .entwallet document. Authentication failures
// deliberately do not distinguish a wrong password from damaged ciphertext.
func DecryptBackup(data, password []byte) (*Material, error) {
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	e, err := unmarshalEnvelope(data)
	if err != nil {
		return nil, err
	}
	if err := e.validateHeader(protectionPassword); err != nil {
		return nil, err
	}
	params, salt, nonce, err := validatePasswordEnvelope(e)
	if err != nil {
		return nil, err
	}
	defer clear(salt)
	ciphertext, err := decodeCiphertext(e.Ciphertext, maxPasswordCiphertext)
	if err != nil {
		return nil, err
	}
	defer clear(ciphertext)
	aad, err := e.associatedData()
	if err != nil {
		return nil, err
	}
	key, err := derivePasswordKey(password, salt, params)
	if err != nil {
		return nil, err
	}
	defer clear(key)
	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("initialize wallet backup cipher: %w", err)
	}
	plaintext, err := cipher.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrAuthentication
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

// ExportBackup atomically writes a portable password-protected .entwallet
// document. An existing destination is replaced only after encryption succeeds.
func ExportBackup(path string, material *Material, password []byte) error {
	data, err := EncryptBackup(material, password)
	if err != nil {
		return err
	}
	defer clear(data)
	return writeEnvelopeFile(path, data, true)
}

// CreateBackup writes a new portable backup and refuses to overwrite an
// existing file.
func CreateBackup(path string, material *Material, password []byte) error {
	data, err := EncryptBackup(material, password)
	if err != nil {
		return err
	}
	defer clear(data)
	return writeEnvelopeFile(path, data, false)
}

// ImportBackup opens a portable password-protected .entwallet file.
func ImportBackup(path string, password []byte) (*Material, error) {
	data, err := readEnvelopeFile(path)
	if err != nil {
		return nil, err
	}
	defer clear(data)
	return DecryptBackup(data, password)
}

// MigrateLegacyBackup writes and verifies a new encrypted portable backup for
// a legacy wallet without changing its P-256 key or address.
func MigrateLegacyBackup(path string, wallet *core.Wallet, password []byte) (*Material, error) {
	material, err := FromLegacy(wallet)
	if err != nil {
		return nil, err
	}
	if err := CreateBackup(path, material, password); err != nil {
		material.Clear()
		return nil, err
	}
	verified, err := ImportBackup(path, password)
	if err != nil {
		material.Clear()
		return nil, rollbackNewVault(path, fmt.Errorf("verify migrated wallet backup: %w", err))
	}
	defer verified.Clear()
	if !sameWallet(verified.Wallet, material.Wallet) || verified.Source != SourceLegacy {
		material.Clear()
		return nil, rollbackNewVault(path, fmt.Errorf("%w: migrated wallet backup mismatch", ErrInvalidVault))
	}
	return material, nil
}

func validatePassword(password []byte) error {
	if len(password) < minPasswordBytes {
		return ErrWeakPassword
	}
	if len(password) > maxPasswordBytes {
		return fmt.Errorf("%w: password exceeds %d bytes", ErrWeakPassword, maxPasswordBytes)
	}
	return nil
}

func validateArgonParams(params argonParams) error {
	if params.time < 2 || params.time > 6 ||
		params.memoryKiB < 32*1024 || params.memoryKiB > 128*1024 ||
		params.threads == 0 || params.threads > 4 {
		return fmt.Errorf("%w: Argon2id parameters are outside policy", ErrInvalidVault)
	}
	return nil
}

func derivePasswordKey(password, salt []byte, params argonParams) ([]byte, error) {
	select {
	case passwordKDFSlot <- struct{}{}:
		defer func() { <-passwordKDFSlot }()
	default:
		return nil, ErrKDFBusy
	}
	return argon2.IDKey(password, salt, params.time, params.memoryKiB, params.threads, passwordKeyBytes), nil
}

func validatePasswordEnvelope(e envelope) (argonParams, []byte, []byte, error) {
	if e.KDF == nil || e.KDF.Algorithm != argon2Name || e.KDF.Version != argon2Version ||
		e.KDF.KeyBytes != passwordKeyBytes || e.Cipher.Algorithm != passwordCipher {
		return argonParams{}, nil, nil, fmt.Errorf("%w: unsupported password protection", ErrUnsupportedVersion)
	}
	params := argonParams{time: e.KDF.Time, memoryKiB: e.KDF.MemoryKiB, threads: e.KDF.Threads}
	if err := validateArgonParams(params); err != nil {
		return argonParams{}, nil, nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(e.KDF.Salt)
	if err != nil || len(salt) != passwordSaltBytes {
		return argonParams{}, nil, nil, fmt.Errorf("%w: invalid Argon2id salt", ErrInvalidVault)
	}
	nonce, err := base64.StdEncoding.DecodeString(e.Cipher.Nonce)
	if err != nil || len(nonce) != chacha20poly1305.NonceSizeX {
		clear(salt)
		return argonParams{}, nil, nil, fmt.Errorf("%w: invalid cipher nonce", ErrInvalidVault)
	}
	return params, salt, nonce, nil
}
