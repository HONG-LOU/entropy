package node

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"entropy/internal/core"
	"entropy/internal/store"
)

const (
	DefaultListenAddress = "0.0.0.0:47821"
	maxTransactionBytes  = 1 << 20
	maxStateBytes        = 64 << 20
)

type Config struct {
	DataDirectory string
	ListenAddress string
}

type Service struct {
	mu            sync.RWMutex
	state         *core.State
	wallet        *core.Wallet
	store         *store.Store
	peers         map[string]struct{}
	peerOnline    map[string]bool
	listenAddress string
	actualAddress string
	client        *http.Client
	server        *http.Server
	cancel        context.CancelFunc
	mining        bool
	miningCancel  context.CancelFunc
	wait          sync.WaitGroup
	lastError     string
	revision      uint64
	directoryLock *store.DirectoryLock
	syncing       map[string]bool
	requestSlots  chan struct{}
	closing       bool
}

type Dashboard struct {
	Name                string         `json:"name"`
	Symbol              string         `json:"symbol"`
	Address             string         `json:"address"`
	ConfirmedBalance    string         `json:"confirmed_balance"`
	SpendableBalance    string         `json:"spendable_balance"`
	Height              uint64         `json:"height"`
	TipHash             string         `json:"tip_hash"`
	Difficulty          uint8          `json:"difficulty"`
	PendingCount        int            `json:"pending_count"`
	PeerCount           int            `json:"peer_count"`
	ConfiguredPeerCount int            `json:"configured_peer_count"`
	Peers               []PeerSummary  `json:"peers"`
	Mining              bool           `json:"mining"`
	ListenAddress       string         `json:"listen_address"`
	Issued              string         `json:"issued"`
	MaxSupply           string         `json:"max_supply"`
	TargetBlockSeconds  int            `json:"target_block_seconds"`
	EmissionBlocks      uint64         `json:"emission_blocks"`
	NextSubsidy         string         `json:"next_subsidy"`
	LastError           string         `json:"last_error"`
	RecentBlocks        []BlockSummary `json:"recent_blocks"`
}

type BlockSummary struct {
	Height       uint64 `json:"height"`
	Hash         string `json:"hash"`
	Timestamp    int64  `json:"timestamp"`
	Transactions int    `json:"transactions"`
	Difficulty   uint8  `json:"difficulty"`
}

type PeerSummary struct {
	URL    string `json:"url"`
	Online bool   `json:"online"`
}

type peerStatus struct {
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Height  uint64 `json:"height"`
	TipHash string `json:"tip_hash"`
	Work    string `json:"work"`
}

func New(config Config) (*Service, error) {
	if config.DataDirectory == "" {
		var err error
		config.DataDirectory, err = store.DefaultDirectory()
		if err != nil {
			return nil, err
		}
	}
	if config.ListenAddress == "" {
		config.ListenAddress = DefaultListenAddress
	}
	directoryLock, err := store.LockDirectory(config.DataDirectory)
	if err != nil {
		return nil, err
	}
	keepLock := false
	defer func() {
		if !keepLock {
			_ = directoryLock.Close()
		}
	}()
	storage := store.New(config.DataDirectory)
	state, err := storage.LoadOrCreateState()
	if err != nil {
		return nil, err
	}
	wallet, err := storage.LoadOrCreateWallet()
	if err != nil {
		return nil, err
	}
	storedPeers, err := storage.LoadPeers()
	if err != nil {
		return nil, err
	}
	service := &Service{
		state:         state,
		wallet:        wallet,
		store:         storage,
		peers:         make(map[string]struct{}),
		peerOnline:    make(map[string]bool),
		listenAddress: config.ListenAddress,
		client:        &http.Client{Timeout: 5 * time.Second},
		syncing:       make(map[string]bool),
		requestSlots:  make(chan struct{}, 32),
		directoryLock: directoryLock,
	}
	for _, peer := range storedPeers {
		normalized, err := normalizePeer(peer)
		if err == nil {
			service.peers[normalized] = struct{}{}
		}
	}
	keepLock = true
	return service, nil
}

func (s *Service) Start(parent context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return fmt.Errorf("node is already running")
	}
	if s.closing {
		return fmt.Errorf("node has been closed")
	}
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.listenAddress, err)
	}
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	s.actualAddress = listener.Addr().String()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/status", s.handleStatus)
	mux.HandleFunc("GET /v1/state", s.handleGetState)
	mux.HandleFunc("POST /v1/state", s.handlePostState)
	mux.HandleFunc("POST /v1/transactions", s.handleTransaction)
	server := &http.Server{
		Handler:           s.limitRequests(mux),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	s.server = server
	s.wait.Add(2)
	go func() {
		defer s.wait.Done()
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.setError(err)
		}
	}()
	go s.syncLoop(ctx)
	return nil
}

