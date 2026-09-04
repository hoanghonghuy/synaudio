<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  listAdminChapters,
  listChapterReviews,
  listContentRevisions,
  regenerateContent,
} from '../../api/client'
import type { Chapter, ChapterReview, ContentRevision } from '../../api/types'
import { createLatestSelectionGuard } from './latestSelection.mjs'

const route = useRoute()
const storyID = computed(() => route.params.storyID as string)
const chapters = ref<Chapter[]>([])
const activeChapter = ref<Chapter | null>(null)
const revisions = ref<ContentRevision[]>([])
const reviews = ref<ChapterReview[]>([])
const loading = ref(false)
const action = ref('')
const error = ref('')
const success = ref('')
const chapterSelection = createLatestSelectionGuard()

const latestRevision = computed(() => revisions.value[revisions.value.length - 1] ?? null)
const approvedRevision = computed(() => [...revisions.value].reverse().find((revision) => revision.Status === 'APPROVED') ?? null)
const latestRevisionReviews = computed(() => {
  if (!latestRevision.value) return []
  return reviews.value.filter((review) => review.ContentRevisionID === latestRevision.value?.ID)
})
const reviewOutcomes = computed(() => new Map(latestRevisionReviews.value.map((review) => [review.ReviewType, review.Outcome])))

const stages = computed(() => {
  const chapter = activeChapter.value
  const latest = latestRevision.value
  const approved = approvedRevision.value
  const hasPlan = Boolean(chapter?.CurrentPlanRevisionID)
  const hasGeneratedContent = Boolean(latest)
  const requiredReviews = ['CONTINUITY', 'QUALITY', 'SAFETY']
  const reviewsPassed = requiredReviews.every((type) => reviewOutcomes.value.get(type) === 'PASS')

  return [
    {
      key: 'plan',
      label: 'Plan',
      state: hasPlan ? 'ready' : 'blocked',
      detail: hasPlan ? `Plan revision ${chapter?.CurrentPlanRevisionID}` : 'Chưa có Chapter Plan hiện hành.',
    },
    {
      key: 'generation',
      label: 'Generation',
      state: hasGeneratedContent ? 'ready' : hasPlan ? 'waiting' : 'blocked',
      detail: latest
        ? `Revision #${latest.RevisionNo} · ${latest.SourceType}${latest.GenerationRunID ? ` · run ${latest.GenerationRunID}` : ''}`
        : hasPlan
          ? 'Plan đã sẵn sàng; chưa có content revision để quan sát.'
          : 'Cần Chapter Plan trước khi đi vào generation.',
    },
    {
      key: 'review',
      label: 'Review',
      state: reviewsPassed ? 'ready' : hasGeneratedContent ? 'waiting' : 'blocked',
      detail: hasGeneratedContent
        ? `Continuity: ${reviewOutcomes.value.get('CONTINUITY') ?? '—'} · Quality: ${reviewOutcomes.value.get('QUALITY') ?? '—'} · Safety: ${reviewOutcomes.value.get('SAFETY') ?? '—'}`
        : 'Chưa có content revision để review.',
    },
    {
      key: 'approved',
      label: 'Approved Content',
      state: approved ? 'ready' : hasGeneratedContent ? 'waiting' : 'blocked',
      detail: approved ? `Approved revision #${approved.RevisionNo} (${approved.ID})` : 'Chưa có revision APPROVED.',
    },
    {
      key: 'canon',
      label: 'Canon / Memory',
      state: approved ? 'waiting' : 'blocked',
      detail: approved
        ? 'Approved content đã có; cần backend workspace projection để xác nhận extraction/commit state.'
        : 'Cần approved content trước.',
    },
    {
      key: 'narration',
      label: 'Narration',
      state: approved ? 'waiting' : 'blocked',
      detail: approved
        ? 'Cần backend workspace projection để hiển thị narration revision và provenance.'
        : 'Cần approved content trước.',
    },
    {
      key: 'audio',
      label: 'TTS / Audio',
      state: 'blocked',
      detail: 'Không suy diễn trạng thái audio từ listener URL; cần authoritative production projection.',
    },
    {
      key: 'publish',
      label: 'Publish',
      state: chapter?.Status === 'PUBLISHED' ? 'ready' : 'blocked',
      detail: chapter?.Status === 'PUBLISHED'
        ? 'Chapter đang ở trạng thái PUBLISHED.'
        : 'Publish luôn explicit và chỉ mở khi backend gates xác nhận đủ điều kiện.',
    },
  ]
})

