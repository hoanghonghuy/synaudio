<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { ApiRequestError } from '../../api/http-error'
import {
  analyzeRetcon,
  applyRetcon,
  approveRetcon,
  cancelRetcon,
  getRetcon,
  listRetcons,
  markRetconReady,
  type RetconRequest,
} from '../../api/retcon'
import {
  retconActionIsDestructive,
  retconActionLabel,
  retconActions,
  retconStatusLabel,
  type RetconAction,
} from './retconPresentation.mjs'

const route = useRoute()
const storyID = computed(() => String(route.params.storyID ?? ''))
const items = ref<RetconRequest[]>([])
const selectedID = ref('')
const selected = ref<RetconRequest | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const busyAction = ref<RetconAction | null>(null)
const error = ref('')
const success = ref('')

const actions = computed(() => selected.value ? retconActions(selected.value.Status) : [])

function messageFor(error: unknown): string {
  if (!(error instanceof ApiRequestError)) return error instanceof Error ? error.message : 'Đã xảy ra lỗi không xác định.'
  switch (error.code) {
    case 'RETCON_INVALID_TRANSITION': return 'Trạng thái yêu cầu đã thay đổi hoặc thao tác này không còn hợp lệ. Dữ liệu sẽ được tải lại từ máy chủ.'
    case 'RETCON_NOT_READY': return 'Yêu cầu chưa sẵn sàng để áp dụng. Hãy kiểm tra lại trạng thái authoritative.'
    case 'RECENT_AUTH_REQUIRED': return 'Thao tác này yêu cầu xác thực gần đây. Hãy xác thực lại tài khoản rồi thử lại.'
    case 'MFA_REQUIRED': return 'Thao tác này yêu cầu MFA. Hãy hoàn tất xác minh bảo mật rồi thử lại.'
    case 'FORBIDDEN': return 'Bạn không có quyền thực hiện thao tác này.'
    case 'UNAUTHENTICATED':
    case 'ADMIN_ACTOR_REQUIRED': return 'Phiên Admin không còn đủ authority. Hãy đăng nhập hoặc xác thực lại.'
    default: return error.message
  }
}

async function loadList(preferredID = selectedID.value) {
  loading.value = true
  error.value = ''
  try {
    const response = await listRetcons(storyID.value)
    items.value = response.retcons
    if (preferredID && items.value.some((item) => item.ID === preferredID)) selectedID.value = preferredID
    else selectedID.value = items.value[0]?.ID ?? ''
    await loadDetail()
  } catch (e) {
    error.value = messageFor(e)
    selected.value = null
  } finally {
    loading.value = false
  }
}

async function loadDetail() {
  if (!selectedID.value) {
    selected.value = null
    return
  }
  detailLoading.value = true
  error.value = ''
  try {
    selected.value = await getRetcon(selectedID.value)
  } catch (e) {
    error.value = messageFor(e)
    selected.value = null
  } finally {
    detailLoading.value = false
  }
}

watch(selectedID, () => { void loadDetail() })

async function perform(action: RetconAction) {
  if (!selected.value || busyAction.value) return
  if (retconActionIsDestructive(action)) {
    const prompt = action === 'apply'
      ? 'Áp dụng Retcon này có thể thay đổi lịch sử đã publish. Tiếp tục?'
      : 'Hủy yêu cầu Retcon này?'
    if (!window.confirm(prompt)) return
  }

  const id = selected.value.ID
  busyAction.value = action
  error.value = ''
  success.value = ''
  try {
    if (action === 'analyze') await analyzeRetcon(id)
    else if (action === 'approve') await approveRetcon(id)
    else if (action === 'ready') await markRetconReady(id)
    else if (action === 'apply') await applyRetcon(id)
    else await cancelRetcon(id)
    await loadList(id)
    success.value = `${retconActionLabel(action)} thành công. Trạng thái đã được tải lại từ authority.`
  } catch (e) {
    error.value = messageFor(e)
    await loadList(id)
  } finally {
    busyAction.value = null
  }
}

onMounted(() => { void loadList() })
</script>

