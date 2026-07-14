package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"

	"entropy/internal/node"
)

func TestParseNodeOptionsReturnsFlagHelp(t *testing.T) {
	_, err := parseNodeOptions([]string{"-h"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("error = %v, want flag.ErrHelp", err)
	}
}

func TestParseNodeOptionsUsesDefaultBootstrapSources(t *testing.T) {
	options, err := parseNodeOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(options.bootstrapManifestURLs) != fmt.Sprint(node.DefaultBootstrapManifestURLs()) {
		t.Fatalf("bootstrap manifests = %v", options.bootstrapManifestURLs)
	}
}

func TestParseNodeOptionsOverridesBootstrapAndEnablesProxyTrust(t *testing.T) {
	options, err := parseNodeOptions([]string{
		"--bootstrap-manifest", "https://one.example.net/mainnet.json",
		"--bootstrap-manifest", "https://two.example.net/mainnet.json",
		"--trust-loopback-proxy",
		"--prune-depth", "0",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://one.example.net/mainnet.json", "https://two.example.net/mainnet.json"}
	if fmt.Sprint(options.bootstrapManifestURLs) != fmt.Sprint(want) {
		t.Fatalf("bootstrap manifests = %v, want %v", options.bootstrapManifestURLs, want)
	}
	if !options.trustLoopbackProxy {
		t.Fatal("loopback proxy trust was not enabled")
	}
	if !options.pruneDepthSet || options.pruneDepth != 0 {
		t.Fatalf("explicit archive policy was lost: set=%v depth=%d", options.pruneDepthSet, options.pruneDepth)
	}
}

func TestParseNodeOptionsDisablesBootstrap(t *testing.T) {
	options, err := parseNodeOptions([]string{"--no-bootstrap"})
	if err != nil {
		t.Fatal(err)
	}
	if len(options.bootstrapManifestURLs) != 0 {
		t.Fatalf("bootstrap manifests = %v, want none", options.bootstrapManifestURLs)
	}
}

func TestParseNodeOptionsRejectsConflictingBootstrapFlags(t *testing.T) {
	_, err := parseNodeOptions([]string{
		"--no-bootstrap",
		"--bootstrap-manifest", "https://seed.example.net/mainnet.json",
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("error = %v", err)
	}
}
