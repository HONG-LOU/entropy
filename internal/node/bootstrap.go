package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/HONG-LOU/entcoin/internal/ledger"
)

const (
	bootstrapManifestVersion  = 1
	maxBootstrapManifestBytes = int64(32 << 10)
	maxBootstrapManifestPeers = 16
	maxBootstrapManifestURLs  = 8
	bootstrapRefreshInterval  = 6 * time.Hour
	bootstrapRetryInterval    = time.Minute
)

var defaultBootstrapManifestURLs = []string{
	"https://raw.githubusercontent.com/HONG-LOU/entcoin/main/network/mainnet.json",
	"https://cdn.jsdelivr.net/gh/HONG-LOU/entcoin@main/network/mainnet.json",
}

type bootstrapManifest struct {
	Version  int      `json:"version"`
	Protocol string   `json:"protocol"`
	Peers    []string `json:"peers"`
}

// DefaultBootstrapManifestURLs returns independent delivery paths for the
// public seed list. Both sources contain the same small, versioned document.
func DefaultBootstrapManifestURLs() []string {
	return append([]string(nil), defaultBootstrapManifestURLs...)
}

func normalizeBootstrapPeers(rawPeers []string) ([]string, error) {
	if len(rawPeers) > maxBootstrapManifestPeers {
		return nil, fmt.Errorf("bootstrap peer list exceeds the %d peer limit", maxBootstrapManifestPeers)
	}
	peers := make([]string, 0, len(rawPeers))
	seen := make(map[string]struct{}, len(rawPeers))
	for _, rawPeer := range rawPeers {
		peer, err := normalizePeer(rawPeer)
		if err != nil {
			return nil, fmt.Errorf("bootstrap peer %q: %w", rawPeer, err)
		}
		if err := validateBootstrapPeerURL(peer); err != nil {
			return nil, fmt.Errorf("bootstrap peer %q: %w", rawPeer, err)
		}
		if _, duplicate := seen[peer]; duplicate {
			continue
		}
		seen[peer] = struct{}{}
		peers = append(peers, peer)
	}
	sort.Strings(peers)
	return peers, nil
}

func normalizeBootstrapManifestURLs(rawSources []string) ([]string, error) {
	if len(rawSources) > maxBootstrapManifestURLs {
		return nil, fmt.Errorf("bootstrap manifest list exceeds the %d source limit", maxBootstrapManifestURLs)
	}
	sources := make([]string, 0, len(rawSources))
	seen := make(map[string]struct{}, len(rawSources))
	for _, rawSource := range rawSources {
		source := strings.TrimSpace(rawSource)
		if err := validateBootstrapManifestURL(source); err != nil {
			return nil, fmt.Errorf("bootstrap manifest %q: %w", rawSource, err)
		}
		if _, duplicate := seen[source]; duplicate {
			continue
		}
		seen[source] = struct{}{}
		sources = append(sources, source)
	}
	return sources, nil
}

func fetchBootstrapManifest(ctx context.Context, client *http.Client, source string) ([]string, error) {
	if client == nil {
		return nil, fmt.Errorf("bootstrap HTTP client is required")
	}
	if err := validateBootstrapManifestURL(source); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("create bootstrap request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch bootstrap manifest: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bootstrap manifest returned HTTP %d", response.StatusCode)
	}
	var manifest bootstrapManifest
	if err := decodeLimitedJSON(response.Body, maxBootstrapManifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode bootstrap manifest: %w", err)
	}
	if manifest.Version != bootstrapManifestVersion {
		return nil, fmt.Errorf("bootstrap manifest version %d is not supported", manifest.Version)
	}
	if manifest.Protocol != ledger.ProtocolName {
		return nil, fmt.Errorf("bootstrap manifest is for an incompatible Entcoin protocol")
	}
	if len(manifest.Peers) == 0 {
		return nil, fmt.Errorf("bootstrap manifest contains no public peers")
	}
	if len(manifest.Peers) > maxBootstrapManifestPeers {
		return nil, fmt.Errorf("bootstrap manifest exceeds the %d peer limit", maxBootstrapManifestPeers)
	}
	peers, err := normalizeBootstrapPeers(manifest.Peers)
	if err != nil {
		return nil, err
	}
	if len(peers) == 0 {
		return nil, fmt.Errorf("bootstrap manifest contains no unique public peers")
	}
	return peers, nil
}

func validateBootstrapManifestURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("bootstrap manifest URL is invalid")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && isLoopbackHostname(parsed.Hostname()) {
		return nil
	}
	return fmt.Errorf("bootstrap manifest must use HTTPS")
}

func validateBootstrapPeerURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("URL is invalid")
	}
	if parsed.Scheme == "http" && isLoopbackHostname(parsed.Hostname()) {
		return nil
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("public bootstrap peers must use HTTPS")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return fmt.Errorf("public bootstrap peers must use HTTPS port 443")
	}
	host := strings.ToLower(parsed.Hostname())
	if address := net.ParseIP(host); address != nil {
		parsedAddress, ok := netipAddrFromIP(address)
		if !ok || !isPublicPeerIP(parsedAddress) {
			return fmt.Errorf("public bootstrap peer IP is not globally routable")
		}
		return nil
	}
	if !validPublicBootstrapDNSName(host) {
		return fmt.Errorf("public bootstrap peer host must be a public DNS name")
	}
	return nil
}

func validPublicBootstrapDNSName(host string) bool {
	if !validDNSName(host) || !strings.Contains(host, ".") {
		return false
	}
	for _, suffix := range []string{".local", ".localhost", ".internal", ".home", ".lan", ".test", ".invalid", ".example"} {
		if strings.HasSuffix(host, suffix) {
			return false
		}
	}
	return true
}

func netipAddrFromIP(ip net.IP) (netip.Addr, bool) {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return netip.Addr{}, false
	}
	return address.Unmap(), true
}

func isLoopbackHostname(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Service) bootstrapLoop(ctx context.Context) {
	defer s.wait.Done()
	delay := time.Duration(0)
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
		if err := s.refreshBootstrap(ctx); err != nil {
			delay = bootstrapRetryInterval
			continue
		}
		delay = bootstrapRefreshInterval
	}
}

func (s *Service) refreshBootstrap(ctx context.Context) error {
	s.mu.Lock()
	s.bootstrapAttempt = time.Now()
	sources := append([]string(nil), s.bootstrapURLs...)
	s.mu.Unlock()

	var failures []error
	for _, source := range sources {
		requestContext, cancel := context.WithTimeout(ctx, 12*time.Second)
		peers, err := fetchBootstrapManifest(requestContext, s.client, source)
		cancel()
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", source, err))
			continue
		}
		for _, peer := range peers {
			if err := s.addBootstrapPeer(ctx, peer); err != nil && !errors.Is(err, context.Canceled) {
				failures = append(failures, err)
			}
		}
		s.mu.Lock()
		s.bootstrapSuccess = time.Now()
		s.bootstrapError = ""
		s.mu.Unlock()
		return nil
	}
	err := errors.Join(failures...)
	if err == nil {
		err = fmt.Errorf("no bootstrap manifest sources are configured")
	}
	s.mu.Lock()
	s.bootstrapError = err.Error()
	s.mu.Unlock()
	return err
}

func (s *Service) addBootstrapPeer(ctx context.Context, raw string) error {
	peer, err := normalizePeer(raw)
	if err != nil {
		return err
	}
	if err := validateBootstrapPeerURL(peer); err != nil {
		return err
	}
	_, publicErr := normalizePublicPeer(peer)
	return s.persistDiscoveredPeer(ctx, peer, publicErr == nil, true)
}
