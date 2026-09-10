import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildChapterProductionStages,
  getPrimaryProductionStage,
  stageStateLabel,
} from '../src/features/admin/chapterProductionStages.mjs'

test('selection loading is represented across every production stage', () => {
  const stages = buildChapterProductionStages({ selectionLoading: true })
  assert.equal(stages.length, 6)
  assert.ok(stages.every((stage) => stage.state === 'loading'))
})

test('missing plan is an explicit blocker instead of a disabled-action mystery', () => {
  const [plan, generation] = buildChapterProductionStages({
    hasPlanRevision: false,
    generationJobStatus: '',
  })
  assert.equal(plan.state, 'blocked')
  assert.match(plan.blocker, /Story Planning/)
  assert.equal(generation.state, 'blocked')
})

test('failed generation exposes actionable recovery semantics', () => {
  const stages = buildChapterProductionStages({
    hasPlanRevision: true,
    generationJobStatus: 'FAILED · PROVIDER_UNAVAILABLE',
  })
  const generation = stages.find((stage) => stage.id === 'generation')
  assert.equal(generation.state, 'failed')
  assert.match(generation.blocker, /Retry/)
  assert.equal(getPrimaryProductionStage(stages).id, 'generation')
})

test('stale audio projection remains blocked and is never promoted client-side', () => {
  const stages = buildChapterProductionStages({
    hasPlanRevision: true,
    generationJobStatus: 'SUCCEEDED',
    hasApprovedRevision: true,
    narrationStatus: 'Revision #3',
    audioStatus: 'BLOCKED — READY asset thuộc narration cũ, không khớp narration mới nhất',
    publishStatus: 'BLOCKED — thiếu: Active durable audio',
  })
  const audio = stages.find((stage) => stage.id === 'audio')
  assert.equal(audio.state, 'blocked')
  assert.match(audio.blocker, /narration cũ/)
})

test('active audio and published chapter produce terminal states', () => {
  const stages = buildChapterProductionStages({
    hasPlanRevision: true,
    generationJobStatus: 'SUCCEEDED',
    hasApprovedRevision: true,
    narrationStatus: 'Revision #4',
    audioStatus: 'ACTIVE v4 · asset-4',
    publishStatus: 'PUBLISHED · chapter-1',
    chapterPublished: true,
  })
  assert.equal(stages.find((stage) => stage.id === 'audio').state, 'active')
  assert.equal(stages.find((stage) => stage.id === 'publish').state, 'published')
  assert.equal(getPrimaryProductionStage(stages).id, 'publish')
  assert.equal(stageStateLabel('published'), 'Đã publish')
})
