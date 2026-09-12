<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  analyzeThreadInactivity,
  listAttentionItems,
  listCreativeDecisions,
  listUsage,
} from '../../api/client'
import {
  postponeCreativeDecision,
  rejectCreativeDecision,
  selectCreativeDecision,
} from '../../api/creative-decisions'
import type {
  AttentionItem,
  CreativeDecision,
  ThreadInactivity,
  UsageRecord,
} from '../../api/types'

const route = useRoute()
const storyID = computed(() => String(route.params.storyID ?? ''))

const decisions = ref<CreativeDecision[]>([])
const attention = ref<AttentionItem[]>([])
const inactiveThreads = ref<ThreadInactivity[]>([])
const usage = ref<UsageRecord[]>([])

const loading = ref(false)
const error = ref('')
const mutationError = ref('')
const mutationMessage = ref('')
const pendingDecisionID = ref('')
const decisionForm = ref<{ id: string; action: 'reject' | 'postpone' } | null>(null)
const decisionNotes = ref<Record<string, string>>({})
let loadGeneration = 0
let decisionRefreshGeneration = 0
let mutationGeneration = 0

const proposedDecisionCount = computed(() => decisions.value.filter((decision) => decision.Status === 'PROPOSED').length)
const summaryItems = computed(() => [
  { label: 'Cần chú ý', value: attention.value.length },
  { label: 'Quyết định mở', value: proposedDecisionCount.value },
  { label: 'Mạch ít hoạt động', value: inactiveThreads.value.length },
  { label: 'Lần gọi gần đây', value: usage.value.length },
])

function priorityLabel(priority: string) {
  const labels: Record<string, string> = {
    HIGH: 'Ưu tiên cao',
    MEDIUM: 'Ưu tiên vừa',
    LOW: 'Ưu tiên thấp',
  }
  return labels[priority] ?? priority
}

function decisionStatusLabel(status: string) {
  const labels: Record<string, string> = {
    PROPOSED: 'Chờ quyết định',
    SELECTED: 'Đã chọn',
    REJECTED: 'Đã từ chối',
    POSTPONED: 'Đã hoãn',
  }
  return labels[status] ?? status
}

async function refreshDecisions(expectedStoryID = storyID.value) {
  const generation = ++decisionRefreshGeneration
  const response = await listCreativeDecisions(expectedStoryID)
  if (generation !== decisionRefreshGeneration || expectedStoryID !== storyID.value) return
  decisions.value = response.decisions
}

async function load() {
  const expectedStoryID = storyID.value
  const generation = ++loadGeneration
  ++decisionRefreshGeneration
  loading.value = true
  error.value = ''
  mutationError.value = ''
  mutationMessage.value = ''

  try {
    const [d, a, t, u] = await Promise.all([
      listCreativeDecisions(expectedStoryID),
      listAttentionItems(expectedStoryID),
      analyzeThreadInactivity(expectedStoryID),
      listUsage(expectedStoryID),
    ])
    if (generation !== loadGeneration || expectedStoryID !== storyID.value) return
    decisions.value = d.decisions
    attention.value = a.items
    inactiveThreads.value = t.inactive_threads
    usage.value = u.usage
  } catch (e) {
    if (generation !== loadGeneration || expectedStoryID !== storyID.value) return
    error.value = e instanceof Error ? e.message : 'Không thể tải dữ liệu điều khiển.'
  } finally {
    if (generation === loadGeneration) loading.value = false
  }
}

