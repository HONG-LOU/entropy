package node

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"entropy/internal/ledger"
)

func TestFetchBootstrapManifestValidatesAndNormalizesPeers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", request.Header.Get("Accept"))
		}
		writeJSON(writer, http.StatusOK, bootstrapManifest{
			Version:  bootstrapManifestVersion,
			Protocol: ledger.ProtocolName,
			Peers: []string{
				"https://seed-b.entropy.org/",
				"https://seed-a.entropy.org",
				"https://seed-a.entropy.org/",
			},
		})
	}))
	defer server.Close()

	peers, err := fetchBootstrapManifest(context.Background(), newHTTPClient(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://seed-a.entropy.org", "https://seed-b.entropy.org"}
	if fmt.Sprint(peers) != fmt.Sprint(want) {
		t.Fatalf("peers = %v, want %v", peers, want)
	}
}

func TestFetchBootstrapManifestRejectsUnsafeDocuments(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		contains string
	}{
		{
			name:     "wrong protocol",
			manifest: `{"version":1,"protocol":"other","peers":["https://seed.entropy.org"]}`,
			contains: "incompatible",
		},
		{
			name:     "plain public HTTP",
			manifest: `{"version":1,"protocol":"` + ledger.ProtocolName + `","peers":["http://seed.entropy.org"]}`,
			contains: "must use HTTPS",
		},
		{
			name:     "unknown field",
			manifest: `{"version":1,"protocol":"` + ledger.ProtocolName + `","peers":["https://seed.entropy.org"],"extra":true}`,
			contains: "unknown",
		},
		{
			name:     "duplicate key",
			manifest: `{"version":1,"version":1,"protocol":"` + ledger.ProtocolName + `","peers":["https://seed.entropy.org"]}`,
			contains: "duplicate",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(test.manifest))
			}))
			defer server.Close()
			_, err := fetchBootstrapManifest(context.Background(), newHTTPClient(), server.URL)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func TestBootstrapManifestLimitsAndDefaultSources(t *testing.T) {
	large := strings.Repeat("x", int(maxBootstrapManifestBytes)+1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(large))
	}))
	defer server.Close()
	if _, err := fetchBootstrapManifest(context.Background(), newHTTPClient(), server.URL); err == nil {
		t.Fatal("oversized bootstrap manifest was accepted")
	}

	defaults := DefaultBootstrapManifestURLs()
	if len(defaults) < 2 {
		t.Fatalf("default bootstrap sources = %v, want at least two", defaults)
	}
	for _, source := range defaults {
		if !strings.HasPrefix(source, "https://") {
			t.Fatalf("default bootstrap source is not HTTPS: %s", source)
		}
	}
	defaults[0] = "modified"
	if DefaultBootstrapManifestURLs()[0] == "modified" {
		t.Fatal("default bootstrap source slice was mutable")
	}
}