func (s *Service) Close(ctx context.Context) error {
	s.mu.Lock()
	s.closing = true
	if s.miningCancel != nil {
		s.miningCancel()
		s.miningCancel = nil
	}
	if s.cancel != nil {
		s.cancel()
	}
	server := s.server
	s.server = nil
	directoryLock := s.directoryLock
	s.directoryLock = nil
	s.mu.Unlock()
	var err error
	if server != nil {
		err = server.Shutdown(ctx)
	}
	s.wait.Wait()
	if directoryLock != nil {
		err = errors.Join(err, directoryLock.Close())
	}
	return err
}

func (s *Service) Address() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.wallet.Address
}

func (s *Service) ActualAddress() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.actualAddress
}

func (s *Service) Dashboard() (Dashboard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	confirmed, spendable, err := s.state.Balances(s.wallet.Address)
	if err != nil {
		return Dashboard{}, err
	}
	tip := s.state.Blocks[len(s.state.Blocks)-1]
	issued := core.MintedThrough(tip.Height)
	peers := s.peerListLocked()
	peerSummaries := make([]PeerSummary, 0, len(peers))
	onlinePeers := 0
	for _, peer := range peers {
		online := s.peerOnline[peer]
		if online {
			onlinePeers++
		}
		peerSummaries = append(peerSummaries, PeerSummary{URL: peer, Online: online})
	}
	recent := make([]BlockSummary, 0, 8)
	for index := len(s.state.Blocks) - 1; index >= 0 && len(recent) < 8; index-- {
		block := s.state.Blocks[index]
		recent = append(recent, BlockSummary{
			Height:       block.Height,
			Hash:         block.Hash,
			Timestamp:    block.Timestamp,
			Transactions: len(block.Transactions),
			Difficulty:   block.Difficulty,
		})
	}
	return Dashboard{
		Name:                core.ChainName,
		Symbol:              core.ChainSymbol,
		Address:             s.wallet.Address,
		ConfirmedBalance:    core.FormatAmount(confirmed),
		SpendableBalance:    core.FormatAmount(spendable),
		Height:              tip.Height,
		TipHash:             tip.Hash,
		Difficulty:          tip.Difficulty,
		PendingCount:        len(s.state.Pending),
		PeerCount:           onlinePeers,
		ConfiguredPeerCount: len(peers),
		Peers:               peerSummaries,
		Mining:              s.mining,
		ListenAddress:       s.actualAddress,
		Issued:              core.FormatAmount(issued),
		MaxSupply:           core.FormatAmount(core.MaxSupply),
		TargetBlockSeconds:  core.TargetBlockSeconds,
		EmissionBlocks:      core.EmissionBlocks,
		NextSubsidy:         core.FormatAmount(core.Subsidy(tip.Height + 1)),
		LastError:           s.lastError,
		RecentBlocks:        recent,
	}, nil
}

func (s *Service) Send(to, amountText, feeText string) (core.Transaction, error) {
	amount, err := core.ParseAmount(amountText)
	if err != nil {
		return core.Transaction{}, err
	}
	fee, err := core.ParseAmount(feeText)
	if err != nil {
		return core.Transaction{}, err
	}
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return core.Transaction{}, fmt.Errorf("node is closed")
	}
	candidate, err := cloneState(s.state)
	if err != nil {
		s.mu.Unlock()
		return core.Transaction{}, err
	}
	utxo, err := candidate.SpendableUTXO()
	if err != nil {
		s.mu.Unlock()
		return core.Transaction{}, err
	}
	tx, err := core.BuildTransaction(s.wallet, to, amount, fee, utxo)
	if err == nil {
		err = candidate.AddPending(tx)
	}
	if err == nil {
		err = s.store.SaveState(candidate)
	}
	if err == nil {
		s.state = candidate
		s.revision++
	}
	s.mu.Unlock()
	if err != nil {
		return core.Transaction{}, err
	}
	s.launch(func() { s.broadcastTransaction(tx) })
	return tx, nil
}

func (s *Service) MineOnce(ctx context.Context) (core.Block, error) {
	s.mu.RLock()
	if s.closing {
		s.mu.RUnlock()
		return core.Block{}, fmt.Errorf("node is closed")
	}
	snapshot, err := cloneState(s.state)
	address := s.wallet.Address
	originalTip := s.state.Blocks[len(s.state.Blocks)-1].Hash
	originalRevision := s.revision
	s.mu.RUnlock()
	if err != nil {
		return core.Block{}, err
	}
	block, err := snapshot.Mine(ctx, address)
	if err != nil {
		return core.Block{}, err
	}

	s.mu.Lock()
	currentTip := s.state.Blocks[len(s.state.Blocks)-1].Hash
	if currentTip != originalTip || s.revision != originalRevision {
		s.mu.Unlock()
		return core.Block{}, fmt.Errorf("chain tip changed while mining; stale block discarded")
	}
	if err := s.store.SaveState(snapshot); err != nil {
		s.mu.Unlock()
		return core.Block{}, err
	}
	s.state = snapshot
	s.revision++
	s.mu.Unlock()
	s.launch(s.broadcastState)
	return block, nil
}

