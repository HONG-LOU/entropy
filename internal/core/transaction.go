package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/asn1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
)

type ecdsaSignature struct {
	R *big.Int
	S *big.Int
}

func NewCoinbase(address string, height, amount uint64) (Transaction, error) {
	if err := ValidateAddress(address); err != nil {
		return Transaction{}, err
	}
	nonce, err := randomUint64()
	if err != nil {
		return Transaction{}, err
	}
	tx := Transaction{
		Coinbase: true,
		Nonce:    nonce ^ height,
	}
	if amount > 0 {
		tx.Outputs = []TxOutput{{Amount: amount, Address: address}}
	}
	tx.ID = tx.ComputeID()
	return tx, nil
}

func BuildTransaction(wallet *Wallet, to string, amount, fee uint64, utxo UTXO) (Transaction, error) {
	if amount == 0 {
		return Transaction{}, fmt.Errorf("amount must be greater than zero")
	}
	if err := ValidateAddress(to); err != nil {
		return Transaction{}, err
	}
	needed, err := safeAdd(amount, fee)
	if err != nil {
		return Transaction{}, err
	}
	publicKey, err := wallet.PublicKeyBytes()
	if err != nil {
		return Transaction{}, err
	}

	type candidate struct {
		outpoint Outpoint
		output   TxOutput
	}
	candidates := make([]candidate, 0)
	for outpoint, output := range utxo {
		if AddressesEqual(output.Address, wallet.Address) {
			candidates = append(candidates, candidate{outpoint: outpoint, output: output})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].outpoint.TxID == candidates[j].outpoint.TxID {
			return candidates[i].outpoint.Index < candidates[j].outpoint.Index
		}
		return candidates[i].outpoint.TxID < candidates[j].outpoint.TxID
	})

	inputs := make([]TxInput, 0)
	previousOutputs := make([]TxOutput, 0)
	total := uint64(0)
	for _, candidate := range candidates {
		if len(inputs) >= MaxTransactionInputs {
			break
		}
		inputs = append(inputs, TxInput{
			TxID:        candidate.outpoint.TxID,
			OutputIndex: candidate.outpoint.Index,
			PublicKey:   append([]byte(nil), publicKey...),
		})
		previousOutputs = append(previousOutputs, candidate.output)
		total, err = safeAdd(total, candidate.output.Amount)
		if err != nil {
			return Transaction{}, err
		}
		if total >= needed {
			break
		}
	}
	if total < needed {
		return Transaction{}, fmt.Errorf("insufficient funds: have %s ENT, need %s ENT", FormatAmount(total), FormatAmount(needed))
	}

	nonce, err := randomUint64()
	if err != nil {
		return Transaction{}, err
	}
	tx := Transaction{
		Nonce:  nonce,
		Inputs: inputs,
		Outputs: []TxOutput{
			{Amount: amount, Address: to},
		},
	}
	if change := total - needed; change > 0 {
		tx.Outputs = append(tx.Outputs, TxOutput{Amount: change, Address: wallet.Address})
	}
	if err := tx.sign(wallet, previousOutputs); err != nil {
		return Transaction{}, err
	}
	tx.ID = tx.ComputeID()
	return tx, nil
}

func (tx Transaction) ComputeID() string {
	var e encoder
	encodeTransaction(&e, tx, true)
	return hashHex(e.Bytes())
}

func (tx Transaction) signingHash(inputIndex int, previousOutput TxOutput) []byte {
	var e encoder
	encodeTransaction(&e, tx, false)
	e.uint64(uint64(inputIndex))
	e.uint64(previousOutput.Amount)
	e.string(previousOutput.Address)
	hash, _ := hex.DecodeString(hashHex(e.Bytes()))
	return hash
}

func (tx *Transaction) sign(wallet *Wallet, previousOutputs []TxOutput) error {
	if len(previousOutputs) != len(tx.Inputs) {
		return fmt.Errorf("previous output count does not match inputs")
	}
	privateKey, err := wallet.privateKey()
	if err != nil {
		return err
	}
	for i := range tx.Inputs {
		r, s, err := ecdsa.Sign(rand.Reader, privateKey, tx.signingHash(i, previousOutputs[i]))
		if err != nil {
			return fmt.Errorf("sign input %d: %w", i, err)
		}
		halfOrder := new(big.Int).Rsh(new(big.Int).Set(privateKey.Params().N), 1)
		if s.Cmp(halfOrder) > 0 {
			s.Sub(privateKey.Params().N, s)
		}
		signature, err := asn1.Marshal(ecdsaSignature{R: r, S: s})
		if err != nil {
			return fmt.Errorf("encode input %d signature: %w", i, err)
		}
		tx.Inputs[i].Signature = signature
	}
	return nil
}

func encodeTransaction(e *encoder, tx Transaction, includeSignatures bool) {
	e.string(NetworkID)
	e.bool(tx.Coinbase)
	e.uint64(tx.Nonce)
	e.uint64(uint64(len(tx.Inputs)))
	for _, input := range tx.Inputs {
		e.string(input.TxID)
		e.uint32(input.OutputIndex)
		e.bytes(input.PublicKey)
		if includeSignatures {
			e.bytes(input.Signature)
		}
	}
	e.uint64(uint64(len(tx.Outputs)))
	for _, output := range tx.Outputs {
		e.uint64(output.Amount)
		e.string(output.Address)
	}
}

