package node

import (
	"errors"
	"testing"
)

func TestChainSyncInProgressRequiresAheadPeer(t *testing.T) {
	tests := []struct {
		name           string
		peerSyncActive bool
		localHeight    uint64
		bestPeerHeight uint64
		want           bool
	}{
		{name: "ahead peer", peerSyncActive: true, localHeight: 100, bestPeerHeight: 101, want: true},
		{name: "equal peer", peerSyncActive: true, localHeight: 100, bestPeerHeight: 100},
		{name: "behind peer", peerSyncActive: true, localHeight: 100, bestPeerHeight: 90},
		{name: "idle", localHeight: 100, bestPeerHeight: 101},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := chainSyncInProgress(test.peerSyncActive, test.localHeight, test.bestPeerHeight); got != test.want {
				t.Fatalf("chainSyncInProgress() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPeerFailureDoesNotBecomeNodeWarning(t *testing.T) {
	const peerURL = "https://peer.example"
	service := &Service{
		peers:           map[string]*peerState{peerURL: {Online: true}},
		activeOutbound:  map[string]struct{}{peerURL: {}},
		outboundSockets: make(map[string]*peerSocket),
	}

	service.markPeerFailure(peerURL, errors.New("peer channel is unavailable"))

	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.lastError != "" {
		t.Fatalf("peer failure became node warning: %q", service.lastError)
	}
	if service.peers[peerURL].LastError == "" {
		t.Fatal("peer failure was not retained on the peer")
	}
}