func (s *Service) StartMining() error {
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return fmt.Errorf("node is closed")
	}
	if s.mining {
		s.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.mining = true
	s.miningCancel = cancel
	s.wait.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.wait.Done()
		defer func() {
			s.mu.Lock()
			s.mining = false
			s.miningCancel = nil
			s.mu.Unlock()
		}()
		for {
			if _, err := s.MineOnce(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				s.setError(err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Second):
				}
			}
		}
	}()
	return nil
}

func (s *Service) StopMining() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.miningCancel != nil {
		s.miningCancel()
	}
}

func (s *Service) AddPeer(raw string) error {
	peer, err := normalizePeer(raw)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return fmt.Errorf("node is closed")
	}
	_, existed := s.peers[peer]
	s.peers[peer] = struct{}{}
	peers := s.peerListLocked()
	err = s.store.SavePeers(peers)
	if err != nil && !existed {
		delete(s.peers, peer)
		delete(s.peerOnline, peer)
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	s.launch(func() { s.syncPeer(peer) })
	return nil
}

func (s *Service) syncLoop(ctx context.Context) {
	defer s.wait.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, peer := range s.peersSnapshot() {
				peer := peer
				s.launch(func() { s.syncPeer(peer) })
			}
		}
	}
}

func (s *Service) syncPeer(peer string) {
	if !s.beginPeerSync(peer) {
		return
	}
	defer s.endPeerSync(peer)
	response, err := s.client.Get(peer + "/v1/state")
	if err != nil {
		s.setPeerOnline(peer, false)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		s.setPeerOnline(peer, false)
		return
	}
	var candidate core.State
	if err := decodeLimitedJSON(response.Body, maxStateBytes, &candidate); err != nil {
		s.setPeerOnline(peer, false)
		return
	}
	s.setPeerOnline(peer, true)
	if err := s.adoptIfBetter(&candidate); err != nil {
		s.setError(err)
	}
}

func (s *Service) adoptIfBetter(candidate *core.State) error {
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("peer sent invalid chain: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if candidate.CumulativeWork().Cmp(s.state.CumulativeWork()) <= 0 {
		return nil
	}
	remotePending := append([]core.Transaction(nil), candidate.Pending...)
	localPending := append([]core.Transaction(nil), s.state.Pending...)
	orphaned := orphanedTransactions(s.state, candidate)
	adopted, err := cloneState(candidate)
	if err != nil {
		return err
	}
	adopted.Pending = nil
	for _, tx := range append(append(orphaned, localPending...), remotePending...) {
		_ = adopted.AddPending(tx)
	}
	if err := s.store.SaveState(adopted); err != nil {
		return err
	}
	s.state = adopted
	s.revision++
	return nil
}

func (s *Service) acceptTransaction(tx core.Transaction) error {
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return fmt.Errorf("node is closed")
	}
	candidate, err := cloneState(s.state)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if err := candidate.AddPending(tx); err != nil {
		s.mu.Unlock()
		return err
	}
	if err := s.store.SaveState(candidate); err != nil {
		s.mu.Unlock()
		return err
	}
	s.state = candidate
	s.revision++
	s.mu.Unlock()
	s.launch(func() { s.broadcastTransaction(tx) })
	return nil
}

func (s *Service) broadcastTransaction(tx core.Transaction) {
	body, err := json.Marshal(tx)
	if err != nil {
		return
	}
	s.broadcastJSON("/v1/transactions", body)
}

func (s *Service) broadcastState() {
	s.mu.RLock()
	body, err := json.Marshal(s.state)
	s.mu.RUnlock()
	if err != nil {
		return
	}
	s.broadcastJSON("/v1/state", body)
}

func (s *Service) broadcastJSON(path string, body []byte) {
	var wait sync.WaitGroup
	for _, peer := range s.peersSnapshot() {
		peer := peer
		wait.Add(1)
		go func() {
			defer wait.Done()
			request, err := http.NewRequest(http.MethodPost, peer+path, bytes.NewReader(body))
			if err != nil {
				return
			}
			request.Header.Set("Content-Type", "application/json")
			response, err := s.client.Do(request)
			if err != nil {
				s.setPeerOnline(peer, false)
				return
			}
			s.setPeerOnline(peer, true)
			_ = response.Body.Close()
		}()
	}
	wait.Wait()
}

