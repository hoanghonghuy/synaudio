package pgstore

import (
	"context"
	"errors"
	"strings"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func (s *IdentityStore) ListAdminUsers(ctx context.Context, query, status string, limit int) ([]identity.AdminUserSummary, error) {
	rows, err := s.q.DBTX().Query(ctx, `
SELECT u.id::text,
       u.email,
       COALESCE(u.display_name, ''),
       u.status,
       (u.email_verified_at IS NOT NULL),
       COALESCE((
           SELECT string_agg(r.code, ',' ORDER BY r.code)
             FROM user_roles ur
             JOIN roles r ON r.id = ur.role_id
            WHERE ur.user_id = u.id
       ), '') AS roles
  FROM users u
 WHERE ($1 = '' OR u.email ILIKE '%' || $1 || '%' OR COALESCE(u.display_name, '') ILIKE '%' || $1 || '%')
   AND ($2 = '' OR u.status = $2)
 ORDER BY u.created_at DESC, u.id DESC
 LIMIT $3
`, query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]identity.AdminUserSummary, 0)
	for rows.Next() {
		var item identity.AdminUserSummary
		var rolesCSV string
		if err := rows.Scan(&item.ID, &item.Email, &item.DisplayName, &item.Status, &item.EmailVerified, &rolesCSV); err != nil {
			return nil, err
		}
		item.Roles = splitRoleCodes(rolesCSV)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *IdentityStore) GetAdminUser(ctx context.Context, userID string) (identity.AdminUserSummary, error) {
	var item identity.AdminUserSummary
	var rolesCSV string
	err := s.q.DBTX().QueryRow(ctx, `
SELECT u.id::text,
       u.email,
       COALESCE(u.display_name, ''),
       u.status,
       (u.email_verified_at IS NOT NULL),
       COALESCE((
           SELECT string_agg(r.code, ',' ORDER BY r.code)
             FROM user_roles ur
             JOIN roles r ON r.id = ur.role_id
            WHERE ur.user_id = u.id
       ), '') AS roles
  FROM users u
 WHERE u.id = $1
`, toUUID(userID)).Scan(&item.ID, &item.Email, &item.DisplayName, &item.Status, &item.EmailVerified, &rolesCSV)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return identity.AdminUserSummary{}, identity.ErrUserNotFound
		}
		return identity.AdminUserSummary{}, err
	}
	item.Roles = splitRoleCodes(rolesCSV)
	return item, nil
}

func splitRoleCodes(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if code := strings.TrimSpace(part); code != "" {
			result = append(result, code)
		}
	}
	return result
}

var _ identity.AdminUserDirectoryStore = (*IdentityStore)(nil)
var _ = errors.New
