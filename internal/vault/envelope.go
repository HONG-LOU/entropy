package vault

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"entropy/internal/core"
)

const (
	protectionPassword = "password-v1"
	protectionDPAPI    = "windows-dpapi-user-v1"

	passwordCipher = "xchacha20-poly1305"
	dpapiCipher    = "windows-dpapi"

	maxEnvelopeBytes = 64 << 10
	maxSecretBytes   = 16 << 10
	maxDPAPIBlob     = maxSecretBytes + 4096
)

type envelope struct {
	Format     string           `json:"format"`
	Version    uint32           `json:"version"`
	Network    string           `json:"network"`
	Protection string           `json:"protection"`
	Source     KeySource        `json:"key_source"`
	Derivation string           `json:"derivation"`
	Address    string           `json:"address"`
	PublicKey  string           `json:"public_key"`
	KDF        *kdfDescriptor   `json:"kdf,omitempty"`
	Cipher     cipherDescriptor `json:"cipher"`
	Ciphertext string           `json:"ciphertext"`
}

type envelopeHeader struct {
	Format     string           `json:"format"`
	Version    uint32           `json:"version"`
	Network    string           `json:"network"`
	Protection string           `json:"protection"`
	Source     KeySource        `json:"key_source"`
	Derivation string           `json:"derivation"`
	Address    string           `json:"address"`
	PublicKey  string           `json:"public_key"`
	KDF        *kdfDescriptor   `json:"kdf,omitempty"`
	Cipher     cipherDescriptor `json:"cipher"`
}

type kdfDescriptor struct {
	Algorithm string `json:"algorithm"`
	Version   uint32 `json:"version"`
	Time      uint32 `json:"time"`
	MemoryKiB uint32 `json:"memory_kib"`
	Threads   uint8  `json:"threads"`
	KeyBytes  uint32 `json:"key_bytes"`
	Salt      string `json:"salt"`
}

type cipherDescriptor struct {
	Algorithm string `json:"algorithm"`
	Nonce     string `json:"nonce,omitempty"`
}

type secretPayload struct {
	Version    uint32    `json:"version"`
	Source     KeySource `json:"key_source"`
	Derivation string    `json:"derivation"`
	Mnemonic   string    `json:"mnemonic,omitempty"`
	PrivateKey string    `json:"private_key,omitempty"`
}

func newEnvelope(material *Material, protection string) (envelope, error) {
	if err := material.Validate(); err != nil {
		return envelope{}, err
	}
	return envelope{
		Format:     FormatName,
		Version:    FormatVersion,
		Network:    NetworkID,
		Protection: protection,
		Source:     material.Source,
		Derivation: material.Derivation,
		Address:    material.Wallet.Address,
		PublicKey:  material.Wallet.PublicKey,
	}, nil
}

func (e envelope) header() envelopeHeader {
	return envelopeHeader{
		Format:     e.Format,
		Version:    e.Version,
		Network:    e.Network,
		Protection: e.Protection,
		Source:     e.Source,
		Derivation: e.Derivation,
		Address:    e.Address,
		PublicKey:  e.PublicKey,
		KDF:        e.KDF,
		Cipher:     e.Cipher,
	}
}

func (e envelope) associatedData() ([]byte, error) {
	data, err := json.Marshal(e.header())
	if err != nil {
		return nil, fmt.Errorf("encode wallet vault header: %w", err)
	}
	return data, nil
}

func (e envelope) validateHeader(expectedProtection string) error {
	if e.Format != FormatName || e.Network != NetworkID {
		return fmt.Errorf("%w: wrong format or network", ErrInvalidVault)
	}
	if e.Version != FormatVersion {
		return fmt.Errorf("%w: envelope version %d", ErrUnsupportedVersion, e.Version)
	}
	if e.Protection != expectedProtection {
		return fmt.Errorf("%w: expected %s protection", ErrInvalidVault, expectedProtection)
	}
	if err := validateSourceAndDerivation(e.Source, e.Derivation); err != nil {
		return err
	}
	if err := core.ValidateAddress(e.Address); err != nil {
		return fmt.Errorf("%w: invalid public address", ErrInvalidVault)
	}
	publicKey, err := hex.DecodeString(e.PublicKey)
	if err != nil || len(publicKey) != 65 || core.AddressFromPublicKey(publicKey) != e.Address {
		return fmt.Errorf("%w: invalid public wallet descriptor", ErrInvalidVault)
	}
	if e.Ciphertext == "" || len(e.Ciphertext) > base64.StdEncoding.EncodedLen(maxDPAPIBlob) {
		return fmt.Errorf("%w: invalid ciphertext size", ErrInvalidVault)
	}
	return nil
}

func validateSourceAndDerivation(source KeySource, derivation string) error {
	switch source {
	case SourceMnemonic:
		if derivation != DerivationMnemonicP256V1 {
			return fmt.Errorf("%w: mnemonic derivation %q", ErrUnsupportedVersion, derivation)
		}
	case SourceLegacy:
		if derivation != DerivationLegacyP256V1 {
			return fmt.Errorf("%w: legacy derivation %q", ErrUnsupportedVersion, derivation)
		}
	default:
		return fmt.Errorf("%w: key source %q", ErrUnsupportedVersion, source)
	}
	return nil
}

func payloadFor(material *Material) (secretPayload, error) {
	if err := material.Validate(); err != nil {
		return secretPayload{}, err
	}
	payload := secretPayload{
		Version:    PayloadVersion,
		Source:     material.Source,
		Derivation: material.Derivation,
	}
	switch material.Source {
	case SourceMnemonic:
		payload.Mnemonic = material.Mnemonic
	case SourceLegacy:
		payload.PrivateKey = material.Wallet.PrivateKey
	default:
		return secretPayload{}, fmt.Errorf("%w: key source %q", ErrUnsupportedVersion, material.Source)
	}
	return payload, nil
}

