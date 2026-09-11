import test from 'node:test'
import assert from 'node:assert/strict'
import { createAudioPreviewState, previewAssetForChapter } from '../src/features/admin/audioPreviewState.mjs'

test('preview state rejects stale completion after chapter or asset selection changes', () => {
  const preview = createAudioPreviewState()
  const first = preview.begin('chapter-1', 'asset-1')
  const second = preview.begin('chapter-2', 'asset-2')

  assert.equal(first.mayCommit(), false)
  first.succeed('https://stale.example/audio.mp3')
  assert.deepEqual(preview.snapshot(), {
    status: 'loading', chapterID: 'chapter-2', assetID: 'asset-2', url: '', error: '',
  })

  second.succeed('https://current.example/audio.mp3')
  assert.deepEqual(preview.snapshot(), {
    status: 'ready', chapterID: 'chapter-2', assetID: 'asset-2', url: 'https://current.example/audio.mp3', error: '',
  })
})

test('reset invalidates an in-flight preview request before a new selection commits', () => {
  const preview = createAudioPreviewState()
  const pending = preview.begin('chapter-1', 'asset-1')
  preview.reset()

  assert.equal(pending.mayCommit(), false)
  pending.succeed('https://stale.example/audio.mp3')
  assert.deepEqual(preview.snapshot(), {
    status: 'idle', chapterID: '', assetID: '', url: '', error: '',
  })
})

test('preview retry replaces error without retaining a stale URL', () => {
  const preview = createAudioPreviewState()
  preview.begin('chapter-1', 'asset-1').fail('provider unavailable')
  assert.equal(preview.snapshot().status, 'error')
  assert.equal(preview.snapshot().url, '')

  preview.begin('chapter-1', 'asset-1').succeed('https://ready.example/audio.mp3')
  assert.equal(preview.snapshot().status, 'ready')
  assert.equal(preview.snapshot().error, '')
})

test('preview fails closed when the authority response has an empty URL', () => {
  const preview = createAudioPreviewState()
  const snapshot = preview.begin('chapter-1', 'asset-1').succeed('   ')

  assert.equal(snapshot.status, 'error')
  assert.equal(snapshot.url, '')
  assert.match(snapshot.error, /URL không hợp lệ/)
})

test('preview asset selection never promotes foreign, non-ready, or stale asset authority', () => {
  const ready = { ID: 'ready-1', ChapterID: 'chapter-1', Status: 'READY' }
  const active = { ID: 'active-1', ChapterID: 'chapter-1', Status: 'READY' }

  assert.equal(previewAssetForChapter({ activeChapterID: 'chapter-1', activeAudio: active, latestReadyAudio: ready }), ready)
  assert.equal(previewAssetForChapter({ activeChapterID: 'chapter-1', activeAudio: active, latestReadyAudio: null }), active)
  assert.equal(previewAssetForChapter({ activeChapterID: 'chapter-1', activeAudio: { ...active, Status: 'PROCESSING' }, latestReadyAudio: null }), null)
  assert.equal(previewAssetForChapter({ activeChapterID: 'chapter-1', activeAudio: null, latestReadyAudio: { ...ready, ChapterID: 'chapter-2' } }), null)
})
