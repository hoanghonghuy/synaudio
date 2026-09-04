package identity

import (
	"context"
	"errors"
	"strings"
)

const PermUserStatusManage = "USER_STATUS_MANAGE"

// AdminUserSummary is the backend-authoritative identity projection used by the
// privileged Admin directory. It deliberately exposes no credential material.
type AdminUserSummary struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	DisplayName   string   `json:"display_name"`
	Status        string   `json:"status"`
	EmailVerified bool     `json:"email_verified"`
	Roles         []string `json:"roles"`
}

// AdminUserDirectoryStore is a narrow read boundary so privileged identity
// browsing does not enlarge the base Store contract used by unrelated tests.
type AdminUserDirectoryStore interface {
	ListAdminUsers(ctx context.Context, query, status string, limit int) ([]AdminUserSummary, error)
	GetAdminUser(ctx context.Context, userID string) (AdminUserSummary, error)
}

func (s *AuthService) adminUserDirectoryStore() (AdminUserDirectoryStore, error) {
	store, ok := s.store.(AdminUserDirectoryStore)
	if !ok {
		return nil, errors.New("admin user directory persistence not configured")
	}
	return store, nil
}

func (s *AuthService) ListAdminUsers(ctx context.Context, actorID, query, status string, limit int) ([]AdminUserSummary, error) {
	if ok, err := s.Authorize(ctx, actorID, PermUserStatusManage); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrForbidden
	}

	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "" && status != StatusActive && status != StatusSuspended && status != StatusDeactivated {
		return nil, errors.New("invalid account status")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	store, err := s.adminUserDirectoryStore()
	if err != nil {
		return nil, err
	}
	return store.ListAdminUsers(ctx, strings.TrimSpace(query), status, limit)
}

func (s *AuthService) GetAdminUser(ctx context.Context, actorID, userID string) (AdminUserSummary, error) {
	if ok, err := s.Authorize(ctx, actorID, PermUserStatusManage); err != nil {
		return AdminUserSummary{}, err
	} else if !ok {
		return AdminUserSummary{}, ErrForbidden
	}
	if strings.TrimSpace(userID) == "" {
		return AdminUserSummary{}, ErrUserNotFound
	}
	store, err := s.adminUserDirectoryStore()
	if err != nil {
		return AdminUserSummary{}, err
	}
	return store.GetAdminUser(ctx, userID)
}
