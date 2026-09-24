import assert from 'node:assert/strict'
import test from 'node:test'

import {
  createLatestChapterSelectionGuard,
  formatPlaybackTime,
  normalizeVolume,
  normalizePlaybackRate,
  toggleMuteState,
} from '../src/features/reader/readerSession.mjs'

test('latest chapter selection invalidates stale async completions', () => {
  const guard = createLatestChapterSelectionGuard()
  const firstMayCommit = guard.begin('chapter-1')
  assert.equal(firstMayCommit(), true)

  const secondMayCommit = guard.begin('chapter-2')
  assert.equal(firstMayCommit(), false)
  assert.equal(secondMayCommit(), true)
})

test('selection generation prevents ABA stale completion after returning to the same chapter', () => {
  const guard = createLatestChapterSelectionGuard()
  const firstA = guard.begin('chapter-a')
  const chapterB = guard.begin('chapter-b')
  const secondA = guard.begin('chapter-a')

  assert.equal(firstA(), false)
  assert.equal(chapterB(), false)
  assert.equal(secondA(), true)
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
  assert.equal(formatPlaybackTime(3661), '1:01:01')
})

test('volume values stay safe for the media element', () => {
  assert.equal(normalizeVolume(0.45), 0.45)
  assert.equal(normalizeVolume(-1), 0)
  assert.equal(normalizeVolume(2), 1)
  assert.equal(normalizeVolume('invalid'), 0.9)
})

test('mute state remembers the last audible volume and restores it', () => {
  const muted = toggleMuteState(false, 0.65, 0.9)
  assert.deepEqual(muted, { muted: true, volume: 0, lastAudibleVolume: 0.65 })

  const restored = toggleMuteState(true, 0, muted.lastAudibleVolume)
  assert.deepEqual(restored, { muted: false, volume: 0.65, lastAudibleVolume: 0.65 })
})
