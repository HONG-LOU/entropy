package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxDiscoveredPeers = 48

type peerState struct {
	URL         string
	Online      bool
	Height      uint64
	Failures    int
	LastError   string
	LastSeen    time.Time
	NextAttempt time.Time
	Discovered  bool
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
	if s.actualAddress != "" {
		if local, normalizeErr := normalizePeer("http://" + s.actualAddress); normalizeErr == nil && local == peer {
			s.mu.Unlock()
			return fmt.Errorf("cannot add this node as its own peer")
		}
	}
	if existing, found := s.peers[peer]; found {
		existing.Discovered = false
		s.mu.Unlock()
		return s.ledger.UpsertPeer(context.Background(), peer, true)
	}
	if len(s.peers) >= maxPeerConnections {
		s.mu.Unlock()
		return fmt.Errorf("peer limit reached")
	}
	if err := s.ledger.UpsertPeer(context.Background(), peer, true); err != nil {
		s.mu.Unlock()
		return err
	}
	s.peers[peer] = &peerState{URL: peer}
	s.mu.Unlock()
	s.launch(func() {
		s.ensurePeerSocket(peer)
		s.syncPeer(peer)
	})
	return nil
}

func (s *Service) RemovePeer(raw string) error {
	peer, err := normalizePeer(raw)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return fmt.Errorf("node is closed")
	}
	_, found := s.peers[peer]
	if !found {
		s.mu.Unlock()
		return nil
	}
	if err := s.ledger.RemovePeer(context.Background(), peer); err != nil {
		s.mu.Unlock()
		return err
	}
	delete(s.peers, peer)
	delete(s.mempoolOffsets, peer)
	sockets := make([]*peerSocket, 0, 2)
	for socket := range s.sockets {
		if socket.baseURL == peer {
			sockets = append(sockets, socket)
		}
	}
	delete(s.outboundSockets, peer)
	s.mu.Unlock()
	for _, socket := range sockets {
		socket.close()
	}
	return nil
}

func (s *Service) addDiscoveredPeer(raw string) {
	peer, err := normalizePeer(raw)
	if err != nil {
		return
	}
	s.mu.Lock()
	if s.closing || len(s.peers) >= maxPeerConnections || s.discoveredPeerCountLocked() >= maxDiscoveredPeers {
		s.mu.Unlock()
		return
	}
	added := false
	if _, exists := s.peers[peer]; !exists {
		s.peers[peer] = &peerState{URL: peer, Discovered: true}
		added = true
	}
	s.mu.Unlock()
	if added {
		s.launch(func() {
			ctx := s.backgroundContext()
			s.mu.Lock()
			state := s.peers[peer]
			if s.closing || state == nil || !state.Discovered {
				s.mu.Unlock()
				return
			}
			err := s.ledger.UpsertPeer(ctx, peer, false)
			s.mu.Unlock()
			if err != nil && !errors.Is(err, context.Canceled) {
				s.setError(err)
			}
		})
	}
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

func (s *Service) syncLoop(ctx context.Context) {
	defer s.wait.Done()
	ticker := time.NewTicker(s.syncInterval)
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
	for _, peer := range s.peerURLs() {
		s.mu.RLock()
		state := s.peers[peer]
		due := state != nil && (state.NextAttempt.IsZero() || !time.Now().Before(state.NextAttempt))
		s.mu.RUnlock()
		if !due {
			continue
		}
		peer := peer
		s.launch(func() {
			s.ensurePeerSocket(peer)
			s.syncPeer(peer)
		})
	}
}

func (s *Service) beginPeerSync(peer string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing || s.syncing[peer] {
		return false
	}
	state, exists := s.peers[peer]
	if !exists || (!state.NextAttempt.IsZero() && time.Now().Before(state.NextAttempt)) {
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
	}
	chain := s.ledger
	closing := s.closing
	s.mu.Unlock()
	if persist && !closing && chain != nil {
		s.launch(func() {
			if err := chain.RecordPeerSuccess(s.backgroundContext(), peer, now); err != nil && !errors.Is(err, context.Canceled) {
				s.setError(err)
			}
		})
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
	nextAttempt := state.NextAttempt
	chain := s.ledger
	closing := s.closing
	s.mu.Unlock()
	if !closing && chain != nil {
		s.launch(func() {
			if recordErr := chain.RecordPeerFailure(s.backgroundContext(), peer, nextAttempt, err); recordErr != nil && !errors.Is(recordErr, context.Canceled) {
				s.setError(recordErr)
			}
		})
	}
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
