<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  approveContent,
  editContent,
  listAdminChapters,
  listChapterReviews,
  listContentRevisions,
  regenerateContent,
  rejectContent,
  runContentReview,
} from '../../api/client'
import type { Chapter, ChapterReview, ContentRevision } from '../../api/types'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const auth = useAuthStore()
const storyID = computed(() => route.params.storyID as string)

const chapters = ref<Chapter[]>([])
const activeChapter = ref<Chapter | null>(null)
const revisions = ref<ContentRevision[]>([])
const activeRevision = ref<ContentRevision | null>(null)
const reviews = ref<ChapterReview[]>([])
const editorText = ref('')
const rejectReason = ref('')

const loading = ref(false)
const action = ref('')
const error = ref('')
const success = ref('')
const showRejectForm = ref(false)

const activeReviews = computed(() =>
  reviews.value.filter((review) => review.ContentRevisionID === activeRevision.value?.ID),
)
const canAct = computed(() => Boolean(activeChapter.value && activeRevision.value && auth.user?.id))
const reviewTypes = ['CONTINUITY', 'QUALITY', 'SAFETY'] as const

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    CANDIDATE: 'Chờ duyệt',
    APPROVED: 'Đã duyệt',
    REJECTED: 'Đã từ chối',
    SUPERSEDED: 'Đã thay thế',
  }
  return labels[status] ?? status
}

function reviewTypeLabel(type: string) {
  const labels: Record<string, string> = {
    CONTINUITY: 'Tính liền mạch',
    QUALITY: 'Chất lượng viết',
    SAFETY: 'An toàn nội dung',
  }
  return labels[type] ?? type
}

function outcomeLabel(outcome: string) {
  return outcome === 'PASS' ? 'Đạt' : outcome === 'FAIL' ? 'Cần xem lại' : outcome
}

async function loadChapters() {
  const response = await listAdminChapters(storyID.value)
  chapters.value = response.chapters
  if (!activeChapter.value && chapters.value.length > 0) {
    await selectChapter(chapters.value[0])
  }
}

async function loadRevisionData(chapterID: string) {
  const [revisionResponse, reviewResponse] = await Promise.all([
    listContentRevisions(chapterID),
    listChapterReviews(chapterID),
  ])
  revisions.value = revisionResponse.revisions
  reviews.value = reviewResponse.reviews

  const nextRevision =
    activeRevision.value && revisions.value.some((revision) => revision.ID === activeRevision.value?.ID)
      ? revisions.value.find((revision) => revision.ID === activeRevision.value?.ID)
      : revisions.value[revisions.value.length - 1]
  selectRevision(nextRevision ?? null)
}

async function selectChapter(chapter: Chapter) {
  activeChapter.value = chapter
  activeRevision.value = null
  revisions.value = []
  reviews.value = []
  editorText.value = ''
  error.value = ''
  try {
    await loadRevisionData(chapter.ID)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải nội dung chương.'
  }
}

function selectRevision(revision: ContentRevision | null) {
  activeRevision.value = revision
  editorText.value = revision?.ContentText ?? ''
  showRejectForm.value = false
  rejectReason.value = ''
}

async function refreshRevisionData() {
  if (!activeChapter.value) return
  await loadRevisionData(activeChapter.value.ID)
}

async function saveEdit() {
  if (!activeChapter.value || !activeRevision.value || !auth.user) return
  action.value = 'edit'
  error.value = ''
  success.value = ''
  try {
    await editContent(activeChapter.value.ID, activeRevision.value.ID, editorText.value)
    success.value = 'Đã tạo revision mới từ bản chỉnh sửa.'
    await refreshRevisionData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể lưu bản chỉnh sửa.'
  } finally {
    action.value = ''
  }
}

async function regenerate() {
  if (!activeChapter.value || !activeRevision.value || !auth.user) return
  action.value = 'regenerate'
  error.value = ''
  success.value = ''
  try {
    await regenerateContent(activeChapter.value.ID, activeRevision.value.ID)
    success.value = 'Đã tạo yêu cầu sinh lại nội dung.'
    await refreshRevisionData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể sinh lại nội dung.'
  } finally {
    action.value = ''
  }
}

