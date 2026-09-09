import assert from 'node:assert/strict'
import test from 'node:test'

import {
  canActivateAudio,
  canStartChapterGeneration,
  canSynthesizeNarration,
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

test('synthesize is blocked while authoritative chapter state is loading', () => {
  assert.equal(canSynthesizeNarration({
    hasApprovedContent: true,
    hasNarration: true,
    narrationBelongsToChapter: true,
    selectionLoading: true,
    actionInProgress: false,
  }), false)
})

test('synthesize is blocked without approved content or narration', () => {
  assert.equal(canSynthesizeNarration({
    hasApprovedContent: false,
    hasNarration: true,
    narrationBelongsToChapter: true,
    selectionLoading: false,
    actionInProgress: false,
  }), false)
  assert.equal(canSynthesizeNarration({
    hasApprovedContent: true,
    hasNarration: false,
    narrationBelongsToChapter: false,
    selectionLoading: false,
    actionInProgress: false,
  }), false)
})

test('synthesize is blocked when narration does not belong to selected chapter', () => {
  assert.equal(canSynthesizeNarration({
    hasApprovedContent: true,
    hasNarration: true,
    narrationBelongsToChapter: false,
    selectionLoading: false,
    actionInProgress: false,
  }), false)
})

test('synthesize is allowed only with approved narration bound to the selected chapter', () => {
  assert.equal(canSynthesizeNarration({
    hasApprovedContent: true,
    hasNarration: true,
    narrationBelongsToChapter: true,
    selectionLoading: false,
    actionInProgress: false,
  }), true)
})

test('activate is blocked while authoritative chapter state is loading', () => {
  assert.equal(canActivateAudio({
    hasReadyAsset: true,
    readyAssetBelongsToChapter: true,
    readyAssetIsInactive: true,
    readyAssetMatchesLatestNarration: true,
    selectionLoading: true,
    actionInProgress: false,
  }), false)
})

test('activate is blocked without an inactive ready asset for the selected chapter', () => {
  assert.equal(canActivateAudio({
    hasReadyAsset: false,
    readyAssetBelongsToChapter: false,
    readyAssetIsInactive: false,
    readyAssetMatchesLatestNarration: false,
    selectionLoading: false,
    actionInProgress: false,
  }), false)
  assert.equal(canActivateAudio({
    hasReadyAsset: true,
    readyAssetBelongsToChapter: true,
    readyAssetIsInactive: false,
    readyAssetMatchesLatestNarration: true,
    selectionLoading: false,
    actionInProgress: false,
  }), false)
})

test('activate is blocked when ready asset is stale relative to latest narration', () => {
  assert.equal(canActivateAudio({
    hasReadyAsset: true,
    readyAssetBelongsToChapter: true,
    readyAssetIsInactive: true,
    readyAssetMatchesLatestNarration: false,
    selectionLoading: false,
    actionInProgress: false,
  }), false)
})

test('activate is allowed only for an inactive ready asset owned by the selected chapter and latest narration', () => {
  assert.equal(canActivateAudio({
    hasReadyAsset: true,
    readyAssetBelongsToChapter: true,
    readyAssetIsInactive: true,
    readyAssetMatchesLatestNarration: true,
    selectionLoading: false,
    actionInProgress: false,
  }), true)
})
