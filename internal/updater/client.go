package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	CurrentVersion   = "1.0.6"
	ReleasesURL      = "https://github.com/HONG-LOU/entcoin/releases/latest"
	releaseFeedURL   = "https://github.com/HONG-LOU/entcoin/releases.atom"
	maxMetadataBytes = 1 << 20
	maxChecksumBytes = 128 << 10
	maxArtifactBytes = 300 << 20
)

type Status struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Available      bool   `json:"available"`
	ReleaseURL     string `json:"release_url"`
	PublishedAt    string `json:"published_at"`
	AssetName      string `json:"asset_name"`
}

type PreparedUpdate struct {
	Status Status
	Path   string
}

type Client struct {
	feedURL      string
	downloadBase string
	httpClient   *http.Client
	platform     string
	architecture string
	cacheRoot    string
	validateURL  func(string) error
}

type atomFeed struct {
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title   string     `xml:"title"`
	Updated string     `xml:"updated"`
	Links   []atomLink `xml:"link"`
}

type atomLink struct {
	Relation string `xml:"rel,attr"`
	Type     string `xml:"type,attr"`
	Address  string `xml:"href,attr"`
}

type releaseAsset struct {
	Name string
	URL  string
}

type releaseSelection struct {
	status   Status
	artifact releaseAsset
	checksum releaseAsset
}

func New() *Client {
	return &Client{
		feedURL:      releaseFeedURL,
		downloadBase: "https://github.com/HONG-LOU/entcoin/releases/download/",
		httpClient:   secureHTTPClient(),
		platform:     runtime.GOOS,
		architecture: runtime.GOARCH,
		validateURL:  validateGitHubURL,
	}
}

func (c *Client) Check(ctx context.Context) (Status, error) {
	selection, err := c.latest(ctx)
	if err != nil {
		return Status{}, err
	}
	return selection.status, nil
}

func (c *Client) PrepareLatest(ctx context.Context) (PreparedUpdate, error) {
	selection, err := c.latest(ctx)
	if err != nil {
		return PreparedUpdate{}, err
	}
	if !selection.status.Available {
		return PreparedUpdate{}, errors.New("Entcoin is already up to date")
	}
	checksumData, err := c.read(ctx, selection.checksum.URL, maxChecksumBytes)
	if err != nil {
		return PreparedUpdate{}, fmt.Errorf("download release checksums: %w", err)
	}
	expected, err := checksumFor(checksumData, selection.artifact.Name)
	if err != nil {
		return PreparedUpdate{}, err
	}
	path, err := c.downloadArtifact(ctx, selection.artifact, expected)
	if err != nil {
		return PreparedUpdate{}, err
	}
	return PreparedUpdate{Status: selection.status, Path: path}, nil
}

func (c *Client) latest(ctx context.Context) (releaseSelection, error) {
	data, err := c.read(ctx, c.feedURL, maxMetadataBytes)
	if err != nil {
		return releaseSelection{}, fmt.Errorf("check latest release: %w", err)
	}
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return releaseSelection{}, fmt.Errorf("decode release feed: %w", err)
	}
	entry, latestVersion, err := latestStableEntry(feed.Entries)
	if err != nil {
		return releaseSelection{}, err
	}
	comparison, err := compareVersions(latestVersion, CurrentVersion)
	if err != nil {
		return releaseSelection{}, fmt.Errorf("validate release version: %w", err)
	}
	releaseURL, err := entry.releaseURL(latestVersion)
	if err != nil {
		return releaseSelection{}, err
	}
	status := Status{
		CurrentVersion: CurrentVersion,
		LatestVersion:  latestVersion,
		Available:      comparison > 0,
		ReleaseURL:     releaseURL,
		PublishedAt:    entry.Updated,
	}
	if !status.Available {
		return releaseSelection{status: status}, nil
	}
	artifactName, checksumName, err := assetNames(c.platform, c.architecture, latestVersion)
	if err != nil {
		return releaseSelection{}, err
	}
	assetBase := c.downloadBase + "v" + latestVersion + "/"
	artifact := releaseAsset{Name: artifactName, URL: assetBase + artifactName}
	checksum := releaseAsset{Name: checksumName, URL: assetBase + checksumName}
	status.AssetName = artifactName
	return releaseSelection{status: status, artifact: artifact, checksum: checksum}, nil
}

func (c *Client) read(ctx context.Context, address string, limit int64) ([]byte, error) {
	if err := c.validateURL(address); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/atom+xml, application/octet-stream")
	request.Header.Set("User-Agent", "Entcoin/"+CurrentVersion)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %s", response.Status)
	}
	if response.ContentLength > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return data, nil
}