async function runDecisionMutation(decision: CreativeDecision, action: 'select' | 'reject' | 'postpone') {
  if (pendingDecisionID.value || decision.Status !== 'PROPOSED') return

  const expectedStoryID = storyID.value
  const generation = ++mutationGeneration
  const note = decisionNotes.value[decision.ID]?.trim() ?? ''
  if (action !== 'select' && !note) {
    mutationError.value = action === 'reject'
      ? 'Hãy nhập phạm vi/lý do từ chối trước khi xác nhận.'
      : 'Hãy nhập lý do hoãn trước khi xác nhận.'
    return
  }

  pendingDecisionID.value = decision.ID
  mutationError.value = ''
  mutationMessage.value = ''

  try {
    if (action === 'select') {
      await selectCreativeDecision(decision.ID)
    } else if (action === 'reject') {
      await rejectCreativeDecision(decision.ID, note)
    } else {
      await postponeCreativeDecision(decision.ID, note)
    }

    if (generation !== mutationGeneration || expectedStoryID !== storyID.value) return
    const messages = {
      select: 'Đã chọn quyết định. Đang đồng bộ trạng thái từ máy chủ.',
      reject: 'Đã từ chối quyết định. Đang đồng bộ trạng thái từ máy chủ.',
      postpone: 'Đã hoãn quyết định. Đang đồng bộ trạng thái từ máy chủ.',
    }
    mutationMessage.value = messages[action]
    decisionForm.value = null
    await refreshDecisions(expectedStoryID)
  } catch (e) {
    if (generation !== mutationGeneration || expectedStoryID !== storyID.value) return
    mutationError.value = e instanceof Error ? e.message : 'Không thể cập nhật quyết định.'
    try {
      await refreshDecisions(expectedStoryID)
    } catch {
      // Preserve the mutation error; the explicit page retry remains available.
    }
  } finally {
    if (generation === mutationGeneration) pendingDecisionID.value = ''
  }
}

function beginDecisionForm(decisionID: string, action: 'reject' | 'postpone') {
  mutationError.value = ''
  mutationMessage.value = ''
  decisionForm.value = { id: decisionID, action }
}

function cancelDecisionForm() {
  decisionForm.value = null
  mutationError.value = ''
}

watch(storyID, () => {
  ++mutationGeneration
  pendingDecisionID.value = ''
  decisionForm.value = null
  decisionNotes.value = {}
  void load()
}, { immediate: true })
</script>

