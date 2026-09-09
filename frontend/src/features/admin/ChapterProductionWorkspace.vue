<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  activateAudioAsset,
  createNarrationRevision,
  getActiveAudioAsset,
  getGenerationRun,
  getLatestNarrationRevision,
  getLatestReadyAudioAsset,
  getStoryWorkflowSettings,
  listAdminChapters,
  listChapterReviews,
  listContentRevisions,
  startChapterGeneration,
  synthesizeNarration,
  type GenerationRun,
} from '../../api/client'
import type { AudioAsset, Chapter, ChapterReview, ContentRevision, NarrationRevision } from '../../api/types'
import {
  canActivateAudio,
  canCreateNarration,
  canSelectChapter,
  canStartChapterGeneration,
  canSynthesizeNarration,
  createLatestSelectionGuard,
  generationRunFromContentResponse,
} from './latestSelection.mjs'

const route = useRoute()
const storyID = computed(() => route.params.storyID as string)
const chapters = ref<Chapter[]>([])
const activeChapter = ref<Chapter | null>(null)
const revisions = ref<ContentRevision[]>([])
const reviews = ref<ChapterReview[]>([])
const generationRun = ref<GenerationRun | null>(null)
const latestNarration = ref<NarrationRevision | null>(null)
const activeAudio = ref<AudioAsset | null>(null)
const latestReadyAudio = ref<AudioAsset | null>(null)
const preferredVoiceID = ref('')
const loading = ref(false)
const selectionLoading = ref(false)
const action = ref('')
const error = ref('')
const chapterSelection = createLatestSelectionGuard()

const latestRevision = computed(() => revisions.value[revisions.value.length - 1] ?? null)
const approvedRevision = computed(() => [...revisions.value].reverse().find((revision) => revision.Status === 'APPROVED') ?? null)
const narrationBelongsToChapter = computed(() => Boolean(
  latestNarration.value
  && activeChapter.value
  && latestNarration.value.ChapterID === activeChapter.value.ID,
))
const readyAssetBelongsToChapter = computed(() => Boolean(
  latestReadyAudio.value
  && activeChapter.value
  && latestReadyAudio.value.ChapterID === activeChapter.value.ID,
))
const readyAssetMatchesLatestNarration = computed(() => Boolean(
  latestReadyAudio.value
  && latestNarration.value
  && latestReadyAudio.value.SourceNarrationRevisionID === latestNarration.value.ID,
))

const mayStartGeneration = computed(() => canStartChapterGeneration({
  hasPlanRevision: Boolean(activeChapter.value?.CurrentPlanRevisionID),
  selectionLoading: selectionLoading.value,
  actionInProgress: Boolean(action.value),
  hasGenerationRun: Boolean(generationRun.value),
  hasGenerationRunProvenance: Boolean(latestRevision.value?.GenerationRunID),
}))
const mayCreateNarration = computed(() => canCreateNarration({
  approvedRevision: approvedRevision.value,
  selectionLoading: selectionLoading.value,
  actionInProgress: Boolean(action.value),
  voiceID: preferredVoiceID.value,
}) && !latestNarration.value)

const maySynthesize = computed(() => canSynthesizeNarration({
  hasApprovedContent: Boolean(approvedRevision.value),
  hasNarration: Boolean(latestNarration.value),
  narrationBelongsToChapter: narrationBelongsToChapter.value,
  selectionLoading: selectionLoading.value,
  actionInProgress: Boolean(action.value),
}))

const maySelectChapter = computed(() => canSelectChapter({ action: action.value }))

const mayActivate = computed(() => canActivateAudio({
  hasReadyAsset: Boolean(latestReadyAudio.value),
  readyAssetBelongsToChapter: readyAssetBelongsToChapter.value,
  readyAssetIsInactive: Boolean(latestReadyAudio.value && !latestReadyAudio.value.IsActive),
  readyAssetMatchesLatestNarration: readyAssetMatchesLatestNarration.value,
  selectionLoading: selectionLoading.value,
  actionInProgress: Boolean(action.value),
}))

const narrationStatus = computed(() => {
  if (selectionLoading.value) return 'Đang tải…'
  if (!approvedRevision.value) return 'WAITING — cần approved content trước khi có narration authoritative'
  if (!latestNarration.value) return 'READY — có thể tạo narration từ approved revision hiện hành'
  if (!narrationBelongsToChapter.value) return 'BLOCKED — narration projection không thuộc chương đang chọn'
  return `Revision #${latestNarration.value.RevisionNo} · ${latestNarration.value.ID} · source ${latestNarration.value.SourceContentRevisionID}`
})

