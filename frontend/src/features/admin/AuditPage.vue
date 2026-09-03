<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { getAuditEvent, listAuditEvents, type AuditFilters } from '../../api/client'
import type { AuditEvent } from '../../api/types'

const events = ref<AuditEvent[]>([])
const selected = ref<AuditEvent | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const error = ref('')

const filters = ref({
  actor_id: '',
  action: '',
  resource_type: '',
  resource_id: '',
  story_id: '',
  chapter_id: '',
  run_id: '',
  correlation_id: '',
  result: '',
  from: '',
  to: '',
  limit: 100,
})

const hasFilters = computed(() =>
  Object.entries(filters.value).some(([key, value]) => key !== 'limit' && String(value).trim() !== ''),
)

function toRFC3339(value: string): string | undefined {
  if (!value) return undefined
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return undefined
  return parsed.toISOString()
}

function buildFilters(): AuditFilters {
  return {
    actor_id: filters.value.actor_id || undefined,
    action: filters.value.action || undefined,
    resource_type: filters.value.resource_type || undefined,
    resource_id: filters.value.resource_id || undefined,
    story_id: filters.value.story_id || undefined,
    chapter_id: filters.value.chapter_id || undefined,
    run_id: filters.value.run_id || undefined,
    correlation_id: filters.value.correlation_id || undefined,
    result: filters.value.result || undefined,
    from: toRFC3339(filters.value.from),
    to: toRFC3339(filters.value.to),
    limit: filters.value.limit,
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    events.value = (await listAuditEvents(buildFilters())).items
    if (selected.value && !events.value.some((event) => event.id === selected.value?.id)) {
      selected.value = null
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải audit trail.'
  } finally {
    loading.value = false
  }
}

async function selectEvent(event: AuditEvent) {
  detailLoading.value = true
  error.value = ''
  try {
    selected.value = await getAuditEvent(event.id)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải chi tiết audit event.'
  } finally {
    detailLoading.value = false
  }
}

function resetFilters() {
  filters.value = {
    actor_id: '',
    action: '',
    resource_type: '',
    resource_id: '',
    story_id: '',
    chapter_id: '',
    run_id: '',
    correlation_id: '',
    result: '',
    from: '',
    to: '',
    limit: 100,
  }
  void load()
}

function formatTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function formatJSON(value: Record<string, unknown> | undefined) {
  return JSON.stringify(value ?? {}, null, 2)
}

onMounted(load)
</script>

<template>
  <section class="page admin">
    <RouterLink class="back-link" to="/admin">← Về Studio</RouterLink>
    <p class="eyebrow">Studio / Governance</p>
    <h1>Audit & Provenance</h1>
    <p class="page-intro">
      Lịch sử append-only cho các thay đổi quan trọng, security actions và generation provenance. Audit chỉ đọc; không có thao tác sửa hoặc xóa lịch sử.
    </p>

    <form class="create-form" aria-label="Bộ lọc audit" @submit.prevent="load">
      <div class="section-heading">
        <div>
          <h2>Lọc sự kiện</h2>
          <p class="muted">Tìm theo actor, action, resource, story/chapter, generation run, correlation, kết quả hoặc khoảng thời gian.</p>
        </div>
        <button v-if="hasFilters" class="secondary-button" type="button" @click="resetFilters">Xóa bộ lọc</button>
      </div>

      <div class="row">
        <label>
          Action
          <input v-model.trim="filters.action" placeholder="CHAPTER_APPROVED" />
        </label>
        <label>
          Kết quả
          <select v-model="filters.result">
            <option value="">Tất cả</option>
            <option value="SUCCEEDED">Succeeded</option>
            <option value="FAILED">Failed</option>
            <option value="DENIED">Denied</option>
          </select>
        </label>
        <label>
          Actor user ID
          <input v-model.trim="filters.actor_id" placeholder="UUID" />
        </label>
      </div>

      <div class="row">
        <label>
          Resource type
          <input v-model.trim="filters.resource_type" placeholder="STORY / CONTENT_REVISION" />
        </label>
        <label>
          Resource ID
          <input v-model.trim="filters.resource_id" />
        </label>
        <label>
          Story ID
          <input v-model.trim="filters.story_id" placeholder="UUID" />
        </label>
        <label>
          Chapter ID
          <input v-model.trim="filters.chapter_id" placeholder="UUID" />
        </label>
      </div>

      <div class="row">
        <label>
          Generation run ID
          <input v-model.trim="filters.run_id" placeholder="UUID" />
        </label>
        <label>
          Correlation ID
          <input v-model.trim="filters.correlation_id" placeholder="request / trace correlation" />
        </label>
        <label>
          Từ thời điểm
          <input v-model="filters.from" type="datetime-local" />
        </label>
        <label>
          Đến thời điểm
          <input v-model="filters.to" type="datetime-local" />
        </label>
        <label>
          Giới hạn
          <input v-model.number="filters.limit" type="number" min="1" max="500" />
        </label>
      </div>

      <button type="submit" :disabled="loading">{{ loading ? 'Đang tải...' : 'Áp dụng bộ lọc' }}</button>
    </form>

    <p v-if="error" class="status-state error" role="alert">{{ error }}</p>

    <div class="admin-workspace">
      <section class="admin-list-panel" aria-labelledby="audit-list-heading">
        <div class="section-heading">
          <div>
            <h2 id="audit-list-heading">Audit trail</h2>
            <p class="muted">Mới nhất trước. Chọn một event để xem provenance và correlation context.</p>
          </div>
          <span class="count-label">{{ events.length }} event</span>
        </div>

        <p v-if="loading" class="status-state">Đang tải audit trail...</p>
        <p v-else-if="events.length === 0" class="empty-state">Không có event phù hợp với bộ lọc hiện tại.</p>

        <div v-else class="story-table-wrap">
          <table class="story-table">
            <thead>
              <tr>
                <th>Thời gian</th>
                <th>Action</th>
                <th>Actor</th>
                <th>Resource</th>
                <th>Kết quả</th>
                <th><span class="sr-only">Chi tiết</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="event in events" :key="event.id">
                <td>{{ formatTime(event.created_at) }}</td>
                <td><strong>{{ event.action }}</strong></td>
                <td>
                  <span class="badge">{{ event.actor_type }}</span>
                  <span v-if="event.actor_user_id" class="slug">{{ event.actor_user_id }}</span>
                </td>
                <td>
                  <span>{{ event.resource_type || '—' }}</span>
                  <span v-if="event.resource_id" class="slug">{{ event.resource_id }}</span>
                </td>
                <td><span class="badge">{{ event.result }}</span></td>
                <td><button class="secondary-button" type="button" @click="selectEvent(event)">Xem</button></td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <aside class="create-form" aria-labelledby="audit-detail-heading">
        <div>
          <p class="eyebrow">Investigation</p>
          <h2 id="audit-detail-heading">Chi tiết event</h2>
        </div>
        <p v-if="detailLoading" class="status-state">Đang tải chi tiết...</p>
        <p v-else-if="!selected" class="empty-state">Chọn một audit event để xem traceability.</p>
        <template v-else>
          <dl>
            <dt>Action</dt>
            <dd>{{ selected.action }}</dd>
            <dt>Result</dt>
            <dd>{{ selected.result }}</dd>
            <dt>Actor</dt>
            <dd>{{ selected.actor_type }} {{ selected.actor_user_id || '' }}</dd>
            <dt>Resource</dt>
            <dd>{{ selected.resource_type || '—' }} {{ selected.resource_id || '' }}</dd>
            <dt>Story / Chapter</dt>
            <dd>{{ selected.story_id || '—' }} / {{ selected.chapter_id || '—' }}</dd>
            <dt>Request</dt>
            <dd>{{ selected.request_id || '—' }}</dd>
            <dt>Correlation</dt>
            <dd>{{ selected.correlation_id || '—' }}</dd>
            <dt>Generation run</dt>
            <dd>{{ selected.generation_run_id || '—' }}</dd>
          </dl>

          <div v-if="selected.story_id" class="story-row-actions">
            <RouterLink class="control-link" :to="`/admin/stories/${selected.story_id}/control`">Mở story</RouterLink>
            <RouterLink class="control-link" :to="`/admin/stories/${selected.story_id}/review`">Duyệt nội dung</RouterLink>
          </div>

          <div>
            <h3>Provenance</h3>
            <pre class="status-state">{{ formatJSON(selected.provenance) }}</pre>
          </div>
          <div>
            <h3>Metadata</h3>
            <pre class="status-state">{{ formatJSON(selected.metadata) }}</pre>
          </div>
        </template>
      </aside>
    </div>
  </section>
</template>