async function runReview(reviewType: 'CONTINUITY' | 'QUALITY' | 'SAFETY') {
  if (!activeChapter.value || !activeRevision.value) return
  action.value = reviewType
  error.value = ''
  success.value = ''
  try {
    await runContentReview(activeChapter.value.ID, reviewType, activeRevision.value.ID, editorText.value)
    success.value = `Đã chạy kiểm tra ${reviewTypeLabel(reviewType).toLowerCase()}.`
    await refreshRevisionData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể chạy kiểm tra nội dung.'
  } finally {
    action.value = ''
  }
}

async function approve() {
  if (!activeChapter.value || !activeRevision.value || !auth.user) return
  if (!window.confirm('Duyệt revision này để chuyển sang bước tiếp theo?')) return

  action.value = 'approve'
  error.value = ''
  success.value = ''
  try {
    await approveContent(activeChapter.value.ID, activeRevision.value.ID)
    success.value = 'Revision đã được duyệt.'
    await refreshRevisionData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể duyệt revision.'
  } finally {
    action.value = ''
  }
}

async function reject() {
  if (!activeRevision.value || !auth.user || !rejectReason.value.trim()) return
  action.value = 'reject'
  error.value = ''
  success.value = ''
  try {
    await rejectContent(activeRevision.value.ID, rejectReason.value)
    success.value = 'Revision đã được từ chối.'
    showRejectForm.value = false
    rejectReason.value = ''
    await refreshRevisionData()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể từ chối revision.'
  } finally {
    action.value = ''
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    await loadChapters()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải dữ liệu review.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="page content-review">
    <RouterLink class="back-link" :to="`/admin/stories/${storyID}/control`">← Về trung tâm điều khiển</RouterLink>

    <header class="review-heading">
      <p class="eyebrow">Studio / Content Review</p>
      <h1>Duyệt nội dung chương</h1>
      <p class="page-intro">
        Kiểm tra từng revision trước khi duyệt, tạo audio và xuất bản. Mọi thao tác được ghi nhận theo tài khoản hiện tại.
      </p>
    </header>

    <p v-if="loading" class="status-state" role="status" aria-live="polite">Đang tải workspace review...</p>
    <div v-else-if="error && chapters.length === 0" class="status-state error" role="alert">
      <strong>Không thể tải workspace review.</strong>
      <p>{{ error }}</p>
      <button class="secondary-link" type="button" @click="load">Thử lại</button>
    </div>

    <div v-else class="review-workspace">
      <aside class="review-sidebar" aria-labelledby="chapter-list-heading">
        <div class="panel-heading">
          <div>
            <p class="panel-kicker">Production queue</p>
            <h2 id="chapter-list-heading">Các chương</h2>
          </div>
          <span class="count-label">{{ chapters.length }}</span>
        </div>
        <p v-if="chapters.length === 0" class="note">Chưa có chương nào.</p>
        <div v-else class="review-chapter-list">
          <button
            v-for="chapter in chapters"
            :key="chapter.ID"
            class="review-chapter"
            :class="{ active: activeChapter?.ID === chapter.ID }"
            type="button"
            :aria-current="activeChapter?.ID === chapter.ID ? 'page' : undefined"
            @click="selectChapter(chapter)"
          >
            <span>Chương {{ chapter.ChapterNumber }}</span>
            <strong>{{ chapter.Title }}</strong>
            <small>{{ chapter.Status }}</small>
          </button>
        </div>
      </aside>

      <section class="review-main" aria-labelledby="revision-heading">
        <div v-if="!activeChapter" class="empty-state">
          <strong>Chọn một chương để bắt đầu.</strong>
          <span>Danh sách revision và công cụ review sẽ hiển thị tại đây.</span>
        </div>

        <template v-else>
          <div class="review-main-heading">
            <div>
              <p class="panel-kicker">Chương {{ activeChapter.ChapterNumber }}</p>
              <h2 id="revision-heading">{{ activeChapter.Title }}</h2>
            </div>
            <span class="badge">{{ revisions.length }} revision</span>
          </div>

          <div v-if="revisions.length === 0" class="empty-state">
            <strong>Chưa có nội dung để review.</strong>
            <span>Hãy chạy quy trình tạo nội dung trước.</span>
          </div>

          <template v-else-if="activeRevision">
            <div class="revision-picker" aria-label="Chọn revision">
              <button
                v-for="revision in revisions"
                :key="revision.ID"
                class="revision-chip"
                :class="{ active: revision.ID === activeRevision.ID }"
                type="button"
                @click="selectRevision(revision)"
              >
                <strong>v{{ revision.RevisionNo }}</strong>
                <span>{{ statusLabel(revision.Status) }}</span>
              </button>
            </div>

            <div class="revision-meta">
              <span>Nguồn: {{ activeRevision.SourceType }}</span>
              <span>Người tạo: {{ activeRevision.CreatedBy || 'Hệ thống' }}</span>
              <span class="badge">{{ statusLabel(activeRevision.Status) }}</span>
            </div>

            <label class="review-editor-label" for="content-editor">Nội dung đang review</label>
            <textarea
              id="content-editor"
              v-model="editorText"
              class="review-editor"
              rows="18"
              aria-describedby="content-editor-help"
            ></textarea>
            <p id="content-editor-help" class="field-help">
              Chỉnh sửa sẽ tạo một revision mới, không ghi đè lịch sử nội dung.
            </p>

            <div class="review-actions" aria-label="Thao tác nội dung">
              <button type="button" :disabled="!canAct || action !== '' || !editorText.trim()" @click="saveEdit">
                {{ action === 'edit' ? 'Đang lưu...' : 'Lưu bản chỉnh sửa' }}
              </button>
              <button class="secondary-button" type="button" :disabled="!canAct || action !== ''" @click="regenerate">
                {{ action === 'regenerate' ? 'Đang tạo...' : 'Sinh lại' }}
              </button>
            </div>

            <section class="review-checks" aria-labelledby="checks-heading">
              <div class="panel-heading">
                <div>
                  <p class="panel-kicker">Quality gates</p>
                  <h3 id="checks-heading">Kiểm tra nội dung</h3>
                </div>
              </div>
              <div class="check-actions">
                <button
                  v-for="reviewType in reviewTypes"
                  :key="reviewType"
                  class="secondary-button"
                  type="button"
                  :disabled="!canAct || action !== ''"
                  @click="runReview(reviewType)"
                >
                  {{ action === reviewType ? 'Đang kiểm tra...' : reviewTypeLabel(reviewType) }}
                </button>
              </div>
              <p v-if="activeReviews.length === 0" class="note">Chưa có kết quả kiểm tra cho revision này.</p>
              <ul v-else class="review-result-list">
                <li v-for="review in activeReviews" :key="review.ID">
                  <div>
                    <strong>{{ reviewTypeLabel(review.ReviewType) }}</strong>
                    <span class="badge" :class="{ 'badge-active': review.Outcome === 'PASS' }">
                      {{ outcomeLabel(review.Outcome) }}
                    </span>
                  </div>
                  <pre v-if="Object.keys(review.Report).length > 0">{{ JSON.stringify(review.Report, null, 2) }}</pre>
                </li>
              </ul>
            </section>

            <section class="review-decision" aria-labelledby="decision-heading">
              <div class="panel-heading">
                <div>
                  <p class="panel-kicker">Final decision</p>
                  <h3 id="decision-heading">Quyết định revision</h3>
                </div>
              </div>
              <div class="review-actions">
                <button type="button" :disabled="!canAct || action !== ''" @click="approve">
                  {{ action === 'approve' ? 'Đang duyệt...' : 'Duyệt revision' }}
                </button>
                <button
                  class="danger-button"
                  type="button"
                  :disabled="!canAct || action !== ''"
                  @click="showRejectForm = !showRejectForm"
                >
                  Từ chối
                </button>
              </div>
              <form v-if="showRejectForm" class="reject-form" @submit.prevent="reject">
                <label for="reject-reason">
                  Lý do từ chối
                  <textarea
                    id="reject-reason"
                    v-model="rejectReason"
                    rows="3"
                    required
                    placeholder="Nêu rõ điểm cần chỉnh sửa..."
                  ></textarea>
                </label>
                <button class="danger-button" type="submit" :disabled="action !== '' || !rejectReason.trim()">
                  {{ action === 'reject' ? 'Đang từ chối...' : 'Xác nhận từ chối' }}
                </button>
              </form>
            </section>

            <p v-if="error" class="status-state error" role="alert">{{ error }}</p>
            <p v-if="success" class="status-state success" role="status" aria-live="polite">{{ success }}</p>
          </template>
        </template>
      </section>
    </div>
  </section>
</template>
