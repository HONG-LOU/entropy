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

const testReleaseVersion = "1.0.11"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{left: "1.0.7", right: "1.0.7"},
		{left: "1.0.8", right: "1.0.7", want: 1},
		{left: "1.1.0", right: "1.9.9", want: -1},
	}
	for _, test := range tests {
		got, err := compareVersions(test.left, test.right)
		if err != nil || got != test.want {
			t.Fatalf("compareVersions(%q, %q) = %d, %v; want %d", test.left, test.right, got, err, test.want)
		}
	}
	if _, err := compareVersions("1.0", "1.0.7"); err == nil {
		t.Fatal("non-canonical version was accepted")
	}
}

func TestLatestStableEntryIgnoresPrereleasesAndSelectsHighestVersion(t *testing.T) {
	entries := []atomEntry{
		{Title: "v1.0.7"},
		{Title: "v1.0.12-rc1"},
		{Title: "v1.0.9"},
		{Title: "v1.0.10"},
		{Title: "v1.0.11"},
	}

	entry, version, err := latestStableEntry(entries)
	if err != nil {
		t.Fatal(err)
	}
	if version != testReleaseVersion || entry.Title != "v"+testReleaseVersion {
		t.Fatalf("latest stable entry = %q (%q), want v%s", entry.Title, version, testReleaseVersion)
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
	if !status.Available || status.CurrentVersion != CurrentVersion || status.LatestVersion != testReleaseVersion {
		t.Fatalf("unexpected update status: %#v", status)
	}
	if status.AssetName != "entcoin_"+testReleaseVersion+"_amd64.deb" {
		t.Fatalf("asset = %q", status.AssetName)
	}
}

func TestPrepareLatestVerifiesAndCachesArtifact(t *testing.T) {
	artifact := []byte("verified deb payload")
	client, server := testClient(t, artifact, false)
	defer server.Close()
	var progress []Progress

	prepared, err := client.PrepareLatest(context.Background(), func(update Progress) {
		progress = append(progress, update)
	})
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
	if len(progress) < 4 || progress[0].Phase != "preparing" || progress[len(progress)-1].Phase != "verifying" {
		t.Fatalf("progress events = %#v", progress)
	}
	download := progress[len(progress)-2]
	if download.Phase != "downloading" || download.Downloaded != int64(len(artifact)) || download.Total != int64(len(artifact)) || download.Percent != 100 {
		t.Fatalf("final download progress = %#v", download)
	}
}

func TestProgressPercent(t *testing.T) {
	tests := []struct {
		downloaded int64
		total      int64
		want       int
	}{
		{downloaded: 0, total: 100, want: 0},
		{downloaded: 67, total: 100, want: 67},
		{downloaded: 200, total: 100, want: 100},
		{downloaded: 10, total: -1, want: 0},
	}
	for _, test := range tests {
		if got := progressPercent(test.downloaded, test.total); got != test.want {
			t.Fatalf("progressPercent(%d, %d) = %d, want %d", test.downloaded, test.total, got, test.want)
		}
	}
}

func TestPrepareLatestRejectsChecksumMismatch(t *testing.T) {
	client, server := testClient(t, []byte("tampered payload"), true)
	defer server.Close()

	_, err := client.PrepareLatest(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("checksum mismatch error = %v", err)
	}
}

func TestValidateGitHubURLRejectsUntrustedHost(t *testing.T) {
	if err := validateGitHubURL("https://github.com/HONG-LOU/entcoin/releases/download/v1.0.8/file"); err != nil {
		t.Fatal(err)
	}
	if err := validateGitHubURL("https://example.com/entcoin.exe"); err == nil {
		t.Fatal("untrusted host was accepted")
	}
	if err := validateGitHubURL("http://github.com/HONG-LOU/entcoin"); err == nil {
		t.Fatal("insecure GitHub URL was accepted")
	}
}

func TestValidateUpdateURLAllowsOnlyTheOfficialManifestOutsideGitHub(t *testing.T) {
	if err := validateUpdateURL(updateManifestURL); err != nil {
		t.Fatal(err)
	}
	if err := validateUpdateURL("https://entcoin.xyz/other.json"); err == nil {
		t.Fatal("unexpected Entcoin website URL was accepted")
	}
}

func TestCheckFallsBackToWebsiteManifest(t *testing.T) {
	client, server := testClient(t, []byte("verified deb payload"), false)
	defer server.Close()
	client.feedURL = server.URL + "/invalid-feed"
	client.manifestURL = server.URL + "/manifest"

	status, err := client.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.Available || status.LatestVersion != testReleaseVersion {
		t.Fatalf("unexpected manifest status: %#v", status)
	}
}

func TestReadRetriesTemporaryFailures(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(response, "temporary", http.StatusBadGateway)
			return
		}
		_, _ = response.Write([]byte("ready"))
	}))
	defer server.Close()
	client := &Client{httpClient: server.Client(), validateURL: func(string) error { return nil }}

	data, err := client.read(context.Background(), server.URL, 32)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ready" || attempts != 3 {
		t.Fatalf("read = %q after %d attempts", data, attempts)
	}
}

func testClient(t *testing.T, artifact []byte, mismatch bool) (*Client, *httptest.Server) {
	t.Helper()
	artifactName := "entcoin_" + testReleaseVersion + "_amd64.deb"
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
					Title:   "v" + testReleaseVersion,
					Updated: "2026-07-22T00:00:00Z",
					Links: []atomLink{{
						Relation: "alternate",
						Type:     "text/html",
						Address:  "https://github.com/HONG-LOU/entcoin/releases/tag/v" + testReleaseVersion,
					}},
				},
			}})
		case "/manifest":
			_, _ = response.Write([]byte(`{"version":"` + testReleaseVersion + `","published_at":"2026-07-22T00:00:00Z","release_url":"https://github.com/HONG-LOU/entcoin/releases/tag/v` + testReleaseVersion + `"}`))
		case "/invalid-feed":
			_, _ = response.Write([]byte("not atom"))
		case "/download/v" + testReleaseVersion + "/" + artifactName:
			_, _ = response.Write(artifact)
		case "/download/v" + testReleaseVersion + "/SHA256SUMS-linux.txt":
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
