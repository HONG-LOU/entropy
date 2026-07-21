package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{left: "1.0.6", right: "1.0.6"},
		{left: "1.0.7", right: "1.0.6", want: 1},
		{left: "1.1.0", right: "1.9.9", want: -1},
	}
	for _, test := range tests {
		got, err := compareVersions(test.left, test.right)
		if err != nil || got != test.want {
			t.Fatalf("compareVersions(%q, %q) = %d, %v; want %d", test.left, test.right, got, err, test.want)
		}
	}
	if _, err := compareVersions("1.0", "1.0.6"); err == nil {
		t.Fatal("non-canonical version was accepted")
	}
}

func TestLatestStableEntryIgnoresPrereleasesAndSelectsHighestVersion(t *testing.T) {
	entries := []atomEntry{
		{Title: "v1.0.6"},
		{Title: "v1.0.8-rc1"},
		{Title: "v1.0.7"},
	}

	entry, version, err := latestStableEntry(entries)
	if err != nil {
		t.Fatal(err)
	}
	if version != "1.0.7" || entry.Title != "v1.0.7" {
		t.Fatalf("latest stable entry = %q (%q), want v1.0.7", entry.Title, version)
	}
}

func TestCheckSelectsLinuxUpdate(t *testing.T) {
	artifact := []byte("verified deb payload")
	client, server := testClient(t, artifact, false)
	defer server.Close()

	status, err := client.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.Available || status.CurrentVersion != CurrentVersion || status.LatestVersion != "1.0.7" {
		t.Fatalf("unexpected update status: %#v", status)
	}
	if status.AssetName != "entcoin_1.0.7_amd64.deb" {
		t.Fatalf("asset = %q", status.AssetName)
	}
}

func TestPrepareLatestVerifiesAndCachesArtifact(t *testing.T) {
	artifact := []byte("verified deb payload")
	client, server := testClient(t, artifact, false)
	defer server.Close()

	prepared, err := client.PrepareLatest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(prepared.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(artifact) {
		t.Fatalf("downloaded artifact = %q", data)
	}
	if filepath.Dir(prepared.Path) != client.cacheRoot {
		t.Fatalf("update escaped cache root: %s", prepared.Path)
	}
}

func TestPrepareLatestRejectsChecksumMismatch(t *testing.T) {
	client, server := testClient(t, []byte("tampered payload"), true)
	defer server.Close()

	_, err := client.PrepareLatest(context.Background())
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("checksum mismatch error = %v", err)
	}
}

func TestValidateGitHubURLRejectsUntrustedHost(t *testing.T) {
	if err := validateGitHubURL("https://github.com/HONG-LOU/entcoin/releases/download/v1.0.7/file"); err != nil {
		t.Fatal(err)
	}
	if err := validateGitHubURL("https://example.com/entcoin.exe"); err == nil {
		t.Fatal("untrusted host was accepted")
	}
	if err := validateGitHubURL("http://github.com/HONG-LOU/entcoin"); err == nil {
		t.Fatal("insecure GitHub URL was accepted")
	}
}

func testClient(t *testing.T, artifact []byte, mismatch bool) (*Client, *httptest.Server) {
	t.Helper()
	artifactName := "entcoin_1.0.7_amd64.deb"
	checksum := sha256.Sum256(artifact)
	checksumText := hex.EncodeToString(checksum[:])
	if mismatch {
		checksumText = strings.Repeat("0", sha256.Size*2)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/feed":
			response.Header().Set("Content-Type", "application/atom+xml")
			_ = xml.NewEncoder(response).Encode(atomFeed{Entries: []atomEntry{
				{
					Title:   "v1.0.7",
					Updated: "2026-07-22T00:00:00Z",
					Links: []atomLink{{
						Relation: "alternate",
						Type:     "text/html",
						Address:  "https://github.com/HONG-LOU/entcoin/releases/tag/v1.0.7",
					}},
				},
			}})
		case "/download/v1.0.7/" + artifactName:
			_, _ = response.Write(artifact)
		case "/download/v1.0.7/SHA256SUMS-linux.txt":
			_, _ = response.Write([]byte(checksumText + "  " + artifactName + "\n"))
		default:
			http.NotFound(response, request)
		}
	}))
	client := &Client{
		feedURL:      server.URL + "/feed",
		downloadBase: server.URL + "/download/",
		httpClient:   server.Client(),
		platform:     "linux",
		architecture: "amd64",
		cacheRoot:    t.TempDir(),
		validateURL:  func(string) error { return nil },
	}
	return client, server
}
