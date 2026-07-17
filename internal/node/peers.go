package node

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	maxDiscoveredPeers       = 48
	maxPublicDiscoveredPeers = 24
	defaultMaxOutboundPeers  = 8
	defaultPeerSyncInterval  = 15 * time.Second
	peerSchedulerInterval    = time.Second
	staleDiscoveredFailures  = 8
)

func (s *Service) removeStaleDiscoveredPeers(ctx context.Context) error {
	s.mu.Lock()
	stale := make([]string, 0)
	for peer, state := range s.peers {
		if state.Discovered && !state.Bootstrap && state.LastSeen.IsZero() && state.Failures >= staleDiscoveredFailures {
			stale = append(stale, peer)
			delete(s.peers, peer)
			delete(s.activeOutbound, peer)
			delete(s.mempoolOffsets, peer)
			delete(s.peerExchangeNext, peer)
		}
	}
	s.mu.Unlock()
	for _, peer := range stale {
		if err := s.ledger.RemovePeer(ctx, peer); err != nil {
			return err
		}
	}
	return nil
}

type peerState struct {
	URL                string
	Online             bool
	Height             uint64
	Failures           int
	LastError          string
	LastSeen           time.Time
	NextAttempt        time.Time
	NextSync           time.Time
	NextSocket         time.Time
	SocketFailures     int
	LivenessGeneration uint64
	Discovered         bool
	Public             bool
	Bootstrap          bool
	ActiveOutbound     bool
}

func (s *Service) AddPeer(raw string) error {
	peer, err := normalizePeer(raw)
	if err != nil {
		return err
	}
	s.peerMutationMu.Lock()
	defer s.peerMutationMu.Unlock()
	s.mu.RLock()
	if s.closing {
		s.mu.RUnlock()
		return fmt.Errorf("node is closed")
	}
	if s.actualAddress != "" {
		if local, normalizeErr := normalizePeer("http://" + s.actualAddress); normalizeErr == nil && local == peer {
			s.mu.RUnlock()
			return fmt.Errorf("cannot add this node as its own peer")
		}
	}
	_, exists := s.peers[peer]
	if !exists && len(s.peers) >= maxPeerConnections {
		s.mu.RUnlock()
		return fmt.Errorf("peer limit reached")
	}
	chain := s.ledger
	s.mu.RUnlock()
	if chain == nil {
		return fmt.Errorf("node ledger is unavailable")
	}
	if err := chain.UpsertPeer(context.Background(), peer, true); err != nil {
		return err
	}
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return fmt.Errorf("node is closed")
	}
	state := s.peers[peer]
	if state == nil {
		_, publicErr := normalizePublicPeer(peer)
		state = &peerState{URL: peer, Public: publicErr == nil}
		s.peers[peer] = state
	} else {
		state.Discovered = false
		state.Bootstrap = false
		if _, publicErr := normalizePublicPeer(peer); publicErr == nil {
			state.Public = true
		}
	}
	activate := s.activatePeerLocked(peer)
	s.mu.Unlock()
	if activate {
		s.launch(func() { s.ensurePeerSocket(peer) })
		s.launch(func() { s.syncPeer(peer) })
	}
	return nil
}

func (s *Service) RemovePeer(raw string) error {
	peer, err := normalizePeer(raw)
	if err != nil {
		return err
	}
	s.peerMutationMu.Lock()
	defer s.peerMutationMu.Unlock()
	s.mu.RLock()
	if s.closing {
		s.mu.RUnlock()
		return fmt.Errorf("node is closed")
	}
	_, found := s.peers[peer]
	if !found {
		s.mu.RUnlock()
		return nil
	}
	chain := s.ledger
	s.mu.RUnlock()
	if chain == nil {
		return fmt.Errorf("node ledger is unavailable")
	}
	if err := chain.RemovePeer(context.Background(), peer); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.peers, peer)
	delete(s.activeOutbound, peer)
	delete(s.mempoolOffsets, peer)
	delete(s.peerExchangeNext, peer)
	sockets := make([]*peerSocket, 0, 2)
	for socket := range s.sockets {
		if socket.baseURL == peer {
			sockets = append(sockets, socket)
		}
	}
	s.mu.Unlock()
	for _, socket := range sockets {
		socket.stop()
	}
	return nil
}

