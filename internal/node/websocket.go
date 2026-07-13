package node

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"entropy/internal/core"
	"entropy/internal/ledger"

	"github.com/gorilla/websocket"
)

type peerSocket struct {
	service    *Service
	connection *websocket.Conn
	baseURL    string
	outbound   bool
	send       chan gossipMessage
	done       chan struct{}
	closeOnce  sync.Once
	helloSeen  bool
}

const (
	webSocketInboundBytesPerSecond = float64(2 << 20)
	webSocketInboundByteBurst      = float64(4 << 20)
	webSocketValidationPerSecond   = float64(512)
	webSocketValidationBurst       = float64(4_096)
	webSocketInvalidScoreLimit     = 12
	invalidTransactionScore        = 1
	invalidBlockScore              = 4
	invalidGossipScoreTTL          = 5 * time.Minute
)

type invalidGossipState struct {
	score   int
	updated time.Time
}

type socketInboundBudget struct {
	last             time.Time
	byteTokens       float64
	validationTokens float64
}

func newSocketInboundBudget(now time.Time) *socketInboundBudget {
	return &socketInboundBudget{
		last:             now,
		byteTokens:       webSocketInboundByteBurst,
		validationTokens: webSocketValidationBurst,
	}
}

func (b *socketInboundBudget) allow(now time.Time, byteCost, validationCost int) bool {
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.byteTokens = min(webSocketInboundByteBurst, b.byteTokens+elapsed*webSocketInboundBytesPerSecond)
		b.validationTokens = min(webSocketValidationBurst, b.validationTokens+elapsed*webSocketValidationPerSecond)
		b.last = now
	}
	if float64(byteCost) > b.byteTokens || float64(validationCost) > b.validationTokens {
		return false
	}
	b.byteTokens -= float64(byteCost)
	b.validationTokens -= float64(validationCost)
	return true
}

var peerUpgrader = websocket.Upgrader{
	HandshakeTimeout: 5 * time.Second,
	ReadBufferSize:   16 << 10,
	WriteBufferSize:  16 << 10,
	CheckOrigin: func(request *http.Request) bool {
		return request.Header.Get("Origin") == ""
	},
}

func (s *Service) handleWebSocket(writer http.ResponseWriter, request *http.Request) {
	connection, err := peerUpgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	socket := &peerSocket{
		service:    s,
		connection: connection,
		send:       make(chan gossipMessage, 128),
		done:       make(chan struct{}),
	}
	if !s.registerSocket(socket) {
		_ = connection.Close()
		return
	}
	socket.queue(s.helloMessage())
	socket.run(request.Context())
}

func (s *Service) ensurePeerSocket(peer string) {
	s.mu.Lock()
	if s.closing || s.outboundSockets[peer] != nil || s.dialing[peer] || s.peers[peer] == nil {
		s.mu.Unlock()
		return
	}
	s.dialing[peer] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.dialing, peer)
		s.mu.Unlock()
	}()
	parsed, err := url.Parse(peer)
	if err != nil {
		return
	}
	if parsed.Scheme == "https" {
		parsed.Scheme = "wss"
	} else {
		parsed.Scheme = "ws"
	}
	parsed.Path = "/v2/p2p"
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second, Proxy: http.ProxyFromEnvironment}
	connection, response, err := dialer.DialContext(s.backgroundContext(), parsed.String(), nil)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		s.markPeerFailure(peer, fmt.Errorf("open realtime channel: %w", err))
		return
	}
	socket := &peerSocket{
		service:    s,
		connection: connection,
		baseURL:    peer,
		outbound:   true,
		send:       make(chan gossipMessage, 128),
		done:       make(chan struct{}),
	}
	if !s.registerSocket(socket) {
		_ = connection.Close()
		return
	}
	socket.queue(s.helloMessage())
	go socket.run(s.backgroundContext())
}

func (s *Service) registerSocket(socket *peerSocket) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing || len(s.sockets) >= maxPeerConnections {
		return false
	}
	if socket.outbound {
		if s.peers[socket.baseURL] == nil {
			return false
		}
		if existing := s.outboundSockets[socket.baseURL]; existing != nil {
			return false
		}
		s.outboundSockets[socket.baseURL] = socket
	}
	s.sockets[socket] = struct{}{}
	s.wait.Add(1)
	return true
}

func (s *Service) unregisterSocket(socket *peerSocket) {
	s.mu.Lock()
	delete(s.sockets, socket)
	if socket.outbound && s.outboundSockets[socket.baseURL] == socket {
		delete(s.outboundSockets, socket.baseURL)
	}
	s.mu.Unlock()
}

