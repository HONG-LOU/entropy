package ledger

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPeerAndHealthPersistence(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	ledger, err := Open(ctx, directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{"", "ftp://localhost:1", "http://user:pass@localhost:1", "http://localhost:0", "http://localhost:1/path"} {
		if err := ledger.UpsertPeer(ctx, invalid, true); err == nil {
			t.Fatalf("invalid peer %q was stored", invalid)
		}
	}
	manualURL := "http://localhost:47821"
	if err := ledger.UpsertPeer(ctx, manualURL+"/", true); err != nil {
		t.Fatalf("store manual peer: %v", err)
	}
	if err := ledger.UpsertPeer(ctx, manualURL, false); err != nil {
		t.Fatalf("update manual peer: %v", err)
	}
	nextAttempt := time.Now().Add(time.Minute).Truncate(time.Second)
	if err := ledger.RecordPeerFailure(ctx, manualURL, nextAttempt, errors.New("dial failed")); err != nil {
		t.Fatalf("record peer failure: %v", err)
	}
	peers, err := ledger.Peers(ctx)
	if err != nil || len(peers) != 1 {
		t.Fatalf("peers after failure = %#v, err %v", peers, err)
	}
	if !peers[0].Manual || peers[0].Failures != 1 || peers[0].LastError != "dial failed" || !peers[0].NextAttempt.Equal(nextAttempt) {
		t.Fatalf("failed peer state = %#v", peers[0])
	}
	seenAt := time.Now().Truncate(time.Second)
	if err := ledger.RecordPeerSuccess(ctx, manualURL, seenAt); err != nil {
		t.Fatalf("record peer success: %v", err)
	}
	discoveredURL := "https://127.0.0.1:47822"
	if err := ledger.UpsertPeer(ctx, discoveredURL, false); err != nil {
		t.Fatalf("store discovered peer: %v", err)
	}
	if err := ledger.RecordPeerFailure(ctx, discoveredURL, time.Time{}, errors.New(strings.Repeat("x", maxPeerErrorBytes+100))); err != nil {
		t.Fatalf("record discovered peer failure: %v", err)
	}
	peers, err = ledger.Peers(ctx)
	if err != nil || len(peers) != 2 {
		t.Fatalf("stored peers = %#v, err %v", peers, err)
	}
	if !peers[0].Manual || peers[0].Failures != 0 || peers[0].LastError != "" || !peers[0].LastSeen.Equal(seenAt) {
		t.Fatalf("successful manual peer state = %#v", peers[0])
	}
	if peers[1].Manual || peers[1].Failures != 1 || len(peers[1].LastError) != maxPeerErrorBytes {
		t.Fatalf("discovered peer state = %#v", peers[1])
	}
	if err := ledger.RemovePeer(ctx, discoveredURL); err != nil {
		t.Fatalf("remove discovered peer: %v", err)
	}
	if err := ledger.RecordPeerFailure(ctx, discoveredURL, time.Now().Add(time.Minute), errors.New("delayed failure")); err != nil {
		t.Fatalf("record delayed peer failure: %v", err)
	}
	if err := ledger.RecordPeerSuccess(ctx, discoveredURL, time.Now()); err != nil {
		t.Fatalf("record delayed peer success: %v", err)
	}
	peers, err = ledger.Peers(ctx)
	if err != nil || len(peers) != 1 || peers[0].URL != manualURL {
		t.Fatalf("delayed records resurrected removed peer: %#v, err %v", peers, err)
	}
	for range 16 {
		if err := ledger.UpsertPeer(ctx, discoveredURL, false); err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		errorsSeen := make(chan error, 2)
		var race sync.WaitGroup
		race.Add(2)
		go func() {
			defer race.Done()
			<-start
			errorsSeen <- ledger.RecordPeerFailure(ctx, discoveredURL, time.Now(), errors.New("in flight"))
		}()
		go func() {
			defer race.Done()
			<-start
			errorsSeen <- ledger.RemovePeer(ctx, discoveredURL)
		}()
		close(start)
		race.Wait()
		close(errorsSeen)
		for recordErr := range errorsSeen {
			if recordErr != nil {
				t.Fatal(recordErr)
			}
		}
		peers, err = ledger.Peers(ctx)
		if err != nil || len(peers) != 1 || peers[0].URL != manualURL {
			t.Fatalf("concurrent record resurrected removed peer: %#v, err %v", peers, err)
		}
	}

	if _, err := ledger.AddHealthEvent(ctx, HealthEvent{Code: "bad code", Severity: "panic", Message: "bad"}); err == nil {
		t.Fatal("invalid health event was stored")
	}
	eventID, err := ledger.AddHealthEvent(ctx, HealthEvent{
		Code:     " disk.quick_check ",
		Severity: " warning ",
		Message:  " integrity check should run ",
		Action:   " restart node ",
	})
	if err != nil {
		t.Fatalf("add health event: %v", err)
	}
	active, err := ledger.HealthEvents(ctx, true, 10)
	if err != nil || len(active) != 1 || active[0].ID != eventID || active[0].Code != "disk.quick_check" || active[0].Resolved {
		t.Fatalf("active health events = %#v, err %v", active, err)
	}
	if err := ledger.ResolveHealthEvent(ctx, eventID); err != nil {
		t.Fatalf("resolve health event: %v", err)
	}
	if err := ledger.ResolveHealthEvent(ctx, eventID+1); !errors.Is(err, ErrHealthEventNotFound) {
		t.Fatalf("missing health event resolution error = %v", err)
	}
	active, err = ledger.HealthEvents(ctx, true, 10)
	if err != nil || len(active) != 0 {
		t.Fatalf("active events after resolution = %#v, err %v", active, err)
	}
	if depth, err := ledger.PruneDepth(ctx); err != nil || depth != 0 {
		t.Fatalf("default prune depth = %d, err %v", depth, err)
	}
	if err := ledger.SetPruneDepth(ctx, 1); err == nil {
		t.Fatal("unsafe prune depth was accepted")
	}
	if err := ledger.SetPruneDepth(ctx, MaximumPruneDepth+1); err == nil {
		t.Fatal("excessive prune depth was accepted")
	}
	if err := ledger.SetPruneDepth(ctx, MinimumPruneDepth); err != nil {
		t.Fatalf("store prune depth: %v", err)
	}
	if err := ledger.Close(); err != nil {
		t.Fatalf("close ledger: %v", err)
	}

	reopened, err := Open(ctx, directory)
	if err != nil {
		t.Fatalf("reopen ledger: %v", err)
	}
	defer reopened.Close()
	peers, err = reopened.Peers(ctx)
	if err != nil || len(peers) != 1 || peers[0].URL != manualURL || !peers[0].Manual {
		t.Fatalf("reopened peers = %#v, err %v", peers, err)
	}
	events, err := reopened.HealthEvents(ctx, false, 10)
	if err != nil || len(events) != 1 || !events[0].Resolved {
		t.Fatalf("reopened health events = %#v, err %v", events, err)
	}
	if depth, err := reopened.PruneDepth(ctx); err != nil || depth != MinimumPruneDepth {
		t.Fatalf("reopened prune depth = %d, err %v", depth, err)
	}
	if err := reopened.SetPruneDepth(ctx, 0); err != nil {
		t.Fatalf("switch to archive mode: %v", err)
	}
	if depth, err := reopened.PruneDepth(ctx); err != nil || depth != 0 {
		t.Fatalf("archive prune depth = %d, err %v", depth, err)
	}
}

func TestOpenRejectsUnsafeDatabasePath(t *testing.T) {
	ctx := context.Background()
	directoryPath := t.TempDir()
	if err := os.Mkdir(filepath.Join(directoryPath, DatabaseName), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(ctx, directoryPath); err == nil {
		t.Fatal("database directory was accepted as a regular database file")
	}

	symlinkPath := t.TempDir()
	target := filepath.Join(t.TempDir(), "target.db")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(symlinkPath, DatabaseName)); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if _, err := Open(ctx, symlinkPath); err == nil {
		t.Fatal("symbolic-link database was accepted")
	}
}

func TestRecentBlocksConcurrentQueries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ledger, err := Open(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	var wait sync.WaitGroup
	errorsFound := make(chan error, 16)
	for index := 0; index < 16; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for request := 0; request < 10; request++ {
				blocks, err := ledger.RecentBlocks(ctx, 10)
				if err != nil {
					errorsFound <- err
					return
				}
				if len(blocks) != 1 || blocks[0].Height != 0 {
					errorsFound <- errors.New("recent genesis query returned an unexpected result")
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
}
