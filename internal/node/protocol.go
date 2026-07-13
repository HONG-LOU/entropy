package node

import (
	"entropy/internal/core"
	"entropy/internal/ledger"
)

const (
	maxTransactionMessageBytes = int64(2*core.MaxTransactionBytes + 16<<10)
	maxBlockBodyBytes          = int64(2 * core.MaxBlockBytes)
	maxProtocolBytes           = maxBlockBodyBytes + 16<<10
	maxHeaderBatch             = 2_000
	maxBlockBatch              = 8
	maxPeerConnections         = 64
)

type protocolStatus struct {
	Protocol   string `json:"protocol"`
	Name       string `json:"name"`
	Symbol     string `json:"symbol"`
	Height     uint64 `json:"height"`
	TipHash    string `json:"tip_hash"`
	ChainWork  string `json:"chain_work"`
	ListenPort int    `json:"listen_port,omitempty"`
}

type headersRequest struct {
	Locator []string `json:"locator"`
	Limit   int      `json:"limit"`
}

type headersResponse struct {
	Protocol     string       `json:"protocol"`
	CommonHeight uint64       `json:"common_height"`
	CommonHash   string       `json:"common_hash"`
	Headers      []core.Block `json:"headers"`
}

type blocksRequest struct {
	Hashes []string `json:"hashes"`
}

type blocksResponse struct {
	Protocol string       `json:"protocol"`
	Blocks   []core.Block `json:"blocks"`
}

type transactionsResponse struct {
	Protocol     string             `json:"protocol"`
	Transactions []core.Transaction `json:"transactions"`
}

type gossipMessage struct {
	Type        string            `json:"type"`
	Protocol    string            `json:"protocol"`
	NodeID      string            `json:"node_id,omitempty"`
	ListenPort  int               `json:"listen_port,omitempty"`
	Status      *protocolStatus   `json:"status,omitempty"`
	Transaction *core.Transaction `json:"transaction,omitempty"`
	Block       *core.Block       `json:"block,omitempty"`
}

func statusFromTip(tip ledger.Tip, listenPort int) protocolStatus {
	return protocolStatus{
		Protocol:   ledger.ProtocolName,
		Name:       core.ChainName,
		Symbol:     core.ChainSymbol,
		Height:     tip.Height,
		TipHash:    tip.Hash,
		ChainWork:  tip.Work.String(),
		ListenPort: listenPort,
	}
}
