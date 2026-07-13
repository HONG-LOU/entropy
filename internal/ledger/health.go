package ledger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	maxHealthCodeBytes    = 64
	maxHealthMessageBytes = 4_096
)

var ErrHealthEventNotFound = errors.New("health event not found")

func (l *Ledger) AddHealthEvent(ctx context.Context, event HealthEvent) (int64, error) {
	event.Code = strings.TrimSpace(event.Code)
	event.Severity = strings.TrimSpace(event.Severity)
	event.Message = strings.TrimSpace(event.Message)
	event.Action = strings.TrimSpace(event.Action)
	if err := validateHealthEvent(event); err != nil {
		return 0, err
	}
	if event.Created.IsZero() {
		event.Created = time.Now()
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	result, err := l.database.ExecContext(ctx, `
		INSERT INTO health_events(code, severity, message, action, created_at, resolved)
		VALUES(?, ?, ?, ?, ?, ?)
	`, event.Code, event.Severity, event.Message, event.Action, event.Created.Unix(), event.Resolved)
	if err != nil {
		return 0, fmt.Errorf("add health event %s: %w", event.Code, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read health event ID: %w", err)
	}
	return id, nil
}

func (l *Ledger) HealthEvents(ctx context.Context, activeOnly bool, limit int) ([]HealthEvent, error) {
	if limit <= 0 || limit > 1_000 {
		return nil, fmt.Errorf("health event limit must be between 1 and 1000")
	}
	rows, err := l.database.QueryContext(ctx, `
		SELECT id, code, severity, message, action, created_at, resolved
		FROM health_events
		WHERE ? = 0 OR resolved = 0
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, activeOnly, limit)
	if err != nil {
		return nil, fmt.Errorf("query health events: %w", err)
	}
	defer rows.Close()
	events := make([]HealthEvent, 0)
	for rows.Next() {
		var event HealthEvent
		var createdAt int64
		var resolved int
		if err := rows.Scan(&event.ID, &event.Code, &event.Severity, &event.Message, &event.Action, &createdAt, &resolved); err != nil {
			return nil, fmt.Errorf("scan health event: %w", err)
		}
		if event.ID <= 0 || (resolved != 0 && resolved != 1) {
			return nil, fmt.Errorf("stored health event contains invalid values")
		}
		event.Created = time.Unix(createdAt, 0)
		event.Resolved = resolved == 1
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate health events: %w", err)
	}
	return events, nil
}

func (l *Ledger) ResolveHealthEvent(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("health event ID must be positive")
	}
	l.writeMu.Lock()
	defer l.writeMu.Unlock()
	result, err := l.database.ExecContext(ctx, "UPDATE health_events SET resolved = 1 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("resolve health event %d: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check resolved health event %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: %d", ErrHealthEventNotFound, id)
	}
	return nil
}

func validateHealthEvent(event HealthEvent) error {
	if event.Code == "" || len(event.Code) > maxHealthCodeBytes {
		return fmt.Errorf("health event code length is invalid")
	}
	for _, character := range event.Code {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') && character != '_' && character != '-' && character != '.' {
			return fmt.Errorf("health event code contains invalid characters")
		}
	}
	switch event.Severity {
	case "info", "warning", "error", "critical":
	default:
		return fmt.Errorf("health event severity is invalid")
	}
	if event.Message == "" || len(event.Message) > maxHealthMessageBytes {
		return fmt.Errorf("health event message length is invalid")
	}
	if len(event.Action) > maxHealthMessageBytes {
		return fmt.Errorf("health event action is too long")
	}
	return nil
}
