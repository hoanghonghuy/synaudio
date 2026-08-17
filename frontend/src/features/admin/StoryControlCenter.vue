<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
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
    <p class="eyebrow">Admin</p>
    <h1>Trung tâm điều khiển truyện</h1>

    <p v-if="loading" class="note">Đang tải...</p>
    <p v-else-if="error" class="error">{{ error }}</p>

    <template v-else>
      <section class="panel">
        <h2>Quyết định sáng tạo</h2>
        <p v-if="decisions.length === 0" class="note">Không có quyết định nào.</p>
        <ul v-else class="item-list">
          <li v-for="d in decisions" :key="d.ID" class="item-row">
            <div>
              <strong>{{ d.Question }}</strong>
              <span class="badge">{{ d.Severity }}</span>
              <span class="badge">{{ d.Status }}</span>
            </div>
          </li>
        </ul>
      </section>

      <section class="panel">
        <h2>Cần chú ý</h2>
        <p v-if="attention.length === 0" class="note">Không có mục cần chú ý.</p>
        <ul v-else class="item-list">
          <li v-for="a in attention" :key="a.ID" class="item-row">
            <div>
              <strong>{{ a.Title }}</strong>
              <span class="badge">{{ a.Priority }}</span>
              <span v-if="a.Resolved" class="badge">Đã xử lý</span>
            </div>
            <p v-if="a.Detail" class="muted">{{ a.Detail }}</p>
          </li>
        </ul>
      </section>

      <section class="panel">
        <h2>Mạch truyện ít hoạt động</h2>
        <p v-if="inactiveThreads.length === 0" class="note">Không có mạch truyện nào bị bỏ quên.</p>
        <ul v-else class="item-list">
          <li v-for="t in inactiveThreads" :key="t.ThreadID" class="item-row">
            <div>
              <strong>{{ t.Title }}</strong>
              <span class="badge">{{ t.Importance }}</span>
              <span class="muted">{{ t.EventCount }} sự kiện</span>
            </div>
          </li>
        </ul>
      </section>

      <section class="panel">
        <h2>Chi phí / sử dụng</h2>
        <p v-if="usage.length === 0" class="note">Chưa có dữ liệu sử dụng.</p>
        <ul v-else class="item-list">
          <li v-for="u in usage" :key="u.ID" class="item-row">
            <div>
              <strong>{{ u.Provider }} / {{ u.Model }}</strong>
              <span class="badge">{{ u.Status }}</span>
              <span class="muted">{{ u.LatencyMs }}ms</span>
            </div>
          </li>
        </ul>
      </section>
    </template>
  </section>
</template>
