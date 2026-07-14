package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"entropy/internal/ledger"
)

const (
	maxPeerExchangePeers         = 16
	maxPeerExchangeResponseBytes = int64(32 << 10)
	peerExchangeInterval         = 10 * time.Minute
	peerExchangeRetryInterval    = time.Minute
)

func (s *Service) handlePeers(writer http.ResponseWriter, request *http.Request) {
	writeBoundedJSON(writer, http.StatusOK, peersResponse{
		Protocol: ledger.ProtocolName,
		Peers:    s.publicPeerURLs(clientIPFromContext(request.Context())),
	}, maxPeerExchangeResponseBytes)
}

func (s *Service) publicPeerURLs(excludeIP string) []string {
	s.mu.RLock()
	peers := make([]string, 0, maxPeerExchangePeers)
	recentCutoff := time.Now().Add(-2 * time.Minute)
	for peer, state := range s.peers {
		if state != nil && state.Public && state.Online && state.ActiveOutbound && state.Failures == 0 &&
			!state.LastSeen.Before(recentCutoff) {
			parsed, err := url.Parse(peer)
			if err == nil && parsed.Hostname() == excludeIP {
				continue
			}
			peers = append(peers, peer)
		}
	}
	s.mu.RUnlock()
	sort.Strings(peers)
	if len(peers) > maxPeerExchangePeers {
		peers = peers[:maxPeerExchangePeers]
	}
	return peers
}

func (s *Service) exchangePeerAddresses(ctx context.Context, peer string) {
	now := time.Now()
	s.mu.Lock()
	if next := s.peerExchangeNext[peer]; !next.IsZero() && now.Before(next) {
		s.mu.Unlock()
		return
	}
	s.peerExchangeNext[peer] = now.Add(peerExchangeInterval)
	s.mu.Unlock()

	var response peersResponse
	found, err := s.getOptionalJSON(ctx, peer+"/v2/peers", maxPeerExchangeResponseBytes, &response)
	if err != nil || !found {
		if err != nil {
			s.mu.Lock()
			if s.peers[peer] != nil {
				s.peerExchangeNext[peer] = time.Now().Add(peerExchangeRetryInterval)
			}
			s.mu.Unlock()
		}
		return
	}
	if response.Protocol != ledger.ProtocolName || len(response.Peers) > maxPeerExchangePeers {
		return
	}
	for _, candidate := range response.Peers {
		if candidate == peer {
			continue
		}
		if err := s.addPublicDiscoveredPeer(ctx, candidate); err != nil && !errors.Is(err, context.Canceled) {
			continue
		}
	}
}

func (s *Service) getOptionalJSON(ctx context.Context, endpoint string, maximum int64, result any) (bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusMethodNotAllowed {
		return false, nil
	}
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("peer returned HTTP %d", response.StatusCode)
	}
	if err := decodeLimitedJSON(response.Body, maximum, result); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) addPublicDiscoveredPeer(ctx context.Context, raw string) error {
	peer, err := normalizePublicPeer(raw)
	if err != nil {
		return err
	}
	return s.persistDiscoveredPeer(ctx, peer, true, false)
}
