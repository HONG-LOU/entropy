package main

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"

	"github.com/HONG-LOU/entcoin/internal/node"
	"github.com/HONG-LOU/entcoin/internal/updater"
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

func TestReleaseVersionMetadataMatches(t *testing.T) {
	type packageMetadata struct {
		Version string `json:"version"`
	}
	type wailsMetadata struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}

	packageData, err := os.ReadFile("frontend/package.json")
	if err != nil {
		t.Fatal(err)
	}
	var frontend packageMetadata
	if err := json.Unmarshal(packageData, &frontend); err != nil {
		t.Fatal(err)
	}
	wailsData, err := os.ReadFile("wails.json")
	if err != nil {
		t.Fatal(err)
	}
	var desktop wailsMetadata
	if err := json.Unmarshal(wailsData, &desktop); err != nil {
		t.Fatal(err)
	}
	if frontend.Version != updater.CurrentVersion || desktop.Info.ProductVersion != updater.CurrentVersion {
		t.Fatalf("release versions differ: updater=%s frontend=%s desktop=%s", updater.CurrentVersion, frontend.Version, desktop.Info.ProductVersion)
	}
}
