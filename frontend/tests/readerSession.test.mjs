import assert from 'node:assert/strict'
import test from 'node:test'

import {
  createLatestChapterSelectionGuard,
  formatPlaybackTime,
  normalizePlaybackRate,
} from '../src/features/reader/readerSession.mjs'

test('latest chapter selection invalidates stale async completions', () => {
  const guard = createLatestChapterSelectionGuard()
  const firstMayCommit = guard.begin('chapter-1')
  assert.equal(firstMayCommit(), true)

  const secondMayCommit = guard.begin('chapter-2')
  assert.equal(firstMayCommit(), false)
  assert.equal(secondMayCommit(), true)
})

test('playback rate accepts common audiobook rates and fails safe to 1x', () => {
  assert.equal(normalizePlaybackRate(0.75), 0.75)
  assert.equal(normalizePlaybackRate('1.5'), 1.5)
  assert.equal(normalizePlaybackRate(3), 1)
  assert.equal(normalizePlaybackRate('garbage'), 1)
})

test('playback time is stable for empty and long-form durations', () => {
  assert.equal(formatPlaybackTime(Number.NaN), '0:00')
  assert.equal(formatPlaybackTime(-1), '0:00')
  assert.equal(formatPlaybackTime(65.8), '1:05')
  assert.equal(formatPlaybackTime(3661), '61:01')
})
