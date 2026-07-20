package node

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"strconv"
	"time"

	"github.com/HONG-LOU/entcoin/internal/core"
	"github.com/HONG-LOU/entcoin/internal/ledger"
)

const discoveryGroup = "239.255.78.21:47822"

type discoveryAnnouncement struct {
	Protocol string `json:"protocol"`
	NodeID   string `json:"node_id"`
	Port     int    `json:"port"`
}

func (s *Service) discoveryLoop(ctx context.Context) {
	defer s.wait.Done()
	group, err := net.ResolveUDPAddr("udp4", discoveryGroup)
	if err != nil {
		return
	}
	listener, err := net.ListenMulticastUDP("udp4", nil, group)
	if err != nil {
		return
	}
	defer listener.Close()
	_ = listener.SetReadBuffer(64 << 10)
	sender, err := net.DialUDP("udp4", nil, group)
	if err != nil {
		return
	}
	defer sender.Close()
	nextAnnouncement := time.Time{}
	buffer := make([]byte, 2<<10)
	for {
		if time.Now().After(nextAnnouncement) {
			s.sendDiscoveryAnnouncement(sender)
			nextAnnouncement = time.Now().Add(5 * time.Second)
		}
		_ = listener.SetReadDeadline(time.Now().Add(time.Second))
		count, remote, readErr := listener.ReadFromUDP(buffer)
		if readErr == nil {
			s.acceptDiscoveryAnnouncement(buffer[:count], remote)
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (s *Service) sendDiscoveryAnnouncement(sender *net.UDPConn) {
	port := s.listenPort()
	if port <= 0 {
		return
	}
	data, err := json.Marshal(discoveryAnnouncement{
		Protocol: ledger.ProtocolName,
		NodeID:   s.Address(),
		Port:     port,
	})
	if err == nil {
		_, _ = sender.Write(data)
	}
}

func (s *Service) acceptDiscoveryAnnouncement(data []byte, remote *net.UDPAddr) {
	if remote == nil || len(data) == 0 || len(data) > 2<<10 {
		return
	}
	var announcement discoveryAnnouncement
	if err := decodeLimitedJSON(bytes.NewReader(data), 2<<10, &announcement); err != nil {
		return
	}
	if announcement.Protocol != ledger.ProtocolName || core.ValidateAddress(announcement.NodeID) != nil ||
		announcement.NodeID == s.Address() || announcement.Port <= 0 || announcement.Port > 65535 {
		return
	}
	peer := "http://" + net.JoinHostPort(remote.IP.String(), strconv.Itoa(announcement.Port))
	s.addDiscoveredPeer(peer)
}
