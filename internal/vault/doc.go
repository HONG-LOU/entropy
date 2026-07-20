// Package vault provides versioned protection and recovery for Entcoin's
// P-256 wallets.
//
// Mnemonic derivation version 1 uses 256 bits of entropy encoded as 24 English
// BIP39 words. The BIP39 passphrase is fixed to empty. Candidate private
// scalars are HMAC-SHA256(seed, ASCII(DerivationMnemonicP256V1) || counter),
// where counter is an unsigned 32-bit big-endian integer beginning at zero.
// The first candidate d for which 0 < d < P-256.N is selected. This is an
// Entcoin-specific derivation, not BIP32 or SLIP-0010; the constant and test
// vector form part of the recovery contract.
//
// Local vaults use Windows user-scope DPAPI or Linux Secret Service with
// XChaCha20-Poly1305 and open without an application password for the same OS
// account. Portable .entwallet backups use Argon2id and XChaCha20-Poly1305 and
// can be opened cross-platform.
package vault