<template>
  <section class="page retcon-workbench">
    <RouterLink class="back-link" :to="`/admin/stories/${storyID}/control`">← Về Story Control Center</RouterLink>
    <p class="eyebrow">Retcon Governance</p>
    <h1>Workbench thay đổi lịch sử</h1>
    <p class="page-intro">Theo dõi và xử lý Retcon theo lifecycle do backend kiểm soát. Workbench không chỉnh trực tiếp Official Canon.</p>

    <div v-if="error" class="retcon-feedback error" role="alert">
      <strong>Không thể hoàn tất thao tác.</strong>
      <span>{{ error }}</span>
      <button type="button" @click="loadList(selectedID)">Tải lại</button>
    </div>
    <div v-if="success" class="retcon-feedback success" role="status" aria-live="polite">{{ success }}</div>

    <p v-if="loading" class="status-state" role="status" aria-live="polite">Đang tải Retcon authoritative...</p>
    <div v-else-if="items.length === 0" class="retcon-empty">
      <h2>Chưa có yêu cầu Retcon</h2>
      <p>Không có thay đổi lịch sử nào cần quản trị cho truyện này.</p>
    </div>

    <div v-else class="retcon-layout">
      <aside class="retcon-list" aria-label="Danh sách yêu cầu Retcon">
        <button
          v-for="item in items"
          :key="item.ID"
          type="button"
          class="retcon-list-item"
          :class="{ active: selectedID === item.ID }"
          :aria-pressed="selectedID === item.ID"
          @click="selectedID = item.ID"
        >
          <span class="retcon-list-title">{{ item.Reason || 'Retcon không có tiêu đề' }}</span>
          <span class="retcon-list-meta">{{ retconStatusLabel(item.Status) }} · {{ item.ImpactScope }}</span>
          <span class="retcon-id">{{ item.ID }}</span>
        </button>
      </aside>

      <section class="retcon-detail" aria-live="polite">
        <p v-if="detailLoading" class="status-state">Đang tải chi tiết...</p>
        <template v-else-if="selected">
          <div class="retcon-detail-header">
            <div>
              <p class="panel-kicker">Authoritative request</p>
              <h2>{{ selected.Reason }}</h2>
            </div>
            <span class="retcon-status">{{ retconStatusLabel(selected.Status) }}</span>
          </div>

          <dl class="retcon-fields">
            <div><dt>Proposed change</dt><dd>{{ selected.ProposedChange }}</dd></div>
            <div><dt>Target chapter</dt><dd class="retcon-id">{{ selected.TargetChapterID || 'Toàn truyện / chưa chỉ định' }}</dd></div>
            <div><dt>Impact scope</dt><dd>{{ selected.ImpactScope }}</dd></div>
            <div><dt>Requested by</dt><dd class="retcon-id">{{ selected.RequestedBy || 'Backend chưa cung cấp' }}</dd></div>
            <div><dt>Approved by</dt><dd class="retcon-id">{{ selected.ApprovedBy || '—' }}</dd></div>
            <div><dt>Applied by</dt><dd class="retcon-id">{{ selected.AppliedBy || '—' }}</dd></div>
          </dl>

          <div class="retcon-actions" aria-label="Retcon lifecycle actions">
            <button
              v-for="action in actions"
              :key="action"
              type="button"
              :class="{ destructive: retconActionIsDestructive(action) }"
              :disabled="busyAction !== null"
              @click="perform(action)"
            >
              {{ busyAction === action ? 'Đang xử lý…' : retconActionLabel(action) }}
            </button>
            <p v-if="actions.length === 0" class="note">Yêu cầu đã ở trạng thái terminal; không còn action hợp lệ.</p>
          </div>
        </template>
      </section>
    </div>
  </section>
</template>

<style scoped>
.retcon-workbench { max-width: 1180px; margin: 0 auto; }
.retcon-layout { display: grid; grid-template-columns: minmax(250px, 0.8fr) minmax(0, 1.6fr); gap: 1rem; align-items: start; }
.retcon-list, .retcon-detail, .retcon-empty, .retcon-feedback { border: 1px solid var(--border, #d7dbe2); border-radius: 14px; background: var(--surface, #fff); }
.retcon-list { display: grid; gap: .5rem; padding: .65rem; max-height: 72vh; overflow: auto; }
.retcon-list-item { min-height: 72px; width: 100%; text-align: left; padding: .8rem; border: 1px solid transparent; border-radius: 10px; background: transparent; cursor: pointer; }
.retcon-list-item:hover, .retcon-list-item.active { border-color: currentColor; }
.retcon-list-item:focus-visible, .retcon-actions button:focus-visible, .retcon-feedback button:focus-visible { outline: 3px solid currentColor; outline-offset: 2px; }
.retcon-list-title { display: block; font-weight: 700; overflow-wrap: anywhere; }
.retcon-list-meta, .retcon-id { display: block; margin-top: .25rem; font-size: .85rem; opacity: .72; overflow-wrap: anywhere; word-break: break-word; }
.retcon-detail { padding: 1rem; min-width: 0; }
.retcon-detail-header { display: flex; gap: 1rem; align-items: flex-start; justify-content: space-between; }
.retcon-status { flex: none; padding: .4rem .7rem; border: 1px solid currentColor; border-radius: 999px; font-size: .82rem; font-weight: 700; }
.retcon-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .75rem; margin: 1rem 0; }
.retcon-fields > div { min-width: 0; padding: .8rem; border-radius: 10px; background: color-mix(in srgb, currentColor 5%, transparent); }
.retcon-fields dt { font-size: .78rem; font-weight: 700; text-transform: uppercase; opacity: .65; }
.retcon-fields dd { margin: .3rem 0 0; overflow-wrap: anywhere; white-space: pre-wrap; }
.retcon-actions { display: flex; flex-wrap: wrap; gap: .6rem; border-top: 1px solid var(--border, #d7dbe2); padding-top: 1rem; }
.retcon-actions button, .retcon-feedback button { min-height: 44px; padding: .65rem .9rem; border-radius: 10px; border: 1px solid currentColor; background: transparent; cursor: pointer; }
.retcon-actions button:disabled { opacity: .55; cursor: wait; }
.retcon-actions .destructive { font-weight: 700; }
.retcon-feedback { display: flex; flex-wrap: wrap; gap: .65rem; align-items: center; padding: .8rem; margin: .8rem 0; overflow-wrap: anywhere; }
.retcon-feedback.error { border-width: 2px; }
.retcon-empty { padding: 1rem; }
@media (max-width: 820px) {
  .retcon-layout { grid-template-columns: 1fr; }
  .retcon-list { grid-template-columns: repeat(2, minmax(0, 1fr)); max-height: none; }
}
@media (max-width: 560px) {
  .retcon-list { grid-template-columns: 1fr; }
  .retcon-fields { grid-template-columns: 1fr; }
  .retcon-detail-header { flex-direction: column; }
  .retcon-actions { display: grid; grid-template-columns: 1fr; }
  .retcon-actions button { width: 100%; }
}
</style>
