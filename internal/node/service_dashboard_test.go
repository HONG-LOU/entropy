package node

import "testing"

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
