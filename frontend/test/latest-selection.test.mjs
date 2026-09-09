import assert from 'node:assert/strict'
import test from 'node:test'

import {
  canActivateAudio,
  canCreateNarration,
  canSelectChapter,
  canStartChapterGeneration,
  canSynthesizeNarration,
  createLatestSelectionGuard,
  generationRunFromContentResponse,
  isChapterSelectionBlockingAction,
} from '../src/features/admin/latestSelection.mjs'

function trySelectChapter(state, chapterID) {
  if (!canSelectChapter({ action: state.action })) {
    return state
  }
  return { ...state, activeChapterID: chapterID }
}

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

test('chapter selection is blocked while synthesize or activate is in flight', () => {
  assert.equal(isChapterSelectionBlockingAction('synthesize'), true)
  assert.equal(isChapterSelectionBlockingAction('activate'), true)
  assert.equal(canSelectChapter({ action: 'synthesize' }), false)
  assert.equal(canSelectChapter({ action: 'activate' }), false)
  assert.equal(canSelectChapter({ action: '' }), true)
  assert.equal(canSelectChapter({ action: 'refresh-generation' }), true)
})

test('in-flight synthesize blocks chapter switch so mutation target cannot be abandoned', () => {
  let state = { activeChapterID: 'chapter-a', action: 'synthesize' }
  state = trySelectChapter(state, 'chapter-b')
  assert.equal(state.activeChapterID, 'chapter-a')
  assert.equal(state.action, 'synthesize')
})

test('in-flight activate blocks chapter switch so mutation target cannot be abandoned', () => {
  let state = { activeChapterID: 'chapter-a', action: 'activate' }
  state = trySelectChapter(state, 'chapter-b')
  assert.equal(state.activeChapterID, 'chapter-a')
  assert.equal(state.action, 'activate')
})

test('chapter selection is blocked while other durable chapter mutations are in flight', () => {
  assert.equal(canSelectChapter({ action: 'create-narration' }), false)
  assert.equal(canSelectChapter({ action: 'start-generation' }), false)
})
