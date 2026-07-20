//go:build linux

package vault

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	secretServiceName    = "Entcoin"
	legacyServiceName    = "Entropy"
	secretServiceAccount = "mainnet-v1-local-wallet-key"
	localKeyBytes        = chacha20poly1305.KeySize
)

func LocalProtectionAvailable() bool { return true }

func newLocalProtection() (string, cipherDescriptor, error) {
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return "", cipherDescriptor{}, fmt.Errorf("generate local wallet nonce: %w", err)
	}
	return protectionSecretService, cipherDescriptor{
		Algorithm: secretServiceCipher,
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
	}, nil
}

func validateLocalProtection(e envelope) error {
	if err := e.validateHeader(protectionSecretService); err != nil {
		return err
	}
	if e.KDF != nil || e.Cipher.Algorithm != secretServiceCipher {
		return fmt.Errorf("%w: invalid Secret Service protection metadata", ErrInvalidVault)
	}
	if _, err := localNonce(e.Cipher); err != nil {
		return err
	}
	return nil
}

func protectLocal(plaintext, aad []byte, cipher cipherDescriptor) ([]byte, error) {
	nonce, err := localNonce(cipher)
	if err != nil {
		return nil, err
	}
	key, err := secretServiceKey(true)
	if err != nil {
		return nil, err
	}
	defer clear(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("initialize local wallet cipher: %w", err)
	}
	return aead.Seal(nil, nonce, plaintext, aad), nil
}

func unprotectLocal(ciphertext, aad []byte, cipher cipherDescriptor) ([]byte, error) {
	nonce, err := localNonce(cipher)
	if err != nil {
		return nil, err
	}
	key, err := secretServiceKey(false)
	if err != nil {
		return nil, err
	}
	defer clear(key)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("initialize local wallet cipher: %w", err)
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrAuthentication
	}
	return plaintext, nil
}

func localNonce(cipher cipherDescriptor) ([]byte, error) {
	nonce, err := base64.StdEncoding.DecodeString(cipher.Nonce)
	if err != nil || len(nonce) != chacha20poly1305.NonceSizeX {
		return nil, fmt.Errorf("%w: invalid local wallet nonce", ErrInvalidVault)
	}
	return nonce, nil
}

func secretServiceKey(create bool) ([]byte, error) {
	encoded, err := keyring.Get(secretServiceName, secretServiceAccount)
	if errors.Is(err, keyring.ErrNotFound) {
		legacy, legacyErr := keyring.Get(legacyServiceName, secretServiceAccount)
		if legacyErr == nil {
			encoded, err = legacy, nil
			if setErr := keyring.Set(secretServiceName, secretServiceAccount, legacy); setErr != nil {
				return nil, fmt.Errorf("%w: migrate wallet key in Secret Service: %v", ErrLocalProtectionFailure, setErr)
			}
		} else if !errors.Is(legacyErr, keyring.ErrNotFound) {
			return nil, fmt.Errorf("%w: open legacy wallet key from Secret Service: %v", ErrLocalProtectionFailure, legacyErr)
		}
	}
	if errors.Is(err, keyring.ErrNotFound) && create {
		key := make([]byte, localKeyBytes)
		if _, randomErr := rand.Read(key); randomErr != nil {
			return nil, fmt.Errorf("generate local wallet key: %w", randomErr)
		}
		encoded = base64.StdEncoding.EncodeToString(key)
		clear(key)
		if setErr := keyring.Set(secretServiceName, secretServiceAccount, encoded); setErr != nil {
			return nil, fmt.Errorf("%w: store wallet key in Secret Service: %v", ErrLocalProtectionFailure, setErr)
		}
		verified, getErr := keyring.Get(secretServiceName, secretServiceAccount)
		if getErr != nil || verified != encoded {
			return nil, fmt.Errorf("%w: verify wallet key in Secret Service", ErrLocalProtectionFailure)
		}
	} else if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, fmt.Errorf("%w: wallet key is missing from Secret Service", ErrLocalProtectionFailure)
		}
		return nil, fmt.Errorf("%w: open wallet key from Secret Service: %v", ErrLocalProtectionFailure, err)
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(key) != localKeyBytes {
		clear(key)
		return nil, fmt.Errorf("%w: invalid wallet key in Secret Service", ErrLocalProtectionFailure)
	}
	return key, nil
}

// UseMemoryKeyringForTests replaces Secret Service only inside a Go test
// binary. It is exported so packages that start a real node can share the
// deterministic in-memory provider on Linux CI runners.
func UseMemoryKeyringForTests() {
	if !strings.HasSuffix(filepath.Base(os.Args[0]), ".test") {
		panic("memory keyring is restricted to Go test binaries")
	}
	keyring.MockInit()
}
