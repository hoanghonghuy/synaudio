import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const reader = await readFile(new URL('../src/features/reader/StoryReader.vue', import.meta.url), 'utf8')
const mobileStyles = reader.match(/@media \(max-width: 760px\) \{([\s\S]*?)\n\}/)?.[1] ?? ''

test('mobile reader keeps the audio player sticky with safe-area spacing', () => {
  assert.match(mobileStyles, /\.audio-section\s*\{\s*position:\s*sticky;/)
  assert.match(mobileStyles, /top:\s*calc\(60px \+ env\(safe-area-inset-top\)\);/)
  assert.match(mobileStyles, /backdrop-filter:\s*blur\(/)
})

test('audio failures expose an explicit retry-audio action', () => {
  assert.match(reader, /class="audio-retry-btn"[\s\S]*>Thử lại audio<\/button>/)
  assert.match(reader, /@click="retryAudio"/)
})

test('mobile reader keeps the compact listening controls and remaining-time label', () => {
  assert.match(reader, /formatTimelineTime\(currentTime\)/)
  assert.match(reader, /formatTimelineTime\(remainingTime\)/)
  assert.match(reader, /class="audio-heading-copy"/)
  assert.match(reader, /class="player-secondary"[\s\S]*class="control-pill(?:\s|")/)
  assert.match(mobileStyles, /\.audio-heading-row h2\s*\{[^}]*font-size:/)
  assert.match(mobileStyles, /\.player-secondary\s*\{[^}]*display:\s*flex;/)
})

test('audio reload and playback paths handle browser media failures', () => {
  assert.match(reader, /function reloadAudioElement\(\)[\s\S]*try\s*\{[\s\S]*audioEl\.value\?\.load\(\)/)
  assert.match(reader, /function togglePlayback\(\)[\s\S]*await el\.play\(\)[\s\S]*NotAllowedError/)
})
