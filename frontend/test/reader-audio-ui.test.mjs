import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const reader = await readFile(new URL('../src/features/reader/StoryReader.vue', import.meta.url), 'utf8')

test('mobile reader keeps the audio player sticky with safe-area spacing', () => {
  const mobileStyles = reader.match(/@media \(max-width: 760px\) \{([\s\S]*?)\n\}/)?.[1] ?? ''

  assert.match(mobileStyles, /\.audio-section\s*\{\s*position:\s*sticky;/)
  assert.match(mobileStyles, /top:\s*calc\(60px \+ env\(safe-area-inset-top\)\);/)
  assert.match(mobileStyles, /backdrop-filter:\s*blur\(/)
})