async function selectChapter(chapter: Chapter) {
  const mayCommit = chapterSelection.begin(chapter.ID)
  activeChapter.value = chapter
  error.value = ''
  success.value = ''
  try {
    const [revisionResponse, reviewResponse] = await Promise.all([
      listContentRevisions(chapter.ID),
      listChapterReviews(chapter.ID),
    ])
    if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
    revisions.value = revisionResponse.revisions
    reviews.value = reviewResponse.reviews
  } catch (e) {
    if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
    revisions.value = []
    reviews.value = []
    error.value = e instanceof Error ? e.message : 'Không thể tải trạng thái production của chương.'
  }
}

async function regenerateLatest() {
  const chapter = activeChapter.value
  const revision = latestRevision.value
  if (!chapter || !revision || action.value) return

  action.value = 'regenerate'
  error.value = ''
  success.value = ''
  try {
    await regenerateContent(chapter.ID, revision.ID)
    if (activeChapter.value?.ID !== chapter.ID) return
    success.value = `Đã tạo Regenerate từ revision #${revision.RevisionNo}; đây là revision mới, không phải Retry của attempt cũ.`
    await selectChapter(chapter)
  } catch (e) {
    if (activeChapter.value?.ID !== chapter.ID) return
    error.value = e instanceof Error ? e.message : 'Không thể Regenerate content revision.'
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
        <h1>Chapter Production Pipeline</h1>
        <p>
          Workspace này chỉ hiển thị state có bằng chứng từ backend hiện tại. Các stage chưa có authoritative projection sẽ được đánh dấu blocker thay vì suy diễn ở client.
        </p>
      </div>
      <RouterLink class="secondary-link" :to="`/admin/stories/${storyID}/planning`">← Story Planning Studio</RouterLink>
    </header>

    <p v-if="loading" class="status-state">Đang tải production workspace...</p>
    <p v-else-if="error && chapters.length === 0" class="status-state error" role="alert">{{ error }}</p>

    <div v-else class="workspace-grid">
      <aside class="chapter-panel">
        <h2>Chapters</h2>
        <button
          v-for="chapter in chapters"
          :key="chapter.ID"
          class="chapter-button"
          :class="{ active: activeChapter?.ID === chapter.ID }"
          type="button"
          @click="selectChapter(chapter)"
        >
          <span>Chương {{ chapter.ChapterNumber }}</span>
          <strong>{{ chapter.Title }}</strong>
          <small>{{ chapter.Status }}</small>
        </button>
        <p v-if="chapters.length === 0" class="muted">Chưa có chapter nào.</p>
      </aside>

      <main class="pipeline-panel">
        <div v-if="activeChapter" class="chapter-heading">
          <div>
            <p class="eyebrow">Chapter {{ activeChapter.ChapterNumber }}</p>
            <h2>{{ activeChapter.Title }}</h2>
          </div>
          <RouterLink class="secondary-link" :to="`/admin/stories/${storyID}/review`">Mở Content Review</RouterLink>
        </div>

        <p v-if="error" class="status-state error" role="alert">{{ error }}</p>
        <p v-if="success" class="status-state success" role="status">{{ success }}</p>

        <section v-if="activeChapter" class="action-panel">
          <div>
            <strong>Production actions</strong>
            <p>Regenerate tạo output/revision mới từ revision hiện tại. Retry của một failed attempt là semantics khác và chỉ được mở khi backend expose đúng attempt/job recovery endpoint.</p>
          </div>
          <button
            type="button"
            :disabled="!latestRevision || Boolean(action)"
            @click="regenerateLatest"
          >
            {{ action === 'regenerate' ? 'Đang Regenerate…' : 'Regenerate latest revision' }}
          </button>
        </section>

        <ol v-if="activeChapter" class="stage-list">
          <li v-for="stage in stages" :key="stage.key" class="stage-card" :data-state="stage.state">
            <div class="stage-state" aria-hidden="true"></div>
            <div>
              <strong>{{ stage.label }}</strong>
              <p>{{ stage.detail }}</p>
            </div>
          </li>
        </ol>

        <section v-if="latestRevision" class="provenance-panel">
          <h3>Current content provenance</h3>
          <dl>
            <div><dt>Revision</dt><dd>{{ latestRevision.ID }}</dd></div>
            <div><dt>Based on</dt><dd>{{ latestRevision.BasedOnRevisionID || '—' }}</dd></div>
            <div><dt>Plan revision</dt><dd>{{ latestRevision.PlanRevisionID || '—' }}</dd></div>
            <div><dt>Generation run</dt><dd>{{ latestRevision.GenerationRunID || '—' }}</dd></div>
            <div><dt>Base canon</dt><dd>{{ latestRevision.BaseCanonVersionID || '—' }}</dd></div>
          </dl>
        </section>
      </main>
    </div>
  </section>
</template>

<style scoped>
.production-page { max-width: 1280px; margin: 0 auto; padding: 32px 24px 64px; }
.production-header, .chapter-heading { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; }
.production-header { margin-bottom: 28px; }
.production-header h1, .chapter-heading h2 { margin: 4px 0 8px; }
.eyebrow { margin: 0; font-size: 12px; font-weight: 800; letter-spacing: .12em; text-transform: uppercase; opacity: .65; }
.workspace-grid { display: grid; grid-template-columns: minmax(220px, 280px) 1fr; gap: 24px; }
.chapter-panel, .pipeline-panel, .provenance-panel, .action-panel { border: 1px solid var(--border-color, #d8d8d8); border-radius: 16px; background: var(--surface, #fff); }
.chapter-panel { padding: 18px; height: fit-content; }
.chapter-button { width: 100%; text-align: left; display: grid; gap: 3px; padding: 12px; margin-top: 8px; border: 1px solid transparent; border-radius: 10px; background: transparent; cursor: pointer; }
.chapter-button.active { border-color: currentColor; }
.chapter-button span, .chapter-button small { opacity: .65; }
.pipeline-panel { padding: 22px; }
.action-panel { margin-top: 20px; padding: 16px; display: flex; gap: 16px; align-items: center; justify-content: space-between; }
.action-panel p { margin: 5px 0 0; max-width: 720px; opacity: .72; }
.action-panel button { white-space: nowrap; }
.stage-list { list-style: none; margin: 24px 0 0; padding: 0; display: grid; gap: 10px; }
.stage-card { display: grid; grid-template-columns: 12px 1fr; gap: 14px; padding: 14px; border: 1px solid var(--border-color, #ddd); border-radius: 12px; }
.stage-card p { margin: 4px 0 0; opacity: .72; }
.stage-state { width: 10px; height: 10px; margin-top: 5px; border-radius: 50%; background: #999; }
.stage-card[data-state='ready'] .stage-state { background: #1f9d55; }
.stage-card[data-state='waiting'] .stage-state { background: #d39b16; }
.stage-card[data-state='blocked'] .stage-state { background: #b64b4b; }
.provenance-panel { margin-top: 20px; padding: 18px; }
.provenance-panel dl { display: grid; gap: 8px; }
.provenance-panel dl div { display: grid; grid-template-columns: 140px 1fr; gap: 12px; }
.provenance-panel dt { font-weight: 700; }
.provenance-panel dd { margin: 0; overflow-wrap: anywhere; }
.status-state.success { color: #137c43; }
.muted { opacity: .65; }
@media (max-width: 800px) { .workspace-grid { grid-template-columns: 1fr; } .production-header, .chapter-heading, .action-panel { flex-direction: column; } }
</style>
