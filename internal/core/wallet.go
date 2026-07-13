package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

type Wallet struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Address    string `json:"address"`
}

func NewWallet() (*Wallet, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate wallet: %w", err)
	}
	publicKey := elliptic.Marshal(elliptic.P256(), key.X, key.Y)
	return &Wallet{
		PrivateKey: fmt.Sprintf("%064x", key.D),
		PublicKey:  hex.EncodeToString(publicKey),
		Address:    AddressFromPublicKey(publicKey),
	}, nil
}

func AddressFromPublicKey(publicKey []byte) string {
	payloadHash := sha256.Sum256(publicKey)
	payload := payloadHash[:20]
	checksumHash := sha256.Sum256(payload)
	encoded := append(append([]byte(nil), payload...), checksumHash[:4]...)
	return "ent1" + hex.EncodeToString(encoded)
}

func ValidateAddress(address string) error {
	if len(address) != 4+48 || address[:4] != "ent1" {
		return fmt.Errorf("invalid ENT address")
	}
	decoded, err := hex.DecodeString(address[4:])
	if err != nil || len(decoded) != 24 {
		return fmt.Errorf("invalid ENT address")
	}
	checksum := sha256.Sum256(decoded[:20])
	if !equalBytes(decoded[20:], checksum[:4]) {
		return fmt.Errorf("invalid ENT address checksum")
	}
	return nil
}

func (w *Wallet) Validate() error {
	privateKey, err := w.privateKey()
	if err != nil {
		return err
	}
	publicKey := elliptic.Marshal(elliptic.P256(), privateKey.X, privateKey.Y)
	if hex.EncodeToString(publicKey) != w.PublicKey {
		return fmt.Errorf("wallet public key does not match private key")
	}
	if AddressFromPublicKey(publicKey) != w.Address {
		return fmt.Errorf("wallet address does not match public key")
	}
	return ValidateAddress(w.Address)
}

func (w *Wallet) PublicKeyBytes() ([]byte, error) {
	if err := w.Validate(); err != nil {
		return nil, err
	}
	return hex.DecodeString(w.PublicKey)
}

func (w *Wallet) privateKey() (*ecdsa.PrivateKey, error) {
	dBytes, err := hex.DecodeString(w.PrivateKey)
	if err != nil || len(dBytes) != 32 {
		return nil, fmt.Errorf("invalid wallet private key")
	}
	d := new(big.Int).SetBytes(dBytes)
	curve := elliptic.P256()
	if d.Sign() <= 0 || d.Cmp(curve.Params().N) >= 0 {
		return nil, fmt.Errorf("invalid wallet private key")
	}
	x, y := curve.ScalarBaseMult(dBytes)
	return &ecdsa.PrivateKey{PublicKey: ecdsa.PublicKey{Curve: curve, X: x, Y: y}, D: d}, nil
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var different byte
	for i := range a {
		different |= a[i] ^ b[i]
	}
	return different == 0
}