func (s *Service) helloMessage() gossipMessage {
	tip, err := s.ledger.Tip(context.Background())
	if err != nil {
		return gossipMessage{Type: "hello", Protocol: ledger.ProtocolName, NodeID: s.Address()}
	}
	status := statusFromTip(tip, s.listenPort())
	return gossipMessage{
		Type:       "hello",
		Protocol:   ledger.ProtocolName,
		NodeID:     s.Address(),
		ListenPort: status.ListenPort,
		Status:     &status,
	}
}

func (s *Service) listenPort() int {
	s.mu.RLock()
	address := s.actualAddress
	s.mu.RUnlock()
	_, rawPort, err := net.SplitHostPort(address)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return 0
	}
	return port
}

func (p *peerSocket) run(ctx context.Context) {
	defer p.service.wait.Done()
	defer p.service.unregisterSocket(p)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		p.writeLoop(ctx)
	}()
	p.readLoop(ctx)
	p.close()
	<-writerDone
}

func (p *peerSocket) readLoop(ctx context.Context) {
	p.connection.SetReadLimit(maxProtocolBytes)
	_ = p.connection.SetReadDeadline(time.Now().Add(45 * time.Second))
	p.connection.SetPongHandler(func(string) error {
		return p.connection.SetReadDeadline(time.Now().Add(45 * time.Second))
	})
	budget := newSocketInboundBudget(time.Now())
	for {
		messageType, data, err := p.connection.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage {
			p.penalize(fmt.Errorf("peer sent a non-text gossip message"))
			return
		}
		if !budget.allow(time.Now(), len(data), 0) {
			p.penalize(fmt.Errorf("peer exceeded the realtime byte budget"))
			return
		}
		var message gossipMessage
		if err := decodeLimitedJSON(bytes.NewReader(data), maxProtocolBytes, &message); err != nil {
			p.penalize(fmt.Errorf("decode realtime message: %w", err))
			return
		}
		if message.Protocol != ledger.ProtocolName {
			p.penalize(fmt.Errorf("peer sent an incompatible realtime protocol"))
			return
		}
		if !budget.allow(time.Now(), 0, gossipValidationCost(message)) {
			p.penalize(fmt.Errorf("peer exceeded the realtime validation budget"))
			return
		}
		keep, invalidScore, handleErr := p.service.handleGossip(ctx, p, message)
		if !keep {
			if handleErr != nil {
				p.penalize(handleErr)
			}
			return
		}
		if invalidScore > 0 {
			if p.service.addInvalidGossipScore(p, invalidScore, time.Now()) >= webSocketInvalidScoreLimit {
				p.penalize(fmt.Errorf("peer exceeded the invalid gossip threshold: %w", handleErr))
				return
			}
		}
	}
}