const audioStatus = computed(() => {
  if (selectionLoading.value) return 'Đang tải…'
  if (activeAudio.value) {
    return `ACTIVE v${activeAudio.value.VersionNo} · ${activeAudio.value.ID} · checksum ${activeAudio.value.Checksum || '—'} · ${activeAudio.value.DurationMs}ms`
  }
  if (latestReadyAudio.value) {
    if (!readyAssetBelongsToChapter.value) return 'BLOCKED — READY asset projection không thuộc chương đang chọn'
    if (!readyAssetMatchesLatestNarration.value) return 'BLOCKED — READY asset thuộc narration cũ, không khớp narration mới nhất'
    return `READY v${latestReadyAudio.value.VersionNo} · ${latestReadyAudio.value.ID} · checksum ${latestReadyAudio.value.Checksum || '—'} · chưa activate`
  }
  if (!latestNarration.value) return 'WAITING — cần narration trước khi synthesize audio'
  return 'WAITING — chưa có READY audio durable'
})

const publishStatus = computed(() => 'BLOCKED — publish wiring chưa có trong slice này; cần active audio + publish authority')

async function loadAudioProjections(chapterID: string) {
  const [narration, active] = await Promise.all([
    getLatestNarrationRevision(chapterID),
    getActiveAudioAsset(chapterID),
  ])
  const ready = narration ? await getLatestReadyAudioAsset(chapterID, narration.ID) : null
  return { narration, active, ready }
}

async function selectChapter(chapter: Chapter) {
  if (!maySelectChapter.value || activeChapter.value?.ID === chapter.ID) return
  const mayCommit = chapterSelection.begin(chapter.ID)
  activeChapter.value = chapter
  revisions.value = []
  reviews.value = []
  generationRun.value = null
  latestNarration.value = null
  activeAudio.value = null
  latestReadyAudio.value = null
  selectionLoading.value = true
  error.value = ''
  try {
    const [revisionResponse, reviewResponse, audioState] = await Promise.all([
      listContentRevisions(chapter.ID),
      listChapterReviews(chapter.ID),
      loadAudioProjections(chapter.ID),
    ])
    if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return

    const authoritativeContentState = revisionResponse as typeof revisionResponse & {
      generation_run?: GenerationRun | null
    }
    revisions.value = revisionResponse.revisions
    reviews.value = reviewResponse.reviews
    generationRun.value = generationRunFromContentResponse(authoritativeContentState)
    latestNarration.value = audioState.narration
    activeAudio.value = audioState.active
    latestReadyAudio.value = audioState.ready
  } catch (e) {
    if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
    revisions.value = []
    reviews.value = []
    generationRun.value = null
    latestNarration.value = null
    activeAudio.value = null
    latestReadyAudio.value = null
    error.value = e instanceof Error ? e.message : 'Không thể tải trạng thái production của chương.'
  } finally {
    if (mayCommit() && activeChapter.value?.ID === chapter.ID) selectionLoading.value = false
  }
}

async function startGeneration() {
  const chapter = activeChapter.value
  if (!chapter || !mayStartGeneration.value) return
  action.value = 'start-generation'
  error.value = ''
  try {
    const run = await startChapterGeneration(storyID.value, chapter.ID)
    if (activeChapter.value?.ID === chapter.ID) generationRun.value = run
  } catch (e) {
    if (activeChapter.value?.ID === chapter.ID) error.value = e instanceof Error ? e.message : 'Không thể bắt đầu Chapter Generation.'
  } finally {
    action.value = ''
  }
}

async function refreshGeneration() {
  const chapter = activeChapter.value
  const run = generationRun.value
  if (!chapter || !run || action.value || selectionLoading.value) return
  action.value = 'refresh-generation'
  error.value = ''
  try {
    const refreshed = await getGenerationRun(run.ID)
    if (activeChapter.value?.ID === chapter.ID) generationRun.value = refreshed
  } catch (e) {
    if (activeChapter.value?.ID === chapter.ID) error.value = e instanceof Error ? e.message : 'Không thể refresh Generation Run.'
  } finally {
    action.value = ''
  }
}

