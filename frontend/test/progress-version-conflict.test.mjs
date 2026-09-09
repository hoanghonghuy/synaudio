import test from 'node:test'
import assert from 'node:assert/strict'

import { ApiRequestError, isProgressVersionConflict } from '../src/api/http-error.ts'

test('detects progress version conflict with authoritative payload', () => {
  const error = new ApiRequestError(409, 'progress was updated elsewhere', 'PROGRESS_VERSION_CONFLICT', {
    UserID: 'user-1',
    ChapterID: 'chapter-1',
    PositionMs: 9000,
    CompletedAt: '',
    LastAudioAssetID: 'asset-1',
    LastPlaybackSessionID: 'session-2',
    Version: 2,
    RelistenStatus: 'NO_RELISTEN_NEEDED',
  })

  assert.equal(isProgressVersionConflict(error), true)
  assert.equal(error.progress.PositionMs, 9000)
})

test('does not treat other 409 responses as progress conflicts', () => {
  const error = new ApiRequestError(409, 'slug already taken', 'SLUG_TAKEN')
  assert.equal(isProgressVersionConflict(error), false)
})
