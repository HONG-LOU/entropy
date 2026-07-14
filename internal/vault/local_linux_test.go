//go:build linux

package vault

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	UseMemoryKeyringForTests()
	os.Exit(m.Run())
}

func TestLocalSecretServiceRoundTripAndNoPlaintextSecret(t *testing.T) {
	keyring.MockInit()
	material := fixedMaterial(t)
	data, err := EncryptLocal(material)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(material.Mnemonic)) || bytes.Contains(data, []byte(material.Wallet.PrivateKey)) {
		t.Fatal("Secret Service vault contains plaintext key material")
	}
	opened, err := DecryptLocal(data)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Clear()
	if !sameWallet(opened.Wallet, material.Wallet) || opened.Mnemonic != material.Mnemonic {
		t.Fatal("Secret Service round trip changed wallet material")
	}
}

func TestLocalSecretServiceRejectsTamperingAndMissingKey(t *testing.T) {
	keyring.MockInit()
	data, err := EncryptLocal(fixedMaterial(t))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	cipher := document["cipher"].(map[string]any)
	nonce, err := base64.StdEncoding.DecodeString(cipher["nonce"].(string))
	if err != nil {
		t.Fatal(err)
	}
	nonce[0] ^= 0x01
	cipher["nonce"] = base64.StdEncoding.EncodeToString(nonce)
	tampered, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptLocal(tampered); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("tampered vault error = %v", err)
	}
	keyring.MockInit()
	if _, err := DecryptLocal(data); !errors.Is(err, ErrLocalProtectionFailure) {
		t.Fatalf("missing key error = %v", err)
	}
}

func TestLocalSecretServiceFailsClosedWhenKeyringUnavailable(t *testing.T) {
	keyring.MockInitWithError(errors.New("locked"))
	defer keyring.MockInit()
	if _, err := EncryptLocal(fixedMaterial(t)); !errors.Is(err, ErrLocalProtectionFailure) {
		t.Fatalf("locked keyring error = %v", err)
	}
}