async function startNarration() {
  const chapter = activeChapter.value
  const approved = approvedRevision.value
  if (!chapter || !approved || !mayCreateNarration.value) return

  const requestChapterID = chapter.ID
  const requestRevisionID = approved.ID
  const requestScript = approved.ContentText
  action.value = 'create-narration'
  error.value = ''
  try {
    const created = await createNarrationRevision(requestChapterID, {
      source_content_revision_id: requestRevisionID,
      voice_id: preferredVoiceID.value,
      script: requestScript,
    })
    if (activeChapter.value?.ID !== requestChapterID || approvedRevision.value?.ID !== requestRevisionID) return
    latestNarration.value = created
  } catch (e) {
    if (activeChapter.value?.ID === requestChapterID && approvedRevision.value?.ID === requestRevisionID) {
      error.value = e instanceof Error ? e.message : 'Không thể tạo narration revision.'
    }
  } finally {
    if (activeChapter.value?.ID === requestChapterID) action.value = ''
  }
}

async function runSynthesize() {
  const chapter = activeChapter.value
  const narration = latestNarration.value
  if (!chapter || !narration || !maySynthesize.value) return
  action.value = 'synthesize'
  error.value = ''
  try {
    const asset = await synthesizeNarration(chapter.ID, narration.ID)
    if (activeChapter.value?.ID !== chapter.ID) return
    latestReadyAudio.value = asset
    if (asset.IsActive) activeAudio.value = asset
  } catch (e) {
    if (activeChapter.value?.ID === chapter.ID) {
      error.value = e instanceof Error ? e.message : 'Không thể synthesize narration thành READY audio.'
    }
  } finally {
    action.value = ''
  }
}

