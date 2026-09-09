import test from 'node:test'
import assert from 'node:assert/strict'

import { ApiRequestError } from '../src/api/http-error.ts'
import { withPrivilegedRetry } from '../src/api/privileged-request.ts'

test('withPrivilegedRetry returns successful operations without re-auth', async () => {
  let reAuthCalls = 0
  const result = await withPrivilegedRetry(
    async () => ({ users: [{ id: 'u1', email: 'a@example.com' }] }),
    async () => {
      reAuthCalls += 1
      return true
    },
  )

  assert.deepEqual(result, { users: [{ id: 'u1', email: 'a@example.com' }] })
  assert.equal(reAuthCalls, 0)
})

test('withPrivilegedRetry recovers from RECENT_AUTH_REQUIRED with one explicit retry', async () => {
  let attempts = 0
  const reasons = []

  const result = await withPrivilegedRetry(
    async () => {
      attempts += 1
      if (attempts === 1) {
        throw new ApiRequestError(403, 'recent authentication required', 'RECENT_AUTH_REQUIRED')
      }
      return { status: 'granted' }
    },
    async (reason) => {
      reasons.push(reason)
      return true
    },
  )

  assert.deepEqual(result, { status: 'granted' })
  assert.equal(attempts, 2)
  assert.deepEqual(reasons, ['RECENT_AUTH_REQUIRED'])
})

test('withPrivilegedRetry recovers from MFA_REQUIRED before retrying mutation', async () => {
  let attempts = 0

  const result = await withPrivilegedRetry(
    async () => {
      attempts += 1
      if (attempts === 1) {
        throw new ApiRequestError(403, 'mfa verification required', 'MFA_REQUIRED')
      }
      return { status: 'ok' }
    },
    async (reason) => {
      assert.equal(reason, 'MFA_REQUIRED')
      return true
    },
  )

  assert.deepEqual(result, { status: 'ok' })
  assert.equal(attempts, 2)
})

test('withPrivilegedRetry propagates backend denial when re-auth is declined', async () => {
  await assert.rejects(
    () =>
      withPrivilegedRetry(
        async () => {
          throw new ApiRequestError(403, 'recent authentication required', 'RECENT_AUTH_REQUIRED')
        },
        async () => false,
      ),
    (error) => error instanceof ApiRequestError && error.code === 'RECENT_AUTH_REQUIRED',
  )
})

test('withPrivilegedRetry propagates non-assurance backend denials without re-auth', async () => {
  let reAuthCalls = 0

  await assert.rejects(
    () =>
      withPrivilegedRetry(
        async () => {
          throw new ApiRequestError(409, 'cannot remove last active admin', 'LAST_ACTIVE_ADMIN')
        },
        async () => {
          reAuthCalls += 1
          return true
        },
      ),
    (error) => error instanceof ApiRequestError && error.code === 'LAST_ACTIVE_ADMIN',
  )

  assert.equal(reAuthCalls, 0)
})