<template>
  <section class="page control-center">
    <RouterLink class="back-link" to="/admin">← Về quản lý truyện</RouterLink>
    <p class="eyebrow">Story Control Center</p>
    <h1>Trung tâm điều khiển truyện</h1>
    <p class="page-intro">Ưu tiên các điểm cần quyết định trước khi tiếp tục tạo nội dung.</p>
    <p>
      <RouterLink class="secondary-link" :to="`/admin/stories/${storyID}/production`">Mở Chapter Production →</RouterLink>
      <span aria-hidden="true"> · </span>
      <RouterLink class="secondary-link" :to="`/admin/stories/${storyID}/retcons`">Mở Retcon Governance →</RouterLink>
    </p>

    <p v-if="loading" class="status-state" role="status" aria-live="polite">Đang tải dữ liệu điều khiển...</p>
    <div v-else-if="error" class="status-state error" role="alert">
      <strong>Không thể tải dữ liệu điều khiển.</strong>
      <p>{{ error }}</p>
      <button class="secondary-link" type="button" @click="load">Thử lại</button>
    </div>

    <template v-else>
      <section class="control-summary" aria-label="Tóm tắt vận hành">
        <div v-for="item in summaryItems" :key="item.label" class="summary-item">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
        </div>
      </section>

      <div class="control-workspace">
        <div class="control-main-column">
          <section class="panel panel-attention" aria-labelledby="attention-heading">
            <div class="panel-heading">
              <div>
                <p class="panel-kicker">Ưu tiên xử lý</p>
                <h2 id="attention-heading">Cần chú ý</h2>
              </div>
              <span class="count-label">{{ attention.length }}</span>
            </div>
            <p v-if="attention.length === 0" class="note">Không có mục cần chú ý.</p>
            <ul v-else class="item-list">
              <li v-for="a in attention" :key="a.ID" class="item-row">
                <div class="item-title-line">
                  <strong>{{ a.Title }}</strong>
                  <span class="badge badge-attention">{{ priorityLabel(a.Priority) }}</span>
                </div>
                <p v-if="a.Detail" class="muted">{{ a.Detail }}</p>
                <div class="item-meta">
                  <span v-if="a.Kind">{{ a.Kind }}</span>
                  <span v-if="a.Action">{{ a.Action }}</span>
                  <span v-if="a.Resolved">Đã xử lý</span>
                </div>
              </li>
            </ul>
          </section>

          <section class="panel" aria-labelledby="decisions-heading">
            <div class="panel-heading">
              <div>
                <p class="panel-kicker">Câu hỏi cần chốt</p>
                <h2 id="decisions-heading">Quyết định sáng tạo</h2>
              </div>
              <span class="count-label">{{ decisions.length }}</span>
            </div>
            <div class="decision-feedback" aria-live="polite" aria-atomic="true">
              <p v-if="mutationMessage" class="note success-note">{{ mutationMessage }}</p>
              <p v-if="mutationError" class="note error" role="alert">{{ mutationError }}</p>
            </div>
            <p v-if="decisions.length === 0" class="note">Không có quyết định nào.</p>
            <ul v-else class="item-list decision-list">
              <li v-for="d in decisions" :key="d.ID" class="item-row decision-row">
                <div class="decision-heading-row">
                  <strong class="decision-question">{{ d.Question }}</strong>
                  <span class="badge" :class="`decision-status decision-status-${d.Status.toLowerCase()}`">
                    {{ decisionStatusLabel(d.Status) }}
                  </span>
                </div>
                <div class="item-meta">
                  <span>{{ d.Severity }}</span>
                  <span>{{ d.BlockingLevel }}</span>
                </div>
                <p v-if="d.ContextSummary" class="muted decision-context">{{ d.ContextSummary }}</p>

                <div v-if="d.Status === 'PROPOSED'" class="decision-actions" :aria-busy="pendingDecisionID === d.ID">
                  <button
                    class="decision-button"
                    type="button"
                    :disabled="Boolean(pendingDecisionID)"
                    @click="runDecisionMutation(d, 'select')"
                  >
                    {{ pendingDecisionID === d.ID ? 'Đang xử lý…' : 'Chọn quyết định' }}
                  </button>
                  <button
                    class="decision-button"
                    type="button"
                    :disabled="Boolean(pendingDecisionID)"
                    :aria-expanded="decisionForm?.id === d.ID && decisionForm.action === 'postpone'"
                    @click="beginDecisionForm(d.ID, 'postpone')"
                  >
                    Hoãn…
                  </button>
                  <button
                    class="decision-button decision-button-danger"
                    type="button"
                    :disabled="Boolean(pendingDecisionID)"
                    :aria-expanded="decisionForm?.id === d.ID && decisionForm.action === 'reject'"
                    @click="beginDecisionForm(d.ID, 'reject')"
                  >
                    Từ chối…
                  </button>
                </div>

                <form
                  v-if="d.Status === 'PROPOSED' && decisionForm?.id === d.ID"
                  class="decision-form"
                  @submit.prevent="runDecisionMutation(d, decisionForm.action)"
                >
                  <label :for="`decision-note-${d.ID}`">
                    {{ decisionForm.action === 'reject' ? 'Phạm vi / lý do từ chối' : 'Lý do hoãn' }}
                  </label>
                  <textarea
                    :id="`decision-note-${d.ID}`"
                    v-model="decisionNotes[d.ID]"
                    rows="3"
                    maxlength="500"
                    required
                    :disabled="Boolean(pendingDecisionID)"
                    :placeholder="decisionForm.action === 'reject'
                      ? 'Mô tả rõ quyết định này bị từ chối vì sao hoặc phạm vi cần tránh.'
                      : 'Nêu rõ vì sao quyết định này nên được hoãn để người vận hành có ngữ cảnh khi xem lại.'"
                  />
                  <div class="decision-form-actions">
                    <button
                      class="decision-button"
                      :class="{ 'decision-button-danger': decisionForm.action === 'reject' }"
                      type="submit"
                      :disabled="Boolean(pendingDecisionID) || !decisionNotes[d.ID]?.trim()"
                    >
                      {{ pendingDecisionID === d.ID
                        ? 'Đang xử lý…'
                        : decisionForm.action === 'reject' ? 'Xác nhận từ chối' : 'Xác nhận hoãn' }}
                    </button>
                    <button class="decision-button" type="button" :disabled="Boolean(pendingDecisionID)" @click="cancelDecisionForm">
                      Hủy
                    </button>
                  </div>
                </form>

                <p v-else-if="d.Status === 'SELECTED' || d.Status === 'REJECTED' || d.Status === 'POSTPONED'" class="terminal-note">
                  Trạng thái cuối — không thể mở lại từ màn hình này.
                </p>
              </li>
            </ul>
          </section>
        </div>

        <div class="control-side-column">
          <section class="panel" aria-labelledby="threads-heading">
            <div class="panel-heading">
              <div>
                <p class="panel-kicker">Theo dõi mạch truyện</p>
                <h2 id="threads-heading">Mạch truyện ít hoạt động</h2>
              </div>
              <span class="count-label">{{ inactiveThreads.length }}</span>
            </div>
            <p v-if="inactiveThreads.length === 0" class="note">Không có mạch truyện nào bị bỏ quên.</p>
            <ul v-else class="item-list">
              <li v-for="t in inactiveThreads" :key="t.ThreadID" class="item-row">
                <strong>{{ t.Title }}</strong>
                <div class="item-meta">
                  <span>{{ t.Importance }}</span>
                  <span>{{ t.EventCount }} sự kiện</span>
                </div>
              </li>
            </ul>
          </section>

          <section class="panel" aria-labelledby="usage-heading">
            <div class="panel-heading">
              <div>
                <p class="panel-kicker">Provider activity</p>
                <h2 id="usage-heading">Chi phí / sử dụng</h2>
              </div>
              <span class="count-label">{{ usage.length }}</span>
            </div>
            <p v-if="usage.length === 0" class="note">Chưa có dữ liệu sử dụng.</p>
            <ul v-else class="item-list">
              <li v-for="u in usage" :key="u.ID" class="item-row">
                <strong>{{ u.Provider }} / {{ u.Model }}</strong>
                <div class="item-meta">
                  <span>{{ u.Status }}</span>
                  <span>{{ u.LatencyMs }}ms</span>
                  <span>Lần {{ u.AttemptNo }}</span>
                </div>
              </li>
            </ul>
          </section>
        </div>
      </div>
    </template>
  </section>