func (s *Service) addDiscoveredPeer(raw string) {
	peer, err := normalizePeer(raw)
	if err != nil {
		return
	}
	publicPeer, publicErr := normalizePublicPeer(peer)
	if publicErr == nil {
		peer = publicPeer
	}
	if err := s.persistDiscoveredPeer(s.backgroundContext(), peer, publicErr == nil, false); err != nil && !errors.Is(err, context.Canceled) {
		s.setError(err)
	}
}

func (s *Service) persistDiscoveredPeer(ctx context.Context, peer string, public, bootstrap bool) error {
	s.peerMutationMu.Lock()
	defer s.peerMutationMu.Unlock()
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return context.Canceled
	}
	if state := s.peers[peer]; state != nil {
		state.Public = state.Public || public
		if state.Discovered {
			state.Bootstrap = state.Bootstrap || bootstrap
		}
		s.mu.Unlock()
		return nil
	}
	if len(s.peers) >= maxPeerConnections || s.discoveredPeerCountLocked() >= maxDiscoveredPeers {
		s.mu.Unlock()
		return fmt.Errorf("discovered peer limit reached")
	}
	if public && !bootstrap && s.publicDiscoveredPeerCountLocked() >= maxPublicDiscoveredPeers {
		s.mu.Unlock()
		return fmt.Errorf("public discovered peer limit reached")
	}
	chain := s.ledger
	s.mu.Unlock()
	if chain == nil {
		return fmt.Errorf("node ledger is unavailable")
	}
	if err := chain.UpsertPeer(ctx, peer, false); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return context.Canceled
	}
	if state := s.peers[peer]; state != nil {
		state.Public = state.Public || public
		if state.Discovered {
			state.Bootstrap = state.Bootstrap || bootstrap
		}
		return nil
	}
	s.peers[peer] = &peerState{URL: peer, Discovered: true, Public: public, Bootstrap: bootstrap}
	return nil
}

func (s *Service) publicDiscoveredPeerCountLocked() int {
	count := 0
	for _, peer := range s.peers {
		if peer.Discovered && peer.Public && !peer.Bootstrap {
			count++
		}
	}
	return count
}

func (s *Service) discoveredPeerCountLocked() int {
	count := 0
	for _, peer := range s.peers {
		if peer.Discovered {
			count++
		}
	}
	return count
}

func (s *Service) peerURLs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	peers := make([]string, 0, len(s.peers))
	for peer := range s.peers {
		peers = append(peers, peer)
	}
	sort.Strings(peers)
	return peers
}

func (s *Service) activePeerURLs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	peers := make([]string, 0, len(s.activeOutbound))
	for peer := range s.activeOutbound {
		if s.peers[peer] != nil {
			peers = append(peers, peer)
		}
	}
	sort.Strings(peers)
	return peers
}

