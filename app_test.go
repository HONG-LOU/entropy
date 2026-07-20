package main

import (
	"context"
	"sync"
	"testing"

	"github.com/HONG-LOU/entcoin/internal/node"
)

func TestConcurrentStartNodePreservesRunningService(t *testing.T) {
	running := &node.Service{}
	app := &App{service: running}

	const attempts = 16
	ready := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(attempts)
	for range attempts {
		go func() {
			defer wait.Done()
			<-ready
			app.startNode(context.Background())
		}()
	}
	close(ready)
	wait.Wait()

	app.mu.RLock()
	defer app.mu.RUnlock()
	if app.service != running || app.start != nil {
		t.Fatalf("concurrent startup replaced ready service: service=%p error=%v", app.service, app.start)
	}
}
