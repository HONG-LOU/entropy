package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxPeerErrorBytes = 2_048

func (l *Ledger) UpsertPeer(ctx context.Context, rawURL string, manual bool) error {
	peerURL, err := normalizeStoredPeerURL(rawURL)
	if err != nil {
		return err
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	if _, err := l.database.ExecContext(ctx, `
		INSERT INTO peers(url, manual, added_at) VALUES(?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			manual = CASE WHEN excluded.manual = 1 THEN 1 ELSE peers.manual END
	`, peerURL, manual, time.Now().Unix()); err != nil {
		return fmt.Errorf("store peer %s: %w", peerURL, err)
	}
	return nil
}

func (l *Ledger) RemovePeer(ctx context.Context, rawURL string) error {
	peerURL, err := normalizeStoredPeerURL(rawURL)
	if err != nil {
		return err
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	if _, err := l.database.ExecContext(ctx, "DELETE FROM peers WHERE url = ?", peerURL); err != nil {
		return fmt.Errorf("remove peer %s: %w", peerURL, err)
	}
	return nil
}

func (l *Ledger) Peers(ctx context.Context) ([]PeerRecord, error) {
	rows, err := l.database.QueryContext(ctx, `
		SELECT url, manual, last_seen, failures, next_attempt, last_error
		FROM peers ORDER BY manual DESC, url
	`)
	if err != nil {
		return nil, fmt.Errorf("query peers: %w", err)
	}
	defer rows.Close()
	peers := make([]PeerRecord, 0)
	for rows.Next() {
		var record PeerRecord
		var manual int
		var lastSeen, nextAttempt sql.NullInt64
		if err := rows.Scan(&record.URL, &manual, &lastSeen, &record.Failures, &nextAttempt, &record.LastError); err != nil {
			return nil, fmt.Errorf("scan peer: %w", err)
		}
		if (manual != 0 && manual != 1) || record.Failures < 0 {
			return nil, fmt.Errorf("stored peer %s contains invalid values", record.URL)
		}
		record.Manual = manual == 1
		record.LastSeen = unixTime(lastSeen)
		record.NextAttempt = unixTime(nextAttempt)
		peers = append(peers, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate peers: %w", err)
	}
	return peers, nil
}

func (l *Ledger) RecordPeerSuccess(ctx context.Context, rawURL string, seenAt time.Time) error {
	peerURL, err := normalizeStoredPeerURL(rawURL)
	if err != nil {
		return err
	}
	if seenAt.IsZero() {
		seenAt = time.Now()
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	if _, err := l.database.ExecContext(ctx, `
		UPDATE peers SET last_seen = ?, failures = 0, next_attempt = NULL, last_error = ''
		WHERE url = ?
	`, seenAt.Unix(), peerURL); err != nil {
		return fmt.Errorf("record peer %s success: %w", peerURL, err)
	}
	return nil
}

func (l *Ledger) RecordPeerFailure(ctx context.Context, rawURL string, nextAttempt time.Time, cause error) error {
	peerURL, err := normalizeStoredPeerURL(rawURL)
	if err != nil {
		return err
	}
	message := "peer request failed"
	if cause != nil {
		message = strings.TrimSpace(cause.Error())
		if message == "" {
			message = "peer request failed"
		}
	}
	message = truncateUTF8(message, maxPeerErrorBytes)
	var next any
	if !nextAttempt.IsZero() {
		next = nextAttempt.Unix()
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	if _, err := l.database.ExecContext(ctx, `
		UPDATE peers SET
			failures = CASE WHEN failures < 1000000 THEN failures + 1 ELSE failures END,
			next_attempt = ?,
			last_error = ?
		WHERE url = ?
	`, next, message, peerURL); err != nil {
		return fmt.Errorf("record peer %s failure: %w", peerURL, err)
	}
	return nil
}

func normalizeStoredPeerURL(raw string) (string, error) {
	raw = strings.TrimSpace(strings.TrimRight(raw, "/"))
	if raw == "" || len(raw) > 2_048 {
		return "", fmt.Errorf("peer URL length is invalid")
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("peer must be an http(s) URL")
	}
	if parsed.User != nil || parsed.Opaque != "" || parsed.Path != "" || parsed.RawPath != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("peer URL must not contain credentials, path, query, or fragment")
	}
	hostname := parsed.Hostname()
	if net.ParseIP(hostname) == nil && !validStoredDNSName(hostname) {
		return "", fmt.Errorf("peer host must be an IP address or ASCII DNS name")
	}
	if port := parsed.Port(); port != "" {
		value, err := strconv.ParseUint(port, 10, 16)
		if err != nil || value == 0 {
			return "", fmt.Errorf("peer port must be between 1 and 65535")
		}
	}
	return parsed.String(), nil
}

func validStoredDNSName(hostname string) bool {
	if hostname == "" || len(hostname) > 253 || strings.HasPrefix(hostname, ".") || strings.HasSuffix(hostname, ".") {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
				(character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}
	return true
}

func truncateUTF8(value string, maximumBytes int) string {
	if len(value) <= maximumBytes {
		return value
	}
	for maximumBytes > 0 && !utf8Start(value[maximumBytes]) {
		maximumBytes--
	}
	return value[:maximumBytes]
}

func utf8Start(value byte) bool {
	return value&0xc0 != 0x80
}
