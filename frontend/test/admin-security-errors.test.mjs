import test from 'node:test'
import assert from 'node:assert/strict'

import {
  ApiRequestError,
  adminSecurityErrorMessage,
  isLastActiveAdmin,
  isMfaRequired,
  isPermissionDenied,
  isPrivilegedAssuranceRequired,
  isRecentAuthRequired,
  privilegedAssuranceCode,
} from '../src/api/http-error.ts'

test('privileged assurance helpers identify backend security codes', () => {
  const mfa = new ApiRequestError(403, 'mfa verification required', 'MFA_REQUIRED')
  const recent = new ApiRequestError(403, 'recent authentication required', 'RECENT_AUTH_REQUIRED')
  const forbidden = new ApiRequestError(403, 'permission required', 'FORBIDDEN')
  const lastAdmin = new ApiRequestError(409, 'cannot remove last active admin', 'LAST_ACTIVE_ADMIN')

  assert.equal(isMfaRequired(mfa), true)
  assert.equal(isRecentAuthRequired(mfa), false)
  assert.equal(isPrivilegedAssuranceRequired(mfa), true)
  assert.equal(privilegedAssuranceCode(mfa), 'MFA_REQUIRED')

  assert.equal(isRecentAuthRequired(recent), true)
  assert.equal(isPrivilegedAssuranceRequired(recent), true)
  assert.equal(privilegedAssuranceCode(recent), 'RECENT_AUTH_REQUIRED')

  assert.equal(isPermissionDenied(forbidden), true)
  assert.equal(isPrivilegedAssuranceRequired(forbidden), false)
  assert.equal(isLastActiveAdmin(lastAdmin), true)
})

test('admin security messages are user-facing and do not echo credential material', () => {
  const message = adminSecurityErrorMessage(
    new ApiRequestError(403, 'mfa verification required', 'MFA_REQUIRED'),
  )

  assert.match(message, /MFA|TOTP|khôi phục/i)
  assert.doesNotMatch(message, /access_token|refresh_token|recovery_codes|secret/i)
})

test('admin security messages cover denial and stale-state cases', () => {
  assert.match(
    adminSecurityErrorMessage(new ApiRequestError(409, 'cannot remove last active admin', 'LAST_ACTIVE_ADMIN')),
    /Admin/i,
  )
  assert.match(
    adminSecurityErrorMessage(new ApiRequestError(403, 'permission required', 'FORBIDDEN')),
    /quyền/i,
  )
  assert.match(
    adminSecurityErrorMessage(new ApiRequestError(404, 'user not found', 'USER_NOT_FOUND')),
    /Không tìm thấy/i,
  )
})
