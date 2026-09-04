<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  analyzeThreadInactivity,
  listAttentionItems,
  listCreativeDecisions,
  listUsage,
} from '../../api/client'
import type {
  AttentionItem,
  CreativeDecision,
  ThreadInactivity,
  UsageRecord,
} from '../../api/types'

const route = useRoute()
const storyID = route.params.storyID as string

const decisions = ref<CreativeDecision[]>([])
const attention = ref<AttentionItem[]>([])
const inactiveThreads = ref<ThreadInactivity[]>([])
const usage = ref<UsageRecord[]>([])

const loading = ref(false)
const error = ref('')

const summaryItems = computed(() => [
  { label: 'Cần chú ý', value: attention.value.length },
  { label: 'Quyết định mở', value: decisions.value.length },
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

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [d, a, t, u] = await Promise.all([
      listCreativeDecisions(storyID),
      listAttentionItems(storyID),
      analyzeThreadInactivity(storyID),
      listUsage(storyID),
    ])
    decisions.value = d.decisions
    attention.value = a.items
    inactiveThreads.value = t.inactive_threads
    usage.value = u.usage
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải dữ liệu điều khiển.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="page control-center">
    <RouterLink class="back-link" to="/admin">← Về quản lý truyện</RouterLink>
    <p class="eyebrow">Story Control Center</p>
    <h1>Trung tâm điều khiển truyện</h1>
    <p class="page-intro">Ưu tiên các điểm cần quyết định trước khi tiếp tục tạo nội dung.</p>
    <p>
      <RouterLink class="secondary-link" :to="`/admin/stories/${storyID}/production`">Mở Chapter Production Pipeline →</RouterLink>
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
            <p v-if="decisions.length === 0" class="note">Không có quyết định nào.</p>
            <ul v-else class="item-list">
              <li v-for="d in decisions" :key="d.ID" class="item-row">
                <strong>{{ d.Question }}</strong>
                <div class="item-meta">
                  <span>{{ d.Severity }}</span>
                  <span>{{ d.Status }}</span>
                  <span>{{ d.BlockingLevel }}</span>
                </div>
                <p v-if="d.ContextSummary" class="muted">{{ d.ContextSummary }}</p>
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
