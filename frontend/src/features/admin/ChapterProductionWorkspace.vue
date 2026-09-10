<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  activateAudioAsset,
  createNarrationRevision,
  getActiveAudioAsset,
  retryGenerationJob,
  type GenerationJobView,
  getLatestNarrationRevision,
  getLatestReadyAudioAsset,
  getPublishReadiness,
  getStoryWorkflowSettings,
  listAdminChapters,
  listChapterReviews,
  listContentRevisions,
  markChapterReady,
  publishChapter,
  startChapterGeneration,
  synthesizeNarration,
  type GenerationRun,
  type PublishReadiness,
} from '../../api/client'
import type { AudioAsset, Chapter, ChapterReview, ContentRevision, NarrationRevision } from '../../api/types'
import {
  canActivateAudio,
  canCreateNarration,
  canMarkChapterReady,
  canPublishChapter,
  canRetryGenerationJob,
  canSelectChapter,
  canStartChapterGeneration,
  canSynthesizeNarration,
  createLatestSelectionGuard,
  formatGenerationJobStatus,
  generationJobFromContentResponse,
  generationRunFromContentResponse,
} from './latestSelection.mjs'
import {
  buildChapterProductionStages,
  getPrimaryProductionStage,
  stageStateLabel,
} from './chapterProductionStages.mjs'

const route = useRoute()
const storyID = computed(() => route.params.storyID as string)
const chapters = ref<Chapter[]>([])
const activeChapter = ref<Chapter | null>(null)
const revisions = ref<ContentRevision[]>([])
const reviews = ref<ChapterReview[]>([])
const generationRun = ref<GenerationRun | null>(null)
const generationJob = ref<GenerationJobView | null>(null)
const latestNarration = ref<NarrationRevision | null>(null)
const activeAudio = ref<AudioAsset | null>(null)
const latestReadyAudio = ref<AudioAsset | null>(null)
const publishReadiness = ref<PublishReadiness | null>(null)
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

const mayRetryGeneration = computed(() => canRetryGenerationJob({
  generationJob: generationJob.value,
  selectionLoading: selectionLoading.value,
  actionInProgress: Boolean(action.value),
}))

const generationJobStatus = computed(() => {
  if (selectionLoading.value) return 'Đang tải…'
  const formatted = formatGenerationJobStatus(generationJob.value)
  if (formatted) return formatted
  if (generationRun.value) return `${generationRun.value.Status} · ${generationRun.value.ID} (chưa có job projection)`
  if (latestRevision.value?.GenerationRunID) return `WAITING — durable run ${latestRevision.value.GenerationRunID} chưa có job projection`
  return latestRevision.value ? `Output revision #${latestRevision.value.RevisionNo}` : 'Chưa có durable run/output'
})

const mayMarkReady = computed(() => canMarkChapterReady({
  chapterStatus: activeChapter.value?.Status,
  publishReadiness: publishReadiness.value,
  selectionLoading: selectionLoading.value,
  actionInProgress: Boolean(action.value),
}))

