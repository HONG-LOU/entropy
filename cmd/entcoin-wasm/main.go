//go:build js && wasm

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"syscall/js"

	"github.com/HONG-LOU/entcoin/internal/core"
	"github.com/HONG-LOU/entcoin/internal/vault"
)

type request struct {
	Action   string       `json:"action"`
	Password string       `json:"password"`
	Mnemonic string       `json:"mnemonic"`
	Backup   string       `json:"backup"`
	To       string       `json:"to"`
	Amount   string       `json:"amount"`
	Fee      string       `json:"fee"`
	UTXOs    []walletUTXO `json:"utxos"`
}

type walletUTXO struct {
	TxID        string `json:"tx_id"`
	OutputIndex uint32 `json:"output_index"`
	Amount      uint64 `json:"amount"`
	Address     string `json:"address"`
}

type response struct {
	OK          bool              `json:"ok"`
	Error       string            `json:"error,omitempty"`
	Address     string            `json:"address,omitempty"`
	Mnemonic    string            `json:"mnemonic,omitempty"`
	Backup      string            `json:"backup,omitempty"`
	Transaction *core.Transaction `json:"transaction,omitempty"`
}

var unlocked *vault.Material

func main() {
	js.Global().Set("entcoinWasmCall", js.FuncOf(call))
	select {}
}

func call(_ js.Value, args []js.Value) (output any) {
	result := response{}
	defer func() {
		if recovered := recover(); recovered != nil {
			output = marshalResponse(response{Error: "钱包处理器发生内部错误"})
		}
	}()
	if len(args) != 1 || args[0].Type() != js.TypeString {
		return marshalResponse(response{Error: "无效的钱包请求"})
	}
	var input request
	if err := json.Unmarshal([]byte(args[0].String()), &input); err != nil {
		return marshalResponse(response{Error: "无法解析钱包请求"})
	}
	result = handle(input)
	output = marshalResponse(result)
	return output
}

func handle(input request) response {
	switch input.Action {
	case "create":
		material, err := vault.NewMnemonic()
		if err != nil {
			return failed(err)
		}
		return installMaterial(material, input.Password, true)
	case "restore":
		material, err := vault.RestoreMnemonic(input.Mnemonic)
		if err != nil {
			return failed(err)
		}
		return installMaterial(material, input.Password, false)
	case "unlock":
		data, err := base64.StdEncoding.DecodeString(input.Backup)
		if err != nil {
			return response{Error: "加密钱包数据无效"}
		}
		material, err := vault.DecryptBackup(data, []byte(input.Password))
		clearBytes(data)
		if err != nil {
			return failed(err)
		}
		replaceUnlocked(material)
		return response{OK: true, Address: material.Wallet.Address}
	case "lock":
		replaceUnlocked(nil)
		return response{OK: true}
	case "export":
		if unlocked == nil {
			return response{Error: "钱包尚未解锁"}
		}
		data, err := vault.EncryptBackup(unlocked, []byte(input.Password))
		if err != nil {
			return failed(err)
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		clearBytes(data)
		return response{OK: true, Address: unlocked.Wallet.Address, Backup: encoded}
	case "validate_address":
		if err := core.ValidateAddress(input.To); err != nil {
			return failed(err)
		}
		return response{OK: true}
	case "sign":
		return signTransaction(input)
	default:
		return response{Error: "未知的钱包操作"}
	}
}

func installMaterial(material *vault.Material, password string, includeMnemonic bool) response {
	data, err := vault.EncryptBackup(material, []byte(password))
	if err != nil {
		material.Clear()
		return failed(err)
	}
	replaceUnlocked(material)
	result := response{OK: true, Address: material.Wallet.Address, Backup: base64.StdEncoding.EncodeToString(data)}
	clearBytes(data)
	if includeMnemonic {
		result.Mnemonic = material.Mnemonic
	}
	return result
}

func signTransaction(input request) response {
	if unlocked == nil {
		return response{Error: "钱包尚未解锁"}
	}
	amount, err := strconv.ParseUint(input.Amount, 10, 64)
	if err != nil {
		return response{Error: "转账金额无效"}
	}
	fee, err := strconv.ParseUint(input.Fee, 10, 64)
	if err != nil {
		return response{Error: "手续费无效"}
	}
	utxo := make(core.UTXO, len(input.UTXOs))
	for _, output := range input.UTXOs {
		utxo[core.Outpoint{TxID: output.TxID, Index: output.OutputIndex}] = core.TxOutput{Amount: output.Amount, Address: output.Address}
	}
	transaction, err := core.BuildTransaction(&unlocked.Wallet, input.To, amount, fee, utxo)
	if err != nil {
		return failed(err)
	}
	return response{OK: true, Address: unlocked.Wallet.Address, Transaction: &transaction}
}

func replaceUnlocked(material *vault.Material) {
	if unlocked != nil {
		unlocked.Clear()
	}
	unlocked = material
}

func failed(err error) response {
	return response{Error: fmt.Sprintf("%v", err)}
}

func marshalResponse(result response) string {
	data, err := json.Marshal(result)
	if err != nil {
		return `{"ok":false,"error":"无法编码钱包响应"}`
	}
	return string(data)
}

func clearBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
