import test from 'node:test'
import assert from 'node:assert/strict'

import { ApiRequestError, isExplicitNotFound } from '../src/api/http-error.ts'

test('only explicit HTTP 404 is treated as an absent optional planning resource', () => {
  assert.equal(isExplicitNotFound(new ApiRequestError(404, 'not found')), true)
  assert.equal(isExplicitNotFound(new ApiRequestError(503, 'temporarily unavailable')), false)
  assert.equal(isExplicitNotFound(new ApiRequestError(401, 'unauthenticated')), false)
  assert.equal(isExplicitNotFound(new TypeError('network failed')), false)
})