const mayPublish = computed(() => canPublishChapter({
  chapterStatus: activeChapter.value?.Status,
  publishReadiness: publishReadiness.value,
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

const publishMissingLabels: Record<string, string> = {
  approved_content: 'Approved content',
  approved_content_authority: 'Approved content authority',
  narration: 'Narration revision',
  active_audio: 'Active durable audio',
  active_audio_authority: 'Active audio authority',
  active_audio_stale: 'Active audio khớp narration mới nhất',
  story_status: 'Story status cho phép publish',
  story_publish_authority: 'Story publish authority',
  publish_authority: 'Publish authority',
}

const publishStatus = computed(() => {
  if (selectionLoading.value) return 'Đang tải…'
  if (activeChapter.value?.Status === 'PUBLISHED') return `PUBLISHED · ${activeChapter.value.ID}`
  if (!publishReadiness.value) return 'WAITING — chưa có publish readiness authoritative'
  if (activeChapter.value?.Status !== 'READY') {
    if (publishReadiness.value.ready) {
      return `READY gates passed — chapter ${activeChapter.value?.Status || '—'} có thể Mark Ready trước publish`
    }
    const missing = publishReadiness.value.missing.map((item) => publishMissingLabels[item] ?? item.replaceAll('_', ' '))
    return `BLOCKED — thiếu: ${missing.join(', ')}`
  }
  if (publishReadiness.value.ready) return 'READY — backend xác nhận đủ điều kiện publish'
  const missing = publishReadiness.value.missing.map((item) => publishMissingLabels[item] ?? item.replaceAll('_', ' '))
  return `BLOCKED — thiếu: ${missing.join(', ')}`
})

const productionStages = computed(() => buildChapterProductionStages({
  selectionLoading: selectionLoading.value,
  hasPlanRevision: Boolean(activeChapter.value?.CurrentPlanRevisionID),
  generationJobStatus: generationJobStatus.value,
  hasApprovedRevision: Boolean(approvedRevision.value),
  narrationStatus: narrationStatus.value,
  audioStatus: audioStatus.value,
  publishStatus: publishStatus.value,
  chapterPublished: activeChapter.value?.Status === 'PUBLISHED',
}))
const primaryStage = computed(() => getPrimaryProductionStage(productionStages.value))

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
  generationJob.value = null
  latestNarration.value = null
  activeAudio.value = null
  latestReadyAudio.value = null
  publishReadiness.value = null
  selectionLoading.value = true
  error.value = ''
  try {
    const [revisionResponse, reviewResponse, audioState, readiness] = await Promise.all([
      listContentRevisions(chapter.ID),
      listChapterReviews(chapter.ID),
      loadAudioProjections(chapter.ID),
      getPublishReadiness(chapter.ID),
    ])
    if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return

    const authoritativeContentState = revisionResponse as typeof revisionResponse & {
      generation_run?: GenerationRun | null
      generation_job?: GenerationJobView | null
    }
    revisions.value = revisionResponse.revisions
    reviews.value = reviewResponse.reviews
    generationRun.value = generationRunFromContentResponse(authoritativeContentState)
    generationJob.value = generationJobFromContentResponse(authoritativeContentState)
    latestNarration.value = audioState.narration
    activeAudio.value = audioState.active
    latestReadyAudio.value = audioState.ready
    publishReadiness.value = readiness
  } catch (e) {
    if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
    revisions.value = []
    reviews.value = []
    generationRun.value = null
    generationJob.value = null
    latestNarration.value = null
    activeAudio.value = null
    latestReadyAudio.value = null
    publishReadiness.value = null
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
  if (!chapter || action.value || selectionLoading.value) return
  action.value = 'refresh-generation'
  error.value = ''
  try {
    const revisionResponse = await listContentRevisions(chapter.ID)
    if (activeChapter.value?.ID !== chapter.ID) return
    const authoritativeContentState = revisionResponse as typeof revisionResponse & {
      generation_run?: GenerationRun | null
      generation_job?: GenerationJobView | null
    }
    generationRun.value = generationRunFromContentResponse(authoritativeContentState)
    generationJob.value = generationJobFromContentResponse(authoritativeContentState)
  } catch (e) {
    if (activeChapter.value?.ID === chapter.ID) error.value = e instanceof Error ? e.message : 'Không thể refresh generation state.'
  } finally {
    action.value = ''
  }
}

async function runRetryGeneration() {
  const chapter = activeChapter.value
  const job = generationJob.value
  if (!chapter || !job || !mayRetryGeneration.value) return
  const requestChapterID = chapter.ID
  const requestJobID = job.ID
  action.value = 'retry-generation'
  error.value = ''
  try {
    const retried = await retryGenerationJob(requestJobID)
    if (activeChapter.value?.ID !== requestChapterID || generationJob.value?.ID !== requestJobID) return
    generationJob.value = retried
  } catch (e) {
    if (activeChapter.value?.ID === requestChapterID && generationJob.value?.ID === requestJobID) {
      error.value = e instanceof Error ? e.message : 'Không thể retry generation job.'
    }
  } finally {
    if (activeChapter.value?.ID === requestChapterID) action.value = ''
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
    if (activeChapter.value?.ID === chapter.ID) publishReadiness.value = await getPublishReadiness(chapter.ID)
  } catch (e) {
    if (activeChapter.value?.ID === chapter.ID) {
      error.value = e instanceof Error ? e.message : 'Không thể activate READY audio asset.'
    }
  } finally {
    if (activeChapter.value?.ID === chapter.ID) action.value = ''
  }
}

async function runMarkReady() {
  const chapter = activeChapter.value
  if (!chapter || !mayMarkReady.value) return
  const requestChapterID = chapter.ID
  action.value = 'mark-ready'
  error.value = ''
  try {
    const ready = await markChapterReady(requestChapterID)
    if (activeChapter.value?.ID !== requestChapterID) return
    activeChapter.value = ready
    chapters.value = chapters.value.map((item) => (item.ID === ready.ID ? ready : item))
  } catch (e) {
    if (activeChapter.value?.ID === requestChapterID) {
      error.value = e instanceof Error ? e.message : 'Không thể mark chapter ready.'
    }
  } finally {
    if (activeChapter.value?.ID === requestChapterID) action.value = ''
  }
}

async function runPublish() {
  const chapter = activeChapter.value
  if (!chapter || !mayPublish.value) return
  const requestChapterID = chapter.ID
  action.value = 'publish'
  error.value = ''
  try {
    const published = await publishChapter(requestChapterID)
    if (activeChapter.value?.ID !== requestChapterID) return
    activeChapter.value = published
    chapters.value = chapters.value.map((item) => (item.ID === published.ID ? published : item))
    publishReadiness.value = await getPublishReadiness(requestChapterID)
  } catch (e) {
    if (activeChapter.value?.ID === requestChapterID) {
      error.value = e instanceof Error ? e.message : 'Không thể publish chapter.'
    }
  } finally {
    if (activeChapter.value?.ID === requestChapterID) action.value = ''
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
        <p>Đi theo pipeline Plan → Generation → Review → Narration → Audio → Publish. Mọi trạng thái và blocker vẫn lấy authority từ backend.</p>
      </div>
      <RouterLink class="back-link" :to="`/admin/stories/${storyID}/planning`">← Story Planning Studio</RouterLink>
    </header>

    <p v-if="loading" class="page-status" role="status">Đang tải production workspace...</p>
    <p v-if="error" class="error" role="alert">{{ error }}</p>

    <div v-if="!loading" class="workspace-grid">
      <aside class="chapter-panel" aria-label="Danh sách chương">
        <div class="panel-heading">
          <div>
            <p class="eyebrow">Story chapters</p>
            <h2>Chapters</h2>
          </div>
          <span class="count-badge">{{ chapters.length }}</span>
        </div>
        <div v-if="chapters.length" class="chapter-list">
          <button
            v-for="chapter in chapters"
            :key="chapter.ID"
            type="button"
            :class="['chapter-card', { active: activeChapter?.ID === chapter.ID }]"
            :disabled="!maySelectChapter"
            :aria-disabled="!maySelectChapter"
            :aria-current="activeChapter?.ID === chapter.ID ? 'true' : undefined"
            @click="selectChapter(chapter)"
          >
            <span class="chapter-number">Chương {{ chapter.ChapterNumber }}</span>
            <strong>{{ chapter.Title }}</strong>
            <small>{{ chapter.Status }}</small>
          </button>
        </div>
        <p v-else class="empty-state">Chưa có chapter để production.</p>
      </aside>

      <main v-if="activeChapter" class="pipeline-panel" :aria-busy="selectionLoading">
        <div class="chapter-heading">
          <div>
            <p class="eyebrow">Chương {{ activeChapter.ChapterNumber }}</p>
            <h2>{{ activeChapter.Title }}</h2>
          </div>
          <span class="chapter-state">{{ activeChapter.Status }}</span>
        </div>

        <p v-if="selectionLoading" class="page-status" role="status">Đang tải trạng thái authoritative của chương…</p>

        <section v-if="primaryStage" class="next-step" aria-labelledby="next-step-title">
          <div>
            <p class="eyebrow">Next attention</p>
            <h3 id="next-step-title">{{ primaryStage.label }}</h3>
          </div>
          <div>
            <span :class="['state-badge', `state-${primaryStage.state}`]">{{ stageStateLabel(primaryStage.state) }}</span>
            <p>{{ primaryStage.summary }}</p>
            <p v-if="primaryStage.blocker" class="blocker-copy">{{ primaryStage.blocker }}</p>
          </div>
        </section>

        <ol class="stage-list" aria-label="Chapter production pipeline">
          <li v-for="(stage, index) in productionStages" :key="stage.id" :class="['stage-card', `stage-${stage.state}`]">
            <div class="stage-index" aria-hidden="true">{{ index + 1 }}</div>
            <div class="stage-copy">
              <div class="stage-title-row">
                <h3>{{ stage.label }}</h3>
                <span :class="['state-badge', `state-${stage.state}`]">{{ stageStateLabel(stage.state) }}</span>
              </div>
              <p>{{ stage.summary }}</p>
              <p v-if="stage.blocker" class="blocker-copy">{{ stage.blocker }}</p>
            </div>
          </li>
        </ol>

        <section class="action-panel" aria-labelledby="actions-title">
          <div class="panel-heading">
            <div>
              <p class="eyebrow">Safe actions</p>
              <h3 id="actions-title">Production controls</h3>
            </div>
            <span v-if="action" class="action-progress" role="status">Đang xử lý…</span>
          </div>
          <div class="actions">
            <button v-if="!generationRun" type="button" :disabled="!mayStartGeneration" @click="startGeneration">
              {{ action === 'start-generation' ? 'Đang bắt đầu…' : 'Start Generation' }}
            </button>
            <button v-if="generationRun || generationJob" type="button" :disabled="Boolean(action) || selectionLoading" @click="refreshGeneration">
              {{ action === 'refresh-generation' ? 'Đang refresh…' : 'Refresh Generation' }}
            </button>
            <button v-if="mayRetryGeneration" type="button" :disabled="!mayRetryGeneration" @click="runRetryGeneration">
              {{ action === 'retry-generation' ? 'Đang retry…' : 'Retry Generation' }}
            </button>
            <RouterLink class="action-link" :to="`/admin/stories/${storyID}/review`">Mở Content Review</RouterLink>
            <button type="button" :disabled="!mayCreateNarration" @click="startNarration">
              {{ action === 'create-narration' ? 'Đang tạo narration…' : 'Create Narration' }}
            </button>
            <button type="button" :disabled="!maySynthesize" @click="runSynthesize">
              {{ action === 'synthesize' ? 'Đang synthesize…' : 'Synthesize TTS' }}
            </button>
            <button type="button" :disabled="!mayActivate" @click="runActivate">
              {{ action === 'activate' ? 'Đang activate…' : 'Activate Audio' }}
            </button>
            <button type="button" :disabled="!mayMarkReady" @click="runMarkReady">
              {{ action === 'mark-ready' ? 'Đang mark ready…' : 'Mark Chapter Ready' }}
            </button>
            <button class="primary-action" type="button" :disabled="!mayPublish" @click="runPublish">
              {{ action === 'publish' ? 'Đang publish…' : 'Publish Chapter' }}
            </button>
          </div>
        </section>

        <details class="technical-details">
          <summary>Technical production details</summary>
          <div class="detail-grid">
            <p v-if="latestRevision"><strong>Latest revision</strong><span>{{ latestRevision.ID }} · {{ latestRevision.SourceType }} · run {{ latestRevision.GenerationRunID || '—' }}</span></p>
            <p v-if="latestNarration"><strong>Narration</strong><span>{{ latestNarration.ID }} · voice {{ latestNarration.VoiceID }} · {{ latestNarration.Status }}</span></p>
            <p v-if="activeAudio"><strong>Active audio</strong><span>v{{ activeAudio.VersionNo }} · {{ activeAudio.DurationMs }}ms · checksum {{ activeAudio.Checksum || '—' }}</span></p>
            <p v-if="latestReadyAudio"><strong>Latest READY audio</strong><span>{{ latestReadyAudio.ID }} · {{ latestReadyAudio.SizeBytes }} bytes</span></p>
            <p><strong>Review records</strong><span>{{ reviews.length }}</span></p>
          </div>
        </details>
      </main>
    </div>
  </section>
</template>

<style scoped>
.production-page { max-width: 1240px; margin: 0 auto; padding: 32px 24px 64px; }
.production-header { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; margin-bottom: 24px; }
.production-header h1, .chapter-heading h2, .panel-heading h2, .panel-heading h3, .stage-card h3, .next-step h3 { margin: 0; }
.production-header p:not(.eyebrow) { max-width: 760px; margin-bottom: 0; line-height: 1.6; }
.back-link, .action-link { min-height: 44px; display: inline-flex; align-items: center; }
.workspace-grid { display: grid; grid-template-columns: minmax(240px, 300px) minmax(0, 1fr); gap: 24px; align-items: start; }
.chapter-panel, .pipeline-panel { border: 1px solid var(--border-color, #d8d8d8); border-radius: 18px; background: var(--surface-color, #fff); }
.chapter-panel { padding: 16px; position: sticky; top: 20px; }
.pipeline-panel { min-width: 0; padding: 22px; }
.panel-heading, .chapter-heading, .stage-title-row { display: flex; justify-content: space-between; gap: 16px; align-items: center; }
.chapter-heading { margin-bottom: 20px; align-items: flex-start; }
.chapter-list { display: grid; gap: 8px; margin-top: 12px; }
.chapter-card { min-height: 68px; width: 100%; display: grid; gap: 4px; text-align: left; padding: 12px 14px; border: 1px solid transparent; border-radius: 12px; background: transparent; overflow-wrap: anywhere; cursor: pointer; }
.chapter-card:hover:not(:disabled), .chapter-card:focus-visible { border-color: currentColor; }
.chapter-card.active { border-color: currentColor; background: color-mix(in srgb, currentColor 7%, transparent); }
.chapter-card:disabled { opacity: .55; cursor: not-allowed; }
.chapter-number, .chapter-card small { font-size: 12px; opacity: .72; }
.count-badge, .chapter-state, .state-badge { display: inline-flex; align-items: center; justify-content: center; min-height: 28px; padding: 4px 9px; border: 1px solid currentColor; border-radius: 999px; font-size: 12px; font-weight: 700; white-space: nowrap; }
.next-step { display: grid; grid-template-columns: minmax(130px, .35fr) 1fr; gap: 18px; padding: 18px; margin-bottom: 18px; border: 1px solid currentColor; border-radius: 14px; }
.next-step p { margin: 8px 0 0; overflow-wrap: anywhere; }
.stage-list { list-style: none; display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; padding: 0; margin: 0; }
.stage-card { min-width: 0; display: grid; grid-template-columns: 34px minmax(0, 1fr); gap: 12px; padding: 16px; border: 1px solid var(--border-color, #d8d8d8); border-radius: 14px; }
.stage-index { width: 32px; height: 32px; display: grid; place-items: center; border: 1px solid currentColor; border-radius: 50%; font-weight: 800; }
.stage-copy { min-width: 0; }
.stage-copy p { margin: 8px 0 0; line-height: 1.45; overflow-wrap: anywhere; }
.blocker-copy { font-weight: 650; }
.stage-loading, .state-loading, .stage-waiting, .state-waiting { opacity: .72; }
.stage-blocked, .state-blocked, .stage-failed, .state-failed { border-style: dashed; }
.action-panel { margin-top: 20px; padding-top: 20px; border-top: 1px solid var(--border-color, #d8d8d8); }
.actions { display: flex; flex-wrap: wrap; gap: 10px; align-items: center; margin-top: 14px; }
.actions button, .action-link { min-height: 44px; padding: 9px 14px; border-radius: 10px; font: inherit; }
.actions button { border: 1px solid currentColor; background: transparent; cursor: pointer; }
.actions button:hover:not(:disabled), .actions button:focus-visible, .action-link:focus-visible { outline: 2px solid currentColor; outline-offset: 2px; }
.actions button:disabled { opacity: .5; cursor: not-allowed; }
.primary-action { font-weight: 800; }
.action-progress, .page-status { font-weight: 650; }
.technical-details { margin-top: 20px; border-top: 1px solid var(--border-color, #d8d8d8); padding-top: 16px; }
.technical-details summary { min-height: 44px; display: flex; align-items: center; cursor: pointer; font-weight: 700; }
.detail-grid { display: grid; gap: 8px; }
.detail-grid p { display: grid; grid-template-columns: 150px minmax(0, 1fr); gap: 12px; margin: 0; }
.detail-grid span { overflow-wrap: anywhere; }
.empty-state { opacity: .72; }
.error { color: #b42318; font-weight: 650; }
.eyebrow { margin: 0 0 5px; font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; opacity: .65; }

@media (max-width: 1024px) {
  .workspace-grid { grid-template-columns: minmax(210px, 250px) minmax(0, 1fr); gap: 16px; }
  .stage-list { grid-template-columns: 1fr; }
  .next-step { grid-template-columns: 1fr; }
}

@media (max-width: 760px) {
  .production-page { padding: 20px 16px 48px; }
  .production-header { flex-direction: column; margin-bottom: 18px; }
  .workspace-grid { grid-template-columns: 1fr; }
  .chapter-panel { position: static; padding: 14px; }
  .chapter-list { display: flex; gap: 8px; overflow-x: auto; padding-bottom: 4px; scroll-snap-type: x proximity; }
  .chapter-card { min-width: min(78vw, 280px); scroll-snap-align: start; }
  .pipeline-panel { padding: 16px; }
  .chapter-heading { align-items: flex-start; }
  .stage-title-row { align-items: flex-start; }
  .actions { display: grid; grid-template-columns: 1fr; }
  .actions button, .action-link { width: 100%; justify-content: center; text-align: center; }
  .detail-grid p { grid-template-columns: 1fr; gap: 2px; }
}
</style>
