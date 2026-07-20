package node

import (
	"github.com/HONG-LOU/entcoin/internal/core"
	"github.com/HONG-LOU/entcoin/internal/ledger"
)

const (
	maxTransactionMessageBytes = int64(2*core.MaxTransactionBytes + 16<<10)
	maxBlockBodyBytes          = int64(2 * core.MaxBlockBytes)
	maxProtocolBytes           = maxBlockBodyBytes + 16<<10
	maxHeaderBatch             = 2_000
	maxSyncHeaderBatch         = 128
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

type mempoolRequest struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type blocksResponse struct {
	Protocol string       `json:"protocol"`
	Blocks   []core.Block `json:"blocks"`
}

type transactionsResponse struct {
	Protocol     string             `json:"protocol"`
	Transactions []core.Transaction `json:"transactions"`
}

type peersResponse struct {
	Protocol string   `json:"protocol"`
	Peers    []string `json:"peers"`
}

type gossipMessage struct {
	Type                 string                `json:"type"`
	Protocol             string                `json:"protocol"`
	NodeID               string                `json:"node_id,omitempty"`
	ListenPort           int                   `json:"listen_port,omitempty"`
	Status               *protocolStatus       `json:"status,omitempty"`
	Transaction          *core.Transaction     `json:"transaction,omitempty"`
	Block                *core.Block           `json:"block,omitempty"`
	RequestID            string                `json:"request_id,omitempty"`
	Part                 int                   `json:"part,omitempty"`
	Parts                int                   `json:"parts,omitempty"`
	HeadersRequest       *headersRequest       `json:"headers_request,omitempty"`
	HeadersResponse      *headersResponse      `json:"headers_response,omitempty"`
	BlocksRequest        *blocksRequest        `json:"blocks_request,omitempty"`
	MempoolRequest       *mempoolRequest       `json:"mempool_request,omitempty"`
	TransactionsResponse *transactionsResponse `json:"transactions_response,omitempty"`
	ReconcileError       string                `json:"reconcile_error,omitempty"`
	ReconcileErrorCode   string                `json:"reconcile_error_code,omitempty"`
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