func (s *Service) syncLoop(ctx context.Context) {
	defer s.wait.Done()
	interval := min(s.syncInterval, peerSchedulerInterval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		s.runSyncRound()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) runSyncRound() {
	now := time.Now()
	for _, peer := range s.selectActivePeers(now) {
		s.mu.RLock()
		state := s.peers[peer]
		due := state != nil && state.ActiveOutbound &&
			(state.NextAttempt.IsZero() || !now.Before(state.NextAttempt)) &&
			(state.NextSync.IsZero() || !now.Before(state.NextSync))
		s.mu.RUnlock()
		peer := peer
		s.launch(func() { s.ensurePeerSocket(peer) })
		if due {
			s.launch(func() { s.syncPeerScheduled(peer) })
		}
	}
}

type peerCandidate struct {
	url      string
	state    *peerState
	tieBreak uint64
}

func (s *Service) selectActivePeers(now time.Time) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for peer := range s.activeOutbound {
		state := s.peers[peer]
		if state == nil || !state.ActiveOutbound {
			delete(s.activeOutbound, peer)
		}
	}
	candidates := make([]peerCandidate, 0, len(s.peers))
	for peer, state := range s.peers {
		if state.ActiveOutbound || (!state.NextAttempt.IsZero() && now.Before(state.NextAttempt)) {
			continue
		}
		candidates = append(candidates, peerCandidate{url: peer, state: state, tieBreak: peerTieBreak(s.wallet.Address, peer)})
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.state.Online != right.state.Online {
			return left.state.Online
		}
		if !left.state.LastSeen.Equal(right.state.LastSeen) {
			return left.state.LastSeen.After(right.state.LastSeen)
		}
		if left.state.Discovered != right.state.Discovered {
			return !left.state.Discovered
		}
		if left.state.Bootstrap != right.state.Bootstrap {
			return left.state.Bootstrap
		}
		if left.state.Failures != right.state.Failures {
			return left.state.Failures < right.state.Failures
		}
		return left.tieBreak < right.tieBreak
	})
	for _, candidate := range candidates {
		if len(s.activeOutbound) >= s.maxOutboundPeers {
			break
		}
		candidate.state.ActiveOutbound = true
		s.activeOutbound[candidate.url] = struct{}{}
	}
	peers := make([]string, 0, len(s.activeOutbound))
	for peer := range s.activeOutbound {
		peers = append(peers, peer)
	}
	sort.Strings(peers)
	return peers
}

func (s *Service) activatePeerLocked(peer string) bool {
	state := s.peers[peer]
	if state == nil || state.ActiveOutbound || len(s.activeOutbound) >= s.maxOutboundPeers ||
		(!state.NextAttempt.IsZero() && time.Now().Before(state.NextAttempt)) {
		return false
	}
	state.ActiveOutbound = true
	s.activeOutbound[peer] = struct{}{}
	return true
}

func peerTieBreak(nodeID, peer string) uint64 {
	hash := sha256.Sum256([]byte(nodeID + "\x00" + peer))
	return binary.BigEndian.Uint64(hash[:8])
}

func (s *Service) beginPeerSync(peer string) bool {
	return s.beginPeerSyncMode(peer, true)
}

func (s *Service) beginPeerSyncMode(peer string, force bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing || s.syncing[peer] {
		return false
	}
	state, exists := s.peers[peer]
	now := time.Now()
	if !exists || (!state.NextAttempt.IsZero() && now.Before(state.NextAttempt)) {
		return false
	}
	if !state.ActiveOutbound && !s.activatePeerLocked(peer) {
		return false
	}
	if !force && !state.NextSync.IsZero() && now.Before(state.NextSync) {
		return false
	}
	state.NextSync = now.Add(s.syncInterval)
	s.syncing[peer] = true
	return true
}

func (s *Service) endPeerSync(peer string) {
	s.mu.Lock()
	delete(s.syncing, peer)
	s.mu.Unlock()
}

func (s *Service) markPeerSuccess(peer string, height uint64) {
	now := time.Now()
	s.mu.Lock()
	persist := false
	if state := s.peers[peer]; state != nil {
		persist = state.LastSeen.IsZero() || now.Sub(state.LastSeen) >= 30*time.Second || state.Failures > 0
		state.Online = true
		state.Height = height
		state.Failures = 0
		state.LastError = ""
		state.LastSeen = now
		state.NextAttempt = time.Time{}
		state.LivenessGeneration++
		generation := state.LivenessGeneration
		if persist {
			chain := s.ledger
			closing := s.closing
			s.mu.Unlock()
			if !closing && chain != nil {
				s.launch(func() { s.persistPeerSuccess(peer, generation, now) })
			}
			return
		}
	}
	s.mu.Unlock()
}

