import assert from 'node:assert/strict'
import test from 'node:test'

import {
  canCreateNarration,
  canStartChapterGeneration,
  createLatestSelectionGuard,
  generationRunFromContentResponse,
} from '../src/features/admin/latestSelection.mjs'

test('only latest chapter selection may commit', () => {
  const guard = createLatestSelectionGuard()
  const mayCommitA = guard.begin('chapter-a')
  const mayCommitB = guard.begin('chapter-b')
  assert.equal(mayCommitA(), false)
  assert.equal(mayCommitB(), true)
})

test('durable chapter state restores generation run before any revision exists', () => {
  const run = { ID: 'run-1', RunType: 'CHAPTER_GENERATION', ChapterID: 'chapter-a', Status: 'RUNNING' }
  assert.equal(generationRunFromContentResponse({ revisions: [], generation_run: run }), run)
  assert.equal(generationRunFromContentResponse({ revisions: [], generation_run: null }), null)
})

test('generation start is blocked while authoritative chapter state is loading', () => {
  assert.equal(canStartChapterGeneration({
    hasPlanRevision: true,
    selectionLoading: true,
    actionInProgress: false,
    hasGenerationRun: false,
    hasGenerationRunProvenance: false,
  }), false)
})

test('generation start is blocked when durable restored run already exists', () => {
  assert.equal(canStartChapterGeneration({
    hasPlanRevision: true,
    selectionLoading: false,
    actionInProgress: false,
    hasGenerationRun: true,
    hasGenerationRunProvenance: false,
  }), false)
})

test('generation start is blocked when durable provenance says a run already exists', () => {
  assert.equal(canStartChapterGeneration({
    hasPlanRevision: true,
    selectionLoading: false,
    actionInProgress: false,
    hasGenerationRun: false,
    hasGenerationRunProvenance: true,
  }), false)
})

test('generation start is allowed only for an idle planned chapter with no existing run identity', () => {
  assert.equal(canStartChapterGeneration({
    hasPlanRevision: true,
    selectionLoading: false,
    actionInProgress: false,
    hasGenerationRun: false,
    hasGenerationRunProvenance: false,
  }), true)
})

test('narration creation requires authoritative approved revision and voice', () => {
  const approvedRevision = {
    ID: 'rev-1',
    ContentText: 'Approved prose',
    Status: 'APPROVED',
  }

  assert.equal(canCreateNarration({
    approvedRevision,
    selectionLoading: false,
    actionInProgress: false,
    voiceID: 'voice-1',
  }), true)

  assert.equal(canCreateNarration({
    approvedRevision: { ...approvedRevision, ContentText: '   ' },
    selectionLoading: false,
    actionInProgress: false,
    voiceID: 'voice-1',
  }), false)

  assert.equal(canCreateNarration({
    approvedRevision: null,
    selectionLoading: false,
    actionInProgress: false,
    voiceID: 'voice-1',
  }), false)

  assert.equal(canCreateNarration({
    approvedRevision,
    selectionLoading: true,
    actionInProgress: false,
    voiceID: 'voice-1',
  }), false)
})
