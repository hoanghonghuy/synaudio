package audit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ActorUser      = "USER"
	ActorSystem    = "SYSTEM"
	ActorAI        = "AI"
	ActorAnonymous = "ANONYMOUS"

	ResultSucceeded = "SUCCEEDED"
	ResultFailed    = "FAILED"
	ResultDenied    = "DENIED"
)

var ErrNotFound = errors.New("audit event not found")

type Event struct {
	ID              string         `json:"id"`
	ActorUserID     string         `json:"actor_user_id,omitempty"`
	ActorType       string         `json:"actor_type"`
	Action          string         `json:"action"`
	ResourceType    string         `json:"resource_type,omitempty"`
	ResourceID      string         `json:"resource_id,omitempty"`
	StoryID         string         `json:"story_id,omitempty"`
	ChapterID       string         `json:"chapter_id,omitempty"`
	Result          string         `json:"result"`
	CorrelationID   string         `json:"correlation_id,omitempty"`
	RequestID       string         `json:"request_id,omitempty"`
	GenerationRunID string         `json:"generation_run_id,omitempty"`
	Provenance      map[string]any `json:"provenance,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

type Filter struct {
	ActorUserID     string
	Action          string
	ResourceType    string
	ResourceID      string
	StoryID         string
	ChapterID       string
	GenerationRunID string
	CorrelationID   string
	Result          string
	From            time.Time
	To              time.Time
	Limit           int
}

type Store interface {
	CreateAuditEvent(ctx context.Context, event Event) (Event, error)
	GetAuditEvent(ctx context.Context, id string) (Event, error)
	ListAuditEvents(ctx context.Context, filter Filter) ([]Event, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) Record(ctx context.Context, event Event) (Event, error) {
	if s == nil || s.store == nil {
		return Event{}, errors.New("audit store not configured")
	}
	event.Action = strings.TrimSpace(event.Action)
	event.ActorType = strings.ToUpper(strings.TrimSpace(event.ActorType))
	event.Result = strings.ToUpper(strings.TrimSpace(event.Result))
	if event.Action == "" {
		return Event{}, errors.New("audit action is required")
	}
	if !validActorType(event.ActorType) {
		return Event{}, fmt.Errorf("invalid audit actor type %q", event.ActorType)
	}
	if !validResult(event.Result) {
		return Event{}, fmt.Errorf("invalid audit result %q", event.Result)
	}
	if event.ActorType == ActorUser && strings.TrimSpace(event.ActorUserID) == "" {
		return Event{}, errors.New("USER audit actor requires actor_user_id")
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	event.Provenance = SanitizeMetadata(event.Provenance)
	event.Metadata = SanitizeMetadata(event.Metadata)
	if event.CreatedAt.IsZero() {
		event.CreatedAt = s.now().UTC()
	}
	return s.store.CreateAuditEvent(ctx, event)
}

func (s *Service) Get(ctx context.Context, id string) (Event, error) {
	if strings.TrimSpace(id) == "" {
		return Event{}, ErrNotFound
	}
	return s.store.GetAuditEvent(ctx, id)
}

func (s *Service) List(ctx context.Context, filter Filter) ([]Event, error) {
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 500 {
		filter.Limit = 500
	}
	return s.store.ListAuditEvents(ctx, filter)
}

func validActorType(value string) bool {
	switch value {
	case ActorUser, ActorSystem, ActorAI, ActorAnonymous:
		return true
	default:
		return false
	}
}

func validResult(value string) bool {
	switch value {
	case ResultSucceeded, ResultFailed, ResultDenied:
		return true
	default:
		return false
	}
}

var sensitiveFragments = []string{
	"password", "passwd", "authorization", "access_token", "refreshtoken", "refresh_token",
	"tokenhash", "token_hash", "totp", "mfa_secret", "recovery_code", "recoverycode",
	"api_key", "apikey", "secret_key", "secretkey", "provider_secret", "cookie",
}

// SanitizeMetadata recursively redacts values whose keys could contain secrets.
// Audit records should prefer stable IDs/hashes and never raw credentials.
func SanitizeMetadata(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		if sensitiveKey(key) {
			out[key] = "[REDACTED]"
			continue
		}
		out[key] = sanitizeValue(value)
	}
	return out
}

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return SanitizeMetadata(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = sanitizeValue(item)
		}
		return out
	default:
		return value
	}
}

func sensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	for _, fragment := range sensitiveFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