func (s *Service) persistPeerSuccess(peer string, generation uint64, seenAt time.Time) {
	s.peerMutationMu.Lock()
	defer s.peerMutationMu.Unlock()
	s.mu.RLock()
	state := s.peers[peer]
	if s.closing || state == nil || !state.Online || state.LivenessGeneration != generation {
		s.mu.RUnlock()
		return
	}
	chain := s.ledger
	s.mu.RUnlock()
	if chain != nil {
		if err := chain.RecordPeerSuccess(s.backgroundContext(), peer, seenAt); err != nil && !errors.Is(err, context.Canceled) {
			s.setError(err)
		}
	}
}

func (s *Service) markPeerFailure(peer string, err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	state := s.peers[peer]
	if state == nil {
		s.mu.Unlock()
		return
	}
	state.Online = false
	state.Failures++
	state.LastError = err.Error()
	shift := state.Failures - 1
	if shift > 8 {
		shift = 8
	}
	delay := time.Second * time.Duration(1<<shift)
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	state.NextAttempt = time.Now().Add(delay)
	state.NextSync = time.Time{}
	state.ActiveOutbound = false
	state.LivenessGeneration++
	generation := state.LivenessGeneration
	delete(s.activeOutbound, peer)
	nextAttempt := state.NextAttempt
	socket := s.outboundSockets[peer]
	chain := s.ledger
	closing := s.closing
	s.mu.Unlock()
	if socket != nil {
		socket.stop()
	}
	if !closing && chain != nil {
		s.launch(func() { s.persistPeerFailure(peer, generation, nextAttempt, err) })
	}
}

func (s *Service) persistPeerFailure(peer string, generation uint64, nextAttempt time.Time, cause error) {
	s.peerMutationMu.Lock()
	defer s.peerMutationMu.Unlock()
	s.mu.RLock()
	state := s.peers[peer]
	if s.closing || state == nil || state.Online || state.LivenessGeneration != generation {
		s.mu.RUnlock()
		return
	}
	chain := s.ledger
	s.mu.RUnlock()
	if chain != nil {
		if err := chain.RecordPeerFailure(s.backgroundContext(), peer, nextAttempt, cause); err != nil && !errors.Is(err, context.Canceled) {
			s.setError(err)
		}
	}
}

func (s *Service) markPeerSocketFailure(peer string, err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	state := s.peers[peer]
	if state == nil || s.closing {
		s.mu.Unlock()
		return
	}
	state.SocketFailures++
	shift := state.SocketFailures - 1
	if shift > 8 {
		shift = 8
	}
	delay := time.Second * time.Duration(1<<shift)
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	state.NextSocket = time.Now().Add(delay)
	s.mu.Unlock()
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

func normalizePublicPeer(raw string) (string, error) {
	peer, err := normalizePeer(raw)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(peer)
	if err != nil {
		return "", fmt.Errorf("public peer URL is invalid")
	}
	if parsed.Port() == "" {
		return "", fmt.Errorf("public peer must include an explicit port")
	}
	address, err := netip.ParseAddr(parsed.Hostname())
	if err != nil {
		return "", fmt.Errorf("public peer host must be an IP address")
	}
	if address.Zone() != "" {
		return "", fmt.Errorf("public peer host must not contain an IPv6 zone")
	}
	address = address.Unmap()
	if !isPublicPeerIP(address) {
		return "", fmt.Errorf("public peer host is not globally routable")
	}
	port, err := strconv.ParseUint(parsed.Port(), 10, 16)
	if err != nil || port == 0 {
		return "", fmt.Errorf("public peer port must be between 1 and 65535")
	}
	parsed.Host = net.JoinHostPort(address.String(), strconv.FormatUint(port, 10))
	return parsed.String(), nil
}

func isPublicPeerIP(address netip.Addr) bool {
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() ||
		address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	for _, prefix := range nonPublicPeerPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

var nonPublicPeerPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
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