func validateRegularTransaction(tx Transaction, utxo UTXO) (uint64, error) {
	if tx.Coinbase || len(tx.Inputs) == 0 || len(tx.Outputs) == 0 {
		return 0, fmt.Errorf("regular transaction requires inputs and outputs")
	}
	if len(tx.Inputs) > MaxTransactionInputs || len(tx.Outputs) > MaxTransactionOutputs {
		return 0, fmt.Errorf("transaction exceeds input or output limit")
	}
	for index, input := range tx.Inputs {
		if _, err := decodeHash(input.TxID); err != nil {
			return 0, fmt.Errorf("input %d has invalid transaction hash", index)
		}
		if len(input.PublicKey) != 65 || len(input.Signature) == 0 || len(input.Signature) > 80 {
			return 0, fmt.Errorf("input %d has invalid key or signature size", index)
		}
	}
	if _, err := sumOutputs(tx.Outputs); err != nil {
		return 0, err
	}
	if tx.encodedSize() > MaxTransactionBytes {
		return 0, fmt.Errorf("transaction exceeds encoded size limit")
	}
	if tx.ID != tx.ComputeID() {
		return 0, fmt.Errorf("transaction ID mismatch")
	}

	inputTotal := uint64(0)
	seen := make(map[Outpoint]struct{}, len(tx.Inputs))
	previousOutputs := make([]TxOutput, len(tx.Inputs))
	for i, input := range tx.Inputs {
		outpoint := Outpoint{TxID: input.TxID, Index: input.OutputIndex}
		if _, exists := seen[outpoint]; exists {
			return 0, fmt.Errorf("duplicate input %s:%d", input.TxID, input.OutputIndex)
		}
		seen[outpoint] = struct{}{}
		previousOutput, exists := utxo[outpoint]
		if !exists {
			return 0, fmt.Errorf("input %s:%d is missing or already spent", input.TxID, input.OutputIndex)
		}
		if !AddressesEqual(AddressFromPublicKey(input.PublicKey), previousOutput.Address) {
			return 0, fmt.Errorf("input %d public key does not own output", i)
		}
		x, y := elliptic.Unmarshal(elliptic.P256(), input.PublicKey)
		if x == nil || y == nil {
			return 0, fmt.Errorf("input %d has invalid public key", i)
		}
		publicKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
		var signature ecdsaSignature
		rest, err := asn1.Unmarshal(input.Signature, &signature)
		halfOrder := new(big.Int).Rsh(new(big.Int).Set(publicKey.Params().N), 1)
		if err != nil || len(rest) != 0 || signature.R == nil || signature.S == nil || signature.R.Sign() <= 0 || signature.S.Sign() <= 0 || signature.S.Cmp(halfOrder) > 0 || !ecdsa.Verify(&publicKey, tx.signingHash(i, previousOutput), signature.R, signature.S) {
			return 0, fmt.Errorf("input %d signature is invalid", i)
		}
		inputTotal, err = safeAdd(inputTotal, previousOutput.Amount)
		if err != nil {
			return 0, err
		}
		previousOutputs[i] = previousOutput
	}

	outputTotal, err := sumOutputs(tx.Outputs)
	if err != nil {
		return 0, err
	}
	if inputTotal < outputTotal {
		return 0, fmt.Errorf("transaction spends more than its inputs")
	}
	return inputTotal - outputTotal, nil
}

func ValidateRegularTransaction(tx Transaction, utxo UTXO) (uint64, error) {
	return validateRegularTransaction(tx, utxo)
}

func validateCoinbase(tx Transaction, maximum uint64) error {
	if !tx.Coinbase || len(tx.Inputs) != 0 {
		return fmt.Errorf("invalid coinbase transaction")
	}
	if len(tx.Outputs) == 0 && maximum != 0 {
		return fmt.Errorf("coinbase transaction has no reward output")
	}
	if len(tx.Outputs) > 1 || tx.encodedSize() > MaxTransactionBytes {
		return fmt.Errorf("coinbase transaction exceeds output or size limit")
	}
	if tx.ID != tx.ComputeID() {
		return fmt.Errorf("coinbase transaction ID mismatch")
	}
	total, err := sumOutputs(tx.Outputs)
	if err != nil {
		return err
	}
	if total != maximum {
		return fmt.Errorf("coinbase reward must equal subsidy plus fees")
	}
	return nil
}

func ValidateCoinbase(tx Transaction, maximum uint64) error {
	return validateCoinbase(tx, maximum)
}

func EncodedTransactionSize(tx Transaction) int {
	return tx.encodedSize()
}

func (tx Transaction) encodedSize() int {
	var e encoder
	encodeTransaction(&e, tx, true)
	return e.Len()
}

func sumOutputs(outputs []TxOutput) (uint64, error) {
	total := uint64(0)
	for _, output := range outputs {
		if output.Amount == 0 {
			return 0, fmt.Errorf("transaction output amount must be greater than zero")
		}
		if err := ValidateAddress(output.Address); err != nil {
			return 0, err
		}
		var err error
		total, err = safeAdd(total, output.Amount)
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

func randomUint64() (uint64, error) {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return 0, fmt.Errorf("generate nonce: %w", err)
	}
	return binary.BigEndian.Uint64(value[:]), nil
}
