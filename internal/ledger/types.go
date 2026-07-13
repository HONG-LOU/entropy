package ledger

import (
	"encoding/json"
	"math/big"
	"time"

	"entropy/internal/core"
)

const (
	SchemaVersion = 2
	DatabaseName  = "entropy.db"
	ProtocolName  = "entropy-testnet-v3"
)

type Tip struct {
	Height     uint64
	Hash       string
	Timestamp  int64
	Difficulty uint8
	Work       *big.Int
}

type UTXORecord struct {
	TxID          string `json:"tx_id"`
	OutputIndex   uint32 `json:"output_index"`
	Amount        uint64 `json:"amount"`
	Address       string `json:"address"`
	CreatedHeight uint64 `json:"created_height"`
	Coinbase      bool   `json:"coinbase"`
}

func (u UTXORecord) Output() core.TxOutput {
	return core.TxOutput{Amount: u.Amount, Address: u.Address}
}

type UndoRecord struct {
	Spent   []UTXORecord    `json:"spent"`
	Created []core.Outpoint `json:"created"`
}

type TransactionRecord struct {
	ID            string           `json:"id"`
	BlockHeight   *uint64          `json:"block_height,omitempty"`
	BlockHash     string           `json:"block_hash,omitempty"`
	Position      int              `json:"position"`
	Coinbase      bool             `json:"coinbase"`
	Pending       bool             `json:"pending"`
	Pruned        bool             `json:"pruned"`
	Confirmations uint64           `json:"confirmations"`
	Timestamp     int64            `json:"timestamp"`
	Received      uint64           `json:"received"`
	Sent          uint64           `json:"sent"`
	Transaction   core.Transaction `json:"transaction"`
}

type PeerRecord struct {
	URL         string    `json:"url"`
	Manual      bool      `json:"manual"`
	LastSeen    time.Time `json:"last_seen,omitempty"`
	Failures    int       `json:"failures"`
	NextAttempt time.Time `json:"next_attempt,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
}

type HealthEvent struct {
	ID       int64     `json:"id"`
	Code     string    `json:"code"`
	Severity string    `json:"severity"`
	Message  string    `json:"message"`
	Action   string    `json:"action,omitempty"`
	Created  time.Time `json:"created"`
	Resolved bool      `json:"resolved"`
}

func encodeJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

func decodeJSON(data []byte, value any) error {
	return json.Unmarshal(data, value)
}