func (s *Service) handleStatus(writer http.ResponseWriter, _ *http.Request) {
	dashboard, err := s.Dashboard()
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err)
		return
	}
	s.mu.RLock()
	work := s.state.CumulativeWork().String()
	s.mu.RUnlock()
	writeJSON(writer, http.StatusOK, peerStatus{
		Name:    dashboard.Name,
		Symbol:  dashboard.Symbol,
		Height:  dashboard.Height,
		TipHash: dashboard.TipHash,
		Work:    work,
	})
}

func (s *Service) handleGetState(writer http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	body, err := json.Marshal(s.state)
	s.mu.RUnlock()
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(append(body, '\n'))
}

func (s *Service) handlePostState(writer http.ResponseWriter, request *http.Request) {
	var candidate core.State
	if err := decodeLimitedJSON(request.Body, maxStateBytes, &candidate); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if err := s.adoptIfBetter(&candidate); err != nil {
		writeError(writer, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"accepted": true})
}

func (s *Service) handleTransaction(writer http.ResponseWriter, request *http.Request) {
	var tx core.Transaction
	if err := decodeLimitedJSON(request.Body, maxTransactionBytes, &tx); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	if err := s.acceptTransaction(tx); err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusAccepted, map[string]string{"transaction_id": tx.ID})
}

func (s *Service) peersSnapshot() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.peerListLocked()
}

func (s *Service) peerListLocked() []string {
	peers := make([]string, 0, len(s.peers))
	for peer := range s.peers {
		peers = append(peers, peer)
	}
	sort.Strings(peers)
	return peers
}

func (s *Service) setError(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	s.lastError = err.Error()
	s.mu.Unlock()
}

func (s *Service) setPeerOnline(peer string, online bool) {
	s.mu.Lock()
	if _, configured := s.peers[peer]; configured {
		s.peerOnline[peer] = online
	}
	s.mu.Unlock()
}

func orphanedTransactions(local, candidate *core.State) []core.Transaction {
	common := len(local.Blocks) - 1
	if candidateTip := len(candidate.Blocks) - 1; candidateTip < common {
		common = candidateTip
	}
	for common > 0 && local.Blocks[common].Hash != candidate.Blocks[common].Hash {
		common--
	}
	transactions := make([]core.Transaction, 0)
	for index := common + 1; index < len(local.Blocks); index++ {
		transactions = append(transactions, local.Blocks[index].Transactions[1:]...)
	}
	return transactions
}

func normalizePeer(raw string) (string, error) {
	raw = strings.TrimSpace(strings.TrimRight(raw, "/"))
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("peer must be an http(s) URL such as http://192.168.1.20:47821")
	}
	if parsed.User != nil || parsed.Opaque != "" || parsed.Path != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("peer URL must not contain a path, query, or fragment")
	}
	hostname := parsed.Hostname()
	if net.ParseIP(hostname) == nil && !validDNSName(hostname) {
		return "", fmt.Errorf("peer host must be an IP address or ASCII DNS name")
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.ParseUint(port, 10, 16)
		if err != nil || value == 0 {
			return "", fmt.Errorf("peer port must be between 1 and 65535")
		}
	}
	return parsed.String(), nil
}

func validDNSName(hostname string) bool {
	if hostname == "" || len(hostname) > 253 || strings.HasPrefix(hostname, ".") || strings.HasSuffix(hostname, ".") {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}
	return true
}

func (s *Service) beginPeerSync(peer string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.syncing[peer] {
		return false
	}
	s.syncing[peer] = true
	return true
}

func (s *Service) endPeerSync(peer string) {
	s.mu.Lock()
	delete(s.syncing, peer)
	s.mu.Unlock()
}

func (s *Service) limitRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		select {
		case s.requestSlots <- struct{}{}:
			defer func() { <-s.requestSlots }()
			next.ServeHTTP(writer, request)
		default:
			writeError(writer, http.StatusServiceUnavailable, fmt.Errorf("node is busy"))
		}
	})
}

func (s *Service) launch(task func()) {
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return
	}
	s.wait.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.wait.Done()
		task()
	}()
}

func cloneState(state *core.State) (*core.State, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	var clone core.State
	if err := json.Unmarshal(data, &clone); err != nil {
		return nil, err
	}
	return &clone, nil
}

func decodeLimitedJSON(reader io.Reader, limit int64, value any) error {
	limited := io.LimitReader(reader, limit+1)
	decoder := json.NewDecoder(limited)
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}
	if decoder.InputOffset() > limit {
		return fmt.Errorf("request exceeds size limit")
	}
	return nil
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeError(writer http.ResponseWriter, status int, err error) {
	writeJSON(writer, status, map[string]string{"error": err.Error()})
}
