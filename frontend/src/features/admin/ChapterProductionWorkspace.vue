<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { listAdminChapters, listChapterReviews, listContentRevisions } from '../../api/client'
import type { Chapter, ChapterReview, ContentRevision } from '../../api/types'
import { getGenerationRun, startChapterGeneration, type GenerationRun } from './chapterProductionApi'
import { createLatestSelectionGuard } from './latestSelection.mjs'

const route = useRoute()
const storyID = computed(() => route.params.storyID as string)
const chapters = ref<Chapter[]>([])
const activeChapter = ref<Chapter | null>(null)
const revisions = ref<ContentRevision[]>([])
const reviews = ref<ChapterReview[]>([])
const generationRun = ref<GenerationRun | null>(null)
const loading = ref(false)
const action = ref('')
const error = ref('')
const chapterSelection = createLatestSelectionGuard()

const latestRevision = computed(() => revisions.value[revisions.value.length - 1] ?? null)
const approvedRevision = computed(() => [...revisions.value].reverse().find((revision) => revision.Status === 'APPROVED') ?? null)
const mayStartGeneration = computed(() => Boolean(activeChapter.value?.CurrentPlanRevisionID && !action.value))

async function selectChapter(chapter: Chapter) {
  const mayCommit = chapterSelection.begin(chapter.ID)
  activeChapter.value = chapter
  generationRun.value = null
  error.value = ''
  try {
    const [revisionResponse, reviewResponse] = await Promise.all([
      listContentRevisions(chapter.ID),
      listChapterReviews(chapter.ID),
    ])
    if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
    revisions.value = revisionResponse.revisions
    reviews.value = reviewResponse.reviews
    const latest = revisionResponse.revisions[revisionResponse.revisions.length - 1] ?? null
    if (latest?.GenerationRunID) {
      try {
        const run = await getGenerationRun(latest.GenerationRunID)
        if (mayCommit() && activeChapter.value?.ID === chapter.ID) generationRun.value = run
      } catch {
        if (mayCommit() && activeChapter.value?.ID === chapter.ID) generationRun.value = null
      }
    }
  } catch (e) {
    if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
    revisions.value = []
    reviews.value = []
    error.value = e instanceof Error ? e.message : 'Không thể tải trạng thái production của chương.'
  }
}

async function startGeneration() {
  const chapter = activeChapter.value
  if (!chapter || !chapter.CurrentPlanRevisionID || action.value) return
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
  if (!chapter || !run || action.value) return
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

async function load() {
  loading.value = true
  error.value = ''
  try {
    const response = await listAdminChapters(storyID.value)
    chapters.value = response.chapters
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
        <button v-for="chapter in chapters" :key="chapter.ID" type="button" :class="{ active: activeChapter?.ID === chapter.ID }" @click="selectChapter(chapter)">
          <span>Chương {{ chapter.ChapterNumber }}</span>
          <strong>{{ chapter.Title }}</strong>
          <small>{{ chapter.Status }}</small>
        </button>
      </aside>

      <main v-if="activeChapter" class="pipeline-panel">
        <h2>{{ activeChapter.Title }}</h2>
        <dl>
          <div><dt>Plan</dt><dd>{{ activeChapter.CurrentPlanRevisionID || 'BLOCKED — chưa có plan revision hiện hành' }}</dd></div>
          <div><dt>Generation</dt><dd>{{ generationRun ? `${generationRun.Status} · ${generationRun.ID}` : latestRevision ? `Output revision #${latestRevision.RevisionNo}` : 'Chưa có durable run/output được khôi phục' }}</dd></div>
          <div><dt>Approved content</dt><dd>{{ approvedRevision ? `Revision #${approvedRevision.RevisionNo}` : 'WAITING — chưa có approved revision' }}</dd></div>
          <div><dt>Narration / Audio / Publish</dt><dd>BLOCKED trong slice này cho tới khi backend projection/action authoritative được nối; không fake readiness từ frontend.</dd></div>
        </dl>

        <div class="actions">
          <button v-if="!generationRun" type="button" :disabled="!mayStartGeneration" @click="startGeneration">
            {{ action === 'start-generation' ? 'Đang bắt đầu…' : 'Start Generation' }}
          </button>
          <button v-else type="button" :disabled="Boolean(action)" @click="refreshGeneration">
            {{ action === 'refresh-generation' ? 'Đang refresh…' : 'Refresh Run' }}
          </button>
          <RouterLink :to="`/admin/stories/${storyID}/review`">Mở Content Review</RouterLink>
        </div>

        <p v-if="latestRevision">Latest revision {{ latestRevision.ID }} · source {{ latestRevision.SourceType }} · run {{ latestRevision.GenerationRunID || '—' }}</p>
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
dl { display: grid; gap: 10px; }
dl div { display: grid; grid-template-columns: 160px 1fr; gap: 12px; }
dt { font-weight: 700; }
dd { margin: 0; overflow-wrap: anywhere; }
.actions { display: flex; gap: 12px; align-items: center; margin-top: 20px; }
.error { color: #b42318; }
.eyebrow { font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; opacity: .65; }
@media (max-width: 800px) { .workspace-grid { grid-template-columns: 1fr; } .production-header { flex-direction: column; } }
</style>