</template>

<style scoped>
.decision-list,
.decision-row {
  min-width: 0;
}

.decision-heading-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.decision-question,
.decision-context {
  min-width: 0;
  overflow-wrap: anywhere;
}

.decision-status {
  flex: 0 0 auto;
}

.decision-status-selected {
  opacity: 0.9;
}

.decision-status-rejected,
.decision-status-postponed {
  opacity: 0.75;
}

.decision-actions,
.decision-form-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 1rem;
}

.decision-button {
  min-height: 44px;
  min-width: 44px;
  padding: 0.65rem 0.9rem;
  border: 1px solid currentColor;
  border-radius: 0.6rem;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.decision-button:hover:not(:disabled) {
  filter: brightness(1.08);
}

.decision-button:focus-visible,
.decision-form textarea:focus-visible {
  outline: 3px solid currentColor;
  outline-offset: 2px;
}

.decision-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.decision-button-danger {
  font-weight: 650;
}

.decision-form {
  display: grid;
  gap: 0.5rem;
  margin-top: 1rem;
  padding: 0.9rem;
  border: 1px solid currentColor;
  border-radius: 0.75rem;
}

.decision-form label {
  font-weight: 650;
}

.decision-form textarea {
  width: 100%;
  min-height: 5.5rem;
  resize: vertical;
  box-sizing: border-box;
  padding: 0.75rem;
  border-radius: 0.5rem;
  font: inherit;
}

.terminal-note,
.success-note {
  margin-top: 0.75rem;
}

@media (max-width: 560px) {
  .decision-heading-row {
    flex-direction: column;
  }

  .decision-actions,
  .decision-form-actions {
    display: grid;
    grid-template-columns: 1fr;
  }

  .decision-button {
    width: 100%;
  }
}
</style>