func (c *Client) downloadArtifact(ctx context.Context, asset releaseAsset, expected string) (string, error) {
	cacheRoot := c.cacheRoot
	if cacheRoot == "" {
		userCache, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("find user cache directory: %w", err)
		}
		cacheRoot = filepath.Join(userCache, "Entcoin", "updates")
	}
	if err := os.MkdirAll(cacheRoot, 0o700); err != nil {
		return "", fmt.Errorf("create update cache: %w", err)
	}
	destination := filepath.Join(cacheRoot, asset.Name)
	if valid, err := fileMatches(destination, expected); err != nil {
		return "", err
	} else if valid {
		return destination, nil
	}

	if err := c.validateURL(asset.URL); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", "Entcoin/"+CurrentVersion)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("download update: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download update: unexpected HTTP status %s", response.Status)
	}
	if response.ContentLength > maxArtifactBytes {
		return "", errors.New("download update: response exceeds the maximum artifact size")
	}

	temporary, err := os.CreateTemp(cacheRoot, ".entcoin-update-*")
	if err != nil {
		return "", fmt.Errorf("create update file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(temporary, hash), io.LimitReader(response.Body, maxArtifactBytes+1))
	closeErr := temporary.Close()
	if copyErr != nil {
		return "", fmt.Errorf("write update file: %w", copyErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close update file: %w", closeErr)
	}
	if written > maxArtifactBytes || (response.ContentLength >= 0 && written != response.ContentLength) {
		return "", errors.New("downloaded update size does not match the HTTP response")
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return "", errors.New("downloaded update failed SHA-256 verification")
	}
	if err := os.Chmod(temporaryPath, 0o700); err != nil {
		return "", fmt.Errorf("secure update file: %w", err)
	}
	if err := os.Remove(destination); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("replace cached update: %w", err)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return "", fmt.Errorf("store verified update: %w", err)
	}
	return destination, nil
}

func secureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Minute,
		CheckRedirect: func(request *http.Request, _ []*http.Request) error {
			return validateGitHubURL(request.URL.String())
		},
	}
}

func validateGitHubURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	host := strings.ToLower(parsed.Hostname())
	trusted := host == "github.com" || host == "api.github.com" ||
		strings.HasSuffix(host, ".githubusercontent.com")
	if parsed.Scheme != "https" || parsed.User != nil || !trusted {
		return errors.New("release URL is not a trusted GitHub HTTPS URL")
	}
	return nil
}

func validateGitHubReleaseURL(raw, version string) error {
	if err := validateGitHubURL(raw); err != nil {
		return fmt.Errorf("validate release page URL: %w", err)
	}
	parsed, _ := url.Parse(raw)
	expectedPath := "/HONG-LOU/entcoin/releases/tag/v" + version
	if parsed.Hostname() != "github.com" || parsed.EscapedPath() != expectedPath {
		return errors.New("release page URL does not belong to Entcoin")
	}
	return nil
}

func compareVersions(left, right string) (int, error) {
	leftParts, err := parseVersion(left)
	if err != nil {
		return 0, err
	}
	rightParts, err := parseVersion(right)
	if err != nil {
		return 0, err
	}
	for index := range leftParts {
		if leftParts[index] < rightParts[index] {
			return -1, nil
		}
		if leftParts[index] > rightParts[index] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseVersion(version string) ([3]int, error) {
	var result [3]int
	parts := strings.Split(version, ".")
	if len(parts) != len(result) {
		return result, fmt.Errorf("version %q must contain three numeric components", version)
	}
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return result, fmt.Errorf("version %q is not canonical", version)
		}
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return result, fmt.Errorf("version %q is invalid", version)
		}
		result[index] = value
	}
	return result, nil
}

func assetNames(platform, architecture, version string) (string, string, error) {
	if architecture != "amd64" {
		return "", "", fmt.Errorf("automatic updates do not support %s/%s", platform, architecture)
	}
	switch platform {
	case "linux":
		return "entcoin_" + version + "_amd64.deb", "SHA256SUMS-linux.txt", nil
	case "windows":
		return "entcoin-amd64-installer.exe", "SHA256SUMS.txt", nil
	default:
		return "", "", fmt.Errorf("automatic updates do not support %s/%s", platform, architecture)
	}
}

func checksumFor(data []byte, filename string) (string, error) {
	match := ""
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != filename {
			continue
		}
		if match != "" {
			return "", fmt.Errorf("checksum list contains duplicate entry for %s", filename)
		}
		candidate := strings.ToLower(fields[0])
		decoded, err := hex.DecodeString(candidate)
		if err != nil || len(decoded) != sha256.Size {
			return "", fmt.Errorf("checksum for %s is invalid", filename)
		}
		match = candidate
	}
	if match == "" {
		return "", fmt.Errorf("checksum list does not contain %s", filename)
	}
	return match, nil
}

func fileMatches(path, expected string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect cached update: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxArtifactBytes {
		return false, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open cached update: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, fmt.Errorf("verify cached update: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)) == expected, nil
}

func latestStableEntry(entries []atomEntry) (atomEntry, string, error) {
	var selected atomEntry
	selectedVersion := ""
	for _, entry := range entries {
		version := strings.TrimPrefix(strings.TrimSpace(entry.Title), "v")
		if _, err := parseVersion(version); err != nil {
			continue
		}
		if selectedVersion == "" {
			selected = entry
			selectedVersion = version
			continue
		}
		comparison, _ := compareVersions(version, selectedVersion)
		if comparison > 0 {
			selected = entry
			selectedVersion = version
		}
	}
	if selectedVersion == "" {
		return atomEntry{}, "", errors.New("release feed does not contain a stable version")
	}
	return selected, selectedVersion, nil
}

func (entry atomEntry) releaseURL(version string) (string, error) {
	for _, link := range entry.Links {
		if link.Relation != "alternate" || link.Type != "text/html" {
			continue
		}
		if err := validateGitHubReleaseURL(link.Address, version); err != nil {
			return "", err
		}
		return link.Address, nil
	}
	return "", errors.New("stable release is missing its GitHub page")
}