func encodePayload(material *Material) ([]byte, error) {
	payload, err := payloadFor(material)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode wallet secret: %w", err)
	}
	if len(data) > maxSecretBytes {
		clear(data)
		return nil, fmt.Errorf("%w: secret payload is too large", ErrInvalidVault)
	}
	return data, nil
}

func decodePayload(data []byte) (*Material, error) {
	if len(data) == 0 || len(data) > maxSecretBytes {
		return nil, fmt.Errorf("%w: invalid secret payload size", ErrInvalidVault)
	}
	var payload secretPayload
	if err := decodeStrictJSON(data, &payload); err != nil {
		return nil, fmt.Errorf("%w: invalid secret payload", ErrInvalidVault)
	}
	if payload.Version != PayloadVersion {
		return nil, fmt.Errorf("%w: payload version %d", ErrUnsupportedVersion, payload.Version)
	}
	if err := validateSourceAndDerivation(payload.Source, payload.Derivation); err != nil {
		return nil, err
	}
	var material *Material
	var err error
	switch payload.Source {
	case SourceMnemonic:
		if payload.PrivateKey != "" {
			return nil, fmt.Errorf("%w: mnemonic payload includes a private key", ErrInvalidVault)
		}
		material, err = RestoreMnemonic(payload.Mnemonic)
	case SourceLegacy:
		if payload.Mnemonic != "" {
			return nil, fmt.Errorf("%w: legacy payload includes a mnemonic", ErrInvalidVault)
		}
		var wallet core.Wallet
		wallet, err = walletFromPrivateKeyHex(payload.PrivateKey)
		if err == nil {
			material, err = FromLegacy(&wallet)
		}
	}
	if err != nil {
		return nil, err
	}
	return material, nil
}

func verifyDescriptor(e envelope, material *Material) error {
	if material.Source != e.Source || material.Derivation != e.Derivation ||
		material.Wallet.Address != e.Address || material.Wallet.PublicKey != e.PublicKey {
		return fmt.Errorf("%w: public descriptor does not match encrypted wallet", ErrInvalidVault)
	}
	return nil
}

func marshalEnvelope(e envelope) ([]byte, error) {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode wallet vault: %w", err)
	}
	data = append(data, '\n')
	if len(data) > maxEnvelopeBytes {
		return nil, fmt.Errorf("%w: envelope is too large", ErrInvalidVault)
	}
	return data, nil
}

func unmarshalEnvelope(data []byte) (envelope, error) {
	if len(data) == 0 || len(data) > maxEnvelopeBytes {
		return envelope{}, fmt.Errorf("%w: invalid envelope size", ErrInvalidVault)
	}
	var result envelope
	if err := decodeStrictJSON(data, &result); err != nil {
		return envelope{}, fmt.Errorf("%w: malformed envelope", ErrInvalidVault)
	}
	return result, nil
}

func decodeStrictJSON(data []byte, value any) error {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var consumeValue func() error
	consumeValue = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("JSON object key is not a string")
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				seen[key] = struct{}{}
				if err := consumeValue(); err != nil {
					return err
				}
			}
			closing, err := decoder.Token()
			if err != nil || closing != json.Delim('}') {
				return fmt.Errorf("invalid JSON object")
			}
		case '[':
			for decoder.More() {
				if err := consumeValue(); err != nil {
					return err
				}
			}
			closing, err := decoder.Token()
			if err != nil || closing != json.Delim(']') {
				return fmt.Errorf("invalid JSON array")
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
		}
		return nil
	}
	if err := consumeValue(); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func readEnvelopeFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open wallet vault: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect wallet vault: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxEnvelopeBytes {
		return nil, fmt.Errorf("%w: invalid vault file", ErrInvalidVault)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxEnvelopeBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read wallet vault: %w", err)
	}
	if len(data) > maxEnvelopeBytes {
		return nil, fmt.Errorf("%w: vault file is too large", ErrInvalidVault)
	}
	return data, nil
}

func writeEnvelopeFile(path string, data []byte, overwrite bool) error {
	if len(data) == 0 || len(data) > maxEnvelopeBytes {
		return fmt.Errorf("%w: invalid encoded envelope size", ErrInvalidVault)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create wallet vault directory: %w", err)
	}
	if !overwrite {
		if _, err := os.Lstat(path); err == nil {
			return ErrAlreadyExists
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect wallet vault destination: %w", err)
		}
	}
	temporary, err := os.CreateTemp(directory, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary wallet vault: %w", err)
	}
	temporaryPath := temporary.Name()
	installed := false
	defer func() {
		_ = temporary.Close()
		if !installed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("protect temporary wallet vault: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary wallet vault: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary wallet vault: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary wallet vault: %w", err)
	}
	if err := installAtomic(temporaryPath, path, overwrite); err != nil {
		if errors.Is(err, os.ErrExist) {
			return ErrAlreadyExists
		}
		return fmt.Errorf("install wallet vault: %w", err)
	}
	installed = true
	if err := syncDirectory(directory); err != nil {
		return fmt.Errorf("sync wallet vault directory: %w", err)
	}
	return nil
}

func decodeCiphertext(value string, maximum int) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) == 0 || len(decoded) > maximum {
		return nil, fmt.Errorf("%w: invalid ciphertext encoding", ErrInvalidVault)
	}
	return decoded, nil
}

func zeroMaterial(material *Material) {
	if material == nil {
		return
	}
	material.Wallet.PrivateKey = ""
	material.Mnemonic = ""
}

func rollbackNewVault(path string, cause error) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.Join(cause, fmt.Errorf("remove unverified wallet vault: %w", err))
	}
	return cause
}
