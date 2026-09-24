import assert from 'node:assert/strict'
import test from 'node:test'

import { resolvePlayableAudioURL } from '../src/api/client.ts'

test('playable audio URLs stay same-origin when storage returns an internal MinIO host', () => {
  assert.equal(
    resolvePlayableAudioURL('http://minio:9000/synaudio/audio/chapter.mp3?X-Amz-Signature=test'),
    '/synaudio/audio/chapter.mp3?X-Amz-Signature=test',
  )
  assert.equal(
    resolvePlayableAudioURL('http://localhost:9000/synaudio/audio/chapter.wav'),
    '/synaudio/audio/chapter.wav',
  )
})

test('playable audio URL resolver does not rewrite public or malformed URLs', () => {
  assert.equal(resolvePlayableAudioURL('https://cdn.example.test/audio.mp3'), 'https://cdn.example.test/audio.mp3')
  assert.equal(resolvePlayableAudioURL('/synaudio/audio/chapter.mp3'), '/synaudio/audio/chapter.mp3')
  assert.equal(resolvePlayableAudioURL('not a url'), 'not a url')
})
