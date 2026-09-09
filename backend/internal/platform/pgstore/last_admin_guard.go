package pgstore

import (
	"context"

	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

const countMfaCapableActiveAdminsSQL = `
SELECT COUNT(*)
  FROM user_roles ur
  JOIN roles r ON r.id = ur.role_id
  JOIN users u ON u.id = ur.user_id
  JOIN user_mfa_methods m ON m.user_id = u.id
 WHERE r.code = 'ADMIN'
   AND u.status = 'ACTIVE'
   AND m.confirmed_at IS NOT NULL
   AND m.disabled_at IS NULL
`

const targetIsMfaCapableActiveAdminSQL = `
SELECT EXISTS (
    SELECT 1
      FROM users u
      JOIN user_roles ur ON ur.user_id = u.id
      JOIN roles r ON r.id = ur.role_id
      JOIN user_mfa_methods m ON m.user_id = u.id
     WHERE u.id = $1
       AND u.status = 'ACTIVE'
       AND r.code = 'ADMIN'
       AND m.confirmed_at IS NOT NULL
       AND m.disabled_at IS NULL
)
`

func countMfaCapableActiveAdmins(ctx context.Context, executor db.DBTX) (int, error) {
	var count int
	if err := executor.QueryRow(ctx, countMfaCapableActiveAdminsSQL).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func targetIsMfaCapableActiveAdmin(ctx context.Context, executor db.DBTX, userID string) (bool, error) {
	var ok bool
	if err := executor.QueryRow(ctx, targetIsMfaCapableActiveAdminSQL, toUUID(userID)).Scan(&ok); err != nil {
		return false, err
	}
	return ok, nil
}