func (s *Service) addInvalidGossipScore(socket *peerSocket, delta int, now time.Time) int {
	if delta <= 0 || socket == nil || socket.connection == nil {
		return 0
	}
	host, _, err := net.SplitHostPort(socket.connection.RemoteAddr().String())
	if err != nil {
		return webSocketInvalidScoreLimit
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.invalidGossipByIP[host]
	if state == nil || now.Sub(state.updated) > invalidGossipScoreTTL {
		state = &invalidGossipState{}
		s.invalidGossipByIP[host] = state
	}
	state.score += delta
	state.updated = now
	if len(s.invalidGossipByIP) > 4_096 {
		for peerIP, candidate := range s.invalidGossipByIP {
			if peerIP != host && now.Sub(candidate.updated) > invalidGossipScoreTTL {
				delete(s.invalidGossipByIP, peerIP)
			}
		}
	}
	return state.score
}

func gossipValidationCost(message gossipMessage) int {
	cost := 1
	if message.Transaction != nil {
		cost += len(message.Transaction.Inputs) + len(message.Transaction.Outputs)
	}
	if message.Block != nil {
		cost += 16 + len(message.Block.Transactions)
		for _, transaction := range message.Block.Transactions {
			cost += len(transaction.Inputs) + len(transaction.Outputs)
			if cost > int(webSocketValidationBurst) {
				return cost
			}
		}
	}
	return cost
}

func (p *peerSocket) penalize(err error) {
	if err != nil && p.baseURL != "" {
		p.service.markPeerFailure(p.baseURL, err)
	}
}

func (p *peerSocket) writeLoop(ctx context.Context) {
	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.done:
			return
		case message := <-p.send:
			_ = p.connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := p.connection.WriteJSON(message); err != nil {
				return
			}
		case <-ping.C:
			_ = p.connection.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := p.connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}

func (p *peerSocket) queue(message gossipMessage) bool {
	select {
	case <-p.done:
		return false
	case p.send <- message:
		return true
	default:
		return false
	}
}

func (p *peerSocket) close() {
	p.closeOnce.Do(func() {
		close(p.done)
		_ = p.connection.Close()
	})
}

func (s *Service) handleGossip(ctx context.Context, socket *peerSocket, message gossipMessage) (bool, int, error) {
	if message.Type != "hello" && !socket.helloSeen {
		return false, 0, fmt.Errorf("peer sent gossip before completing hello")
	}
	switch message.Type {
	case "hello":
		if socket.helloSeen || core.ValidateAddress(message.NodeID) != nil || message.NodeID == s.Address() {
			return false, 0, fmt.Errorf("peer hello contains an invalid node ID")
		}
		if !socket.outbound && (message.ListenPort <= 0 || message.ListenPort > 65535) {
			return false, 0, fmt.Errorf("inbound peer hello contains an invalid listen port")
		}
		if message.Status != nil {
			if err := validateRemoteStatus(*message.Status); err != nil {
				return false, 0, err
			}
			if socket.baseURL != "" {
				s.markPeerSuccess(socket.baseURL, message.Status.Height)
			}
		}
		if socket.baseURL == "" && message.ListenPort > 0 {
			host, _, err := net.SplitHostPort(socket.connection.RemoteAddr().String())
			if err == nil {
				base := "http://" + net.JoinHostPort(host, strconv.Itoa(message.ListenPort))
				if normalized, err := normalizePeer(base); err == nil {
					socket.baseURL = normalized
					s.addDiscoveredPeer(normalized)
				}
			}
		}
		socket.helloSeen = true
		return true, 0, nil
	case "status":
		if message.Status == nil {
			return false, 0, fmt.Errorf("peer status is missing")
		}
		if err := validateRemoteStatus(*message.Status); err != nil {
			return false, 0, err
		}
		if socket.baseURL != "" {
			s.markPeerSuccess(socket.baseURL, message.Status.Height)
		}
		return true, 0, nil
	case "transaction":
		if message.Transaction == nil {
			return false, 0, fmt.Errorf("peer transaction is missing")
		}
		if err := s.acceptTransaction(ctx, *message.Transaction, socket); err != nil {
			if errors.Is(err, ledger.ErrTransactionAlreadyKnown) {
				return true, 0, nil
			}
			return true, invalidTransactionScore, err
		}
		return true, 0, nil
	case "block":
		if message.Block == nil {
			return false, 0, fmt.Errorf("peer block is missing")
		}
		tip, tipErr := s.ledger.Tip(ctx)
		err := s.acceptBlock(ctx, *message.Block, socket)
		if err != nil {
			if socket.baseURL != "" {
				peer := socket.baseURL
				s.launch(func() { s.syncPeer(peer) })
			}
			if tipErr == nil && message.Block.Height == tip.Height+1 && message.Block.PreviousHash == tip.Hash {
				return true, invalidBlockScore, err
			}
		}
		return true, 0, nil
	case "ping":
		return socket.queue(gossipMessage{Type: "pong", Protocol: ledger.ProtocolName}), 0, nil
	case "pong":
		return true, 0, nil
	default:
		return false, 0, fmt.Errorf("peer sent an unknown gossip message type")
	}
}

func (s *Service) broadcastTransaction(transaction core.Transaction, source *peerSocket) {
	s.broadcastGossip(gossipMessage{Type: "transaction", Protocol: ledger.ProtocolName, Transaction: &transaction}, source)
	s.broadcastHTTP("/v2/transactions", transaction)
}

func (s *Service) broadcastBlock(block core.Block, source *peerSocket) {
	s.broadcastGossip(gossipMessage{Type: "block", Protocol: ledger.ProtocolName, Block: &block}, source)
	s.broadcastHTTP("/v2/block", block)
}

func (s *Service) broadcastGossip(message gossipMessage, source *peerSocket) {
	s.mu.RLock()
	sockets := make([]*peerSocket, 0, len(s.sockets))
	for socket := range s.sockets {
		if socket != source {
			sockets = append(sockets, socket)
		}
	}
	s.mu.RUnlock()
	for _, socket := range sockets {
		_ = socket.queue(message)
	}
}

func (s *Service) broadcastHTTP(path string, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		return
	}
	for _, peer := range s.peerURLs() {
		peer := peer
		s.launch(func() {
			ctx, cancel := context.WithTimeout(s.backgroundContext(), 8*time.Second)
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, peer+path, bytes.NewReader(body))
			if err != nil {
				return
			}
			request.Header.Set("Content-Type", "application/json")
			response, err := s.client.Do(request)
			if err != nil {
				s.markPeerFailure(peer, err)
				return
			}
			_ = response.Body.Close()
			if (response.StatusCode >= 300 && response.StatusCode < 400) || response.StatusCode >= 500 {
				s.markPeerFailure(peer, fmt.Errorf("peer returned HTTP %d", response.StatusCode))
			}
		})
	}
}
