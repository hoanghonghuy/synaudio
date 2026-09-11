import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildChapterProductionStages,
  getPrimaryProductionStage,
  stageStateLabel,
} from '../src/features/admin/chapterProductionStages.mjs'

test('selection loading keeps the legacy six-stage workspace until canon authority is wired', () => {
  const stages = buildChapterProductionStages({ selectionLoading: true })
  assert.equal(stages.length, 6)
  assert.ok(stages.every((stage) => stage.state === 'loading'))
})

test('selection loading includes canon when authoritative canon status is supplied', () => {
  const stages = buildChapterProductionStages({ selectionLoading: true, canonStatus: 'Đang tải Canon/Memory authority…' })
  assert.equal(stages.length, 7)
  assert.equal(stages[3].id, 'canon')
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

test('canon memory becomes the next attention after review until exact approved revision is current', () => {
  const readyToCommit = buildChapterProductionStages({
    hasPlanRevision: true,
    generationJobStatus: 'SUCCEEDED',
    hasApprovedRevision: true,
    canonStatus: 'READY TO COMMIT — approved revision rev-2 mới hơn Canon source rev-1',
    narrationStatus: 'READY — có thể tạo narration từ approved revision hiện hành',
  })
  const canon = readyToCommit.find((stage) => stage.id === 'canon')
  assert.equal(canon.state, 'waiting')
  assert.match(canon.blocker, /exact approved revision/)
  assert.equal(getPrimaryProductionStage(readyToCommit).id, 'canon')

  const current = buildChapterProductionStages({
    hasPlanRevision: true,
    generationJobStatus: 'SUCCEEDED',
    hasApprovedRevision: true,
    canonStatus: 'CURRENT — Canon/Memory đã commit từ approved revision rev-2',
    narrationStatus: 'WAITING — chưa có narration authoritative',
  })
  assert.equal(current.find((stage) => stage.id === 'canon').state, 'ready')
  assert.equal(getPrimaryProductionStage(current).id, 'narration')
})

test('missing canon authority fails closed instead of allowing client-side readiness inference', () => {
  const stages = buildChapterProductionStages({
    hasPlanRevision: true,
    generationJobStatus: 'SUCCEEDED',
    hasApprovedRevision: true,
    canonStatus: 'BLOCKED — thiếu ACTIVE OFFICIAL CanonBranch authority',
    narrationStatus: 'READY — có thể tạo narration từ approved revision hiện hành',
  })
  const canon = stages.find((stage) => stage.id === 'canon')
  assert.equal(canon.state, 'blocked')
  assert.match(canon.blocker, /ACTIVE OFFICIAL/)
  assert.equal(getPrimaryProductionStage(stages).id, 'canon')
})

test('stale audio projection remains blocked and is never promoted client-side', () => {
  const stages = buildChapterProductionStages({
    hasPlanRevision: true,
    generationJobStatus: 'SUCCEEDED',
    hasApprovedRevision: true,
    canonStatus: 'CURRENT — Canon/Memory đã commit từ approved revision rev-3',
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
    canonStatus: 'CURRENT — Canon/Memory đã commit từ approved revision rev-4',
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