async function runActivate() {
  const chapter = activeChapter.value
  const ready = latestReadyAudio.value
  if (!chapter || !ready || !mayActivate.value) return
  action.value = 'activate'
  error.value = ''
  try {
    const asset = await activateAudioAsset(chapter.ID, ready.ID)
    if (activeChapter.value?.ID !== chapter.ID) return
    activeAudio.value = asset
    latestReadyAudio.value = asset.IsActive ? null : asset
    const narration = latestNarration.value
    const refreshed = narration ? await getLatestReadyAudioAsset(chapter.ID, narration.ID) : null
    if (activeChapter.value?.ID === chapter.ID) latestReadyAudio.value = refreshed
  } catch (e) {
    if (activeChapter.value?.ID === chapter.ID) {
      error.value = e instanceof Error ? e.message : 'Không thể activate READY audio asset.'
    }
  } finally {
    action.value = ''
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [chapterResponse, workflowSettings] = await Promise.all([
      listAdminChapters(storyID.value),
      getStoryWorkflowSettings(storyID.value),
    ])
    chapters.value = chapterResponse.chapters
    preferredVoiceID.value = workflowSettings.preferred_voice_id
    if (chapters.value.length > 0) await selectChapter(chapters.value[0])
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải Chapter Production workspace.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="production-page">
    <header class="production-header">
      <div>
        <p class="eyebrow">Studio / Chapter Production</p>
        <h1>Chapter Production</h1>
        <p>Backend-authoritative workspace cho P0 Chapter → Audio → Listen. Stage chưa có authority sẽ hiển thị blocker, không suy diễn client-side.</p>
      </div>
      <RouterLink :to="`/admin/stories/${storyID}/planning`">← Story Planning Studio</RouterLink>
    </header>

    <p v-if="loading">Đang tải production workspace...</p>
    <p v-if="error" class="error" role="alert">{{ error }}</p>

    <div v-if="!loading" class="workspace-grid">
      <aside class="chapter-panel">
        <h2>Chapters</h2>
        <button v-for="chapter in chapters" :key="chapter.ID" type="button" :class="{ active: activeChapter?.ID === chapter.ID }" :disabled="!maySelectChapter" :aria-disabled="!maySelectChapter" @click="selectChapter(chapter)">
          <span>Chương {{ chapter.ChapterNumber }}</span>
          <strong>{{ chapter.Title }}</strong>
          <small>{{ chapter.Status }}</small>
        </button>
      </aside>

      <main v-if="activeChapter" class="pipeline-panel" :aria-busy="selectionLoading">
        <h2>{{ activeChapter.Title }}</h2>
        <p v-if="selectionLoading" role="status">Đang tải trạng thái authoritative của chương…</p>
        <dl>
          <div><dt>Plan</dt><dd>{{ activeChapter.CurrentPlanRevisionID || 'BLOCKED — chưa có plan revision hiện hành' }}</dd></div>
          <div><dt>Generation</dt><dd>{{ generationRun ? `${generationRun.Status} · ${generationRun.ID}` : latestRevision?.GenerationRunID ? `WAITING — durable run ${latestRevision.GenerationRunID} chưa được projection trả về; tạo run mới bị chặn` : latestRevision ? `Output revision #${latestRevision.RevisionNo}` : selectionLoading ? 'Đang tải…' : 'Chưa có durable run/output' }}</dd></div>
          <div><dt>Approved content</dt><dd>{{ approvedRevision ? `Revision #${approvedRevision.RevisionNo} · ${approvedRevision.ID}` : selectionLoading ? 'Đang tải…' : 'WAITING — chưa có approved revision' }}</dd></div>
          <div><dt>Narration</dt><dd>{{ narrationStatus }}</dd></div>
          <div><dt>Audio</dt><dd>{{ audioStatus }}</dd></div>
          <div><dt>Publish</dt><dd>{{ publishStatus }}</dd></div>
        </dl>

        <div class="actions">
          <button v-if="!generationRun" type="button" :disabled="!mayStartGeneration" @click="startGeneration">
            {{ action === 'start-generation' ? 'Đang bắt đầu…' : 'Start Generation' }}
          </button>
          <button v-else type="button" :disabled="Boolean(action) || selectionLoading" @click="refreshGeneration">
            {{ action === 'refresh-generation' ? 'Đang refresh…' : 'Refresh Run' }}
          </button>
          <button type="button" :disabled="!mayCreateNarration" @click="startNarration">
            {{ action === 'create-narration' ? 'Đang tạo narration…' : 'Create Narration' }}
          </button>
          <button type="button" :disabled="!maySynthesize" @click="runSynthesize">
            {{ action === 'synthesize' ? 'Đang synthesize…' : 'Synthesize TTS' }}
          </button>
          <button type="button" :disabled="!mayActivate" @click="runActivate">
            {{ action === 'activate' ? 'Đang activate…' : 'Activate Audio' }}
          </button>
          <RouterLink :to="`/admin/stories/${storyID}/review`">Mở Content Review</RouterLink>
        </div>

        <p v-if="latestRevision">Latest revision {{ latestRevision.ID }} · source {{ latestRevision.SourceType }} · run {{ latestRevision.GenerationRunID || '—' }}</p>
        <p v-if="latestNarration">Narration {{ latestNarration.ID }} · voice {{ latestNarration.VoiceID }} · status {{ latestNarration.Status }}</p>
        <p v-if="latestReadyAudio">Latest READY asset {{ latestReadyAudio.ID }} · narration {{ latestReadyAudio.SourceNarrationRevisionID }} · {{ latestReadyAudio.SizeBytes }} bytes</p>
        <p>Review records loaded: {{ reviews.length }}</p>
      </main>
    </div>
  </section>
</template>

<style scoped>
.production-page { max-width: 1180px; margin: 0 auto; padding: 32px 24px 64px; }
.production-header { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; margin-bottom: 24px; }
.workspace-grid { display: grid; grid-template-columns: minmax(220px, 280px) 1fr; gap: 24px; }
.chapter-panel, .pipeline-panel { border: 1px solid #d8d8d8; border-radius: 16px; padding: 18px; }
.chapter-panel button { width: 100%; display: grid; gap: 3px; text-align: left; margin: 8px 0; padding: 12px; border: 1px solid transparent; border-radius: 10px; background: transparent; }
.chapter-panel button.active { border-color: currentColor; }
.chapter-panel button:disabled { opacity: 0.55; cursor: not-allowed; }
dl { display: grid; gap: 10px; }
dl div { display: grid; grid-template-columns: 160px 1fr; gap: 12px; }
dt { font-weight: 700; }
dd { margin: 0; overflow-wrap: anywhere; }
.actions { display: flex; flex-wrap: wrap; gap: 12px; align-items: center; margin-top: 20px; }
.error { color: #b42318; }
.eyebrow { font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; opacity: .65; }
@media (max-width: 800px) { .workspace-grid { grid-template-columns: 1fr; } .production-header { flex-direction: column; } }
</style>
