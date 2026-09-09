<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import {
  getAdminUser,
  grantAdminRole,
  listAdminUsers,
  revokeAdminRole,
  setAdminUserStatus,
} from '../../api/client'
import {
  adminSecurityErrorMessage,
  isPermissionDenied,
  isPrivilegedAssuranceRequired,
  privilegedAssuranceCode,
} from '../../api/http-error'
import { withPrivilegedRetry, type PrivilegedAssuranceReason } from '../../api/privileged-request'
import type { AdminUserSummary, UserAccountStatus } from '../../api/types'
import ReAuthChallenge from './ReAuthChallenge.vue'

const users = ref<AdminUserSummary[]>([])
const selected = ref<AdminUserSummary | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const actionLoading = ref(false)
const listError = ref('')
const detailError = ref('')
const actionError = ref('')
const actionSuccess = ref('')

const filters = ref({
  q: '',
  status: '' as '' | UserAccountStatus,
  limit: 50,
})

const reAuthOpen = ref(false)
const reAuthReason = ref<PrivilegedAssuranceReason | null>(null)
let reAuthResolver: ((success: boolean) => void) | null = null

const hasFilters = computed(() => filters.value.q.trim() !== '' || filters.value.status !== '')

const isAdminRole = (user: AdminUserSummary) => user.roles.includes('ADMIN')

function statusLabel(status: UserAccountStatus) {
  const labels: Record<UserAccountStatus, string> = {
    ACTIVE: 'Đang hoạt động',
    SUSPENDED: 'Tạm khóa',
    DEACTIVATED: 'Đã vô hiệu',
  }
  return labels[status]
}

function requestReAuth(reason: PrivilegedAssuranceReason): Promise<boolean> {
  return new Promise((resolve) => {
    reAuthReason.value = reason
    reAuthOpen.value = true
    reAuthResolver = resolve
  })
}

function handleReAuthCompleted(success: boolean) {
  reAuthOpen.value = false
  reAuthReason.value = null
  reAuthResolver?.(success)
  reAuthResolver = null
}

function handleReAuthDismissed() {
  handleReAuthCompleted(false)
}

async function runPrivileged<T>(operation: () => Promise<T>): Promise<T> {
  return withPrivilegedRetry(operation, requestReAuth)
}

async function refreshSelected(userID: string) {
  try {
    const fresh = await runPrivileged(() => getAdminUser(userID))
    selected.value = fresh
    const index = users.value.findIndex((user) => user.id === userID)
    if (index >= 0) users.value[index] = fresh
    detailError.value = ''
  } catch (e) {
    if (isPrivilegedAssuranceRequired(e)) {
      detailError.value = adminSecurityErrorMessage(e)
      return
    }
    selected.value = null
    detailError.value = adminSecurityErrorMessage(e)
    await loadUsers()
  }
}

async function loadUsers() {
  loading.value = true
  listError.value = ''
  try {
    const response = await runPrivileged(() =>
      listAdminUsers({
        q: filters.value.q || undefined,
        status: filters.value.status || undefined,
        limit: filters.value.limit,
      }),
    )
    users.value = response.users
    if (selected.value && !users.value.some((user) => user.id === selected.value?.id)) {
      selected.value = null
    }
  } catch (e) {
    listError.value = adminSecurityErrorMessage(e)
    users.value = []
    if (privilegedAssuranceCode(e)) {
      // Re-auth modal already handled; keep list empty until retry succeeds.
    }
  } finally {
    loading.value = false
  }
}

async function selectUser(user: AdminUserSummary) {
  detailLoading.value = true
  detailError.value = ''
  actionError.value = ''
  actionSuccess.value = ''
  try {
    selected.value = await runPrivileged(() => getAdminUser(user.id))
  } catch (e) {
    selected.value = null
    detailError.value = adminSecurityErrorMessage(e)
  } finally {
    detailLoading.value = false
  }
}

async function runAction(label: string, operation: () => Promise<unknown>) {
  if (!selected.value) return
  const userID = selected.value.id
  actionLoading.value = true
  actionError.value = ''
  actionSuccess.value = ''
  try {
    await runPrivileged(operation)
    actionSuccess.value = label
    await refreshSelected(userID)
    await loadUsers()
  } catch (e) {
    actionError.value = adminSecurityErrorMessage(e)
    if (isPermissionDenied(e) || e instanceof Error) {
      await refreshSelected(userID)
    }
  } finally {
    actionLoading.value = false
  }
}

function confirmAction(message: string) {
  return window.confirm(message)
}

async function grantAdmin() {
  if (!selected.value || !confirmAction(`Cấp quyền Admin cho ${selected.value.email}?`)) return
  await runAction('Đã cấp quyền Admin.', () => grantAdminRole(selected.value!.id))
}

async function revokeAdmin() {
  if (!selected.value || !confirmAction(`Thu hồi quyền Admin của ${selected.value.email}?`)) return
  await runAction('Đã thu hồi quyền Admin.', () => revokeAdminRole(selected.value!.id))
}

async function changeStatus(status: UserAccountStatus) {
  if (!selected.value) return
  const labels: Record<UserAccountStatus, string> = {
    ACTIVE: 'kích hoạt lại',
    SUSPENDED: 'tạm khóa',
    DEACTIVATED: 'vô hiệu hóa',
  }
  if (!confirmAction(`Xác nhận ${labels[status]} tài khoản ${selected.value.email}?`)) return
  const successLabels: Record<UserAccountStatus, string> = {
    ACTIVE: 'Đã kích hoạt lại tài khoản.',
    SUSPENDED: 'Đã tạm khóa tài khoản.',
    DEACTIVATED: 'Đã vô hiệu hóa tài khoản.',
  }
  await runAction(successLabels[status], () => setAdminUserStatus(selected.value!.id, status))
}

function resetFilters() {
  filters.value = { q: '', status: '', limit: 50 }
  void loadUsers()
}

onMounted(loadUsers)
</script>

<template>
  <section class="page admin">
    <RouterLink class="back-link" to="/admin">← Về Studio</RouterLink>
    <p class="eyebrow">Studio / Governance</p>
    <h1>Quản lý bảo mật người dùng</h1>
    <p class="page-intro">
      Tìm kiếm, xem chi tiết và thực hiện các thao tác vai trò/trạng thái được backend hỗ trợ. Mọi quyền và xác thực gần đây được máy chủ quyết định — giao diện chỉ phản hồi kết quả từ API.
    </p>

    <form class="create-form" aria-label="Bộ lọc người dùng" @submit.prevent="loadUsers">
      <div class="section-heading">
        <div>
          <h2>Tìm người dùng</h2>
          <p class="muted">Lọc theo email, tên hiển thị hoặc trạng thái tài khoản.</p>
        </div>
        <button v-if="hasFilters" class="secondary-button" type="button" @click="resetFilters">Xóa bộ lọc</button>
      </div>

      <div class="row">
        <label>
          Từ khóa
          <input v-model.trim="filters.q" placeholder="email hoặc tên hiển thị" />
        </label>
        <label>
          Trạng thái
          <select v-model="filters.status">
            <option value="">Tất cả</option>
            <option value="ACTIVE">Đang hoạt động</option>
            <option value="SUSPENDED">Tạm khóa</option>
            <option value="DEACTIVATED">Đã vô hiệu</option>
          </select>
        </label>
        <label>
          Giới hạn
          <input v-model.number="filters.limit" type="number" min="1" max="100" />
        </label>
      </div>

      <button type="submit" :disabled="loading">{{ loading ? 'Đang tải...' : 'Tìm kiếm' }}</button>
    </form>

    <p v-if="listError" class="status-state error" role="alert">{{ listError }}</p>

    <div class="admin-workspace">
      <section class="admin-list-panel" aria-labelledby="user-list-heading">
        <div class="section-heading">
          <div>
            <h2 id="user-list-heading">Danh sách người dùng</h2>
            <p class="muted">Chọn một người dùng để xem chi tiết và thực hiện thao tác bảo mật.</p>
          </div>
          <span class="count-label">{{ users.length }} người dùng</span>
        </div>

        <p v-if="loading" class="status-state">Đang tải danh sách...</p>
        <p v-else-if="users.length === 0" class="empty-state">Không có người dùng phù hợp với bộ lọc hiện tại.</p>

        <div v-else class="story-table-wrap">
          <table class="story-table">
            <thead>
              <tr>
                <th>Email</th>
                <th>Trạng thái</th>
                <th>Vai trò</th>
                <th><span class="sr-only">Chi tiết</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="user in users" :key="user.id" :class="{ selected: selected?.id === user.id }">
                <td>
                  <strong>{{ user.email }}</strong>
                  <span v-if="user.display_name" class="slug">{{ user.display_name }}</span>
                </td>
                <td><span class="badge">{{ statusLabel(user.status) }}</span></td>
                <td>
                  <span v-for="role in user.roles" :key="role" class="badge">{{ role }}</span>
                </td>
                <td>
                  <button class="secondary-button" type="button" @click="selectUser(user)">Xem</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <aside class="create-form" aria-labelledby="user-detail-heading">
        <div>
          <p class="eyebrow">Chi tiết</p>
          <h2 id="user-detail-heading">Hồ sơ người dùng</h2>
        </div>

        <p v-if="detailLoading" class="status-state">Đang tải chi tiết...</p>
        <p v-else-if="!selected" class="empty-state">Chọn một người dùng để xem thông tin và thao tác bảo mật.</p>

        <template v-else>
          <dl>
            <dt>Email</dt>
            <dd>{{ selected.email }}</dd>
            <dt>Tên hiển thị</dt>
            <dd>{{ selected.display_name || '—' }}</dd>
            <dt>Trạng thái</dt>
            <dd><span class="badge">{{ statusLabel(selected.status) }}</span></dd>
            <dt>Email đã xác minh</dt>
            <dd>{{ selected.email_verified ? 'Có' : 'Chưa' }}</dd>
            <dt>Vai trò</dt>
            <dd>
              <span v-for="role in selected.roles" :key="role" class="badge">{{ role }}</span>
            </dd>
            <dt>User ID</dt>
            <dd class="slug">{{ selected.id }}</dd>
          </dl>

          <p v-if="detailError" class="status-state error" role="alert">{{ detailError }}</p>

          <section class="security-card" aria-labelledby="role-actions-heading">
            <h3 id="role-actions-heading">Vai trò Admin</h3>
            <p class="muted">Yêu cầu xác thực gần đây. Backend quyết định quyền cấp/thu hồi.</p>
            <div class="recovery-actions">
              <button
                v-if="!isAdminRole(selected)"
                type="button"
                :disabled="actionLoading"
                @click="grantAdmin"
              >
                {{ actionLoading ? 'Đang xử lý...' : 'Cấp quyền Admin' }}
              </button>
              <button
                v-else
                class="danger-button"
                type="button"
                :disabled="actionLoading"
                @click="revokeAdmin"
              >
                {{ actionLoading ? 'Đang xử lý...' : 'Thu hồi quyền Admin' }}
              </button>
            </div>
          </section>

          <section class="security-card" aria-labelledby="status-actions-heading">
            <h3 id="status-actions-heading">Trạng thái tài khoản</h3>
            <p class="muted">Yêu cầu xác thực gần đây. Không thể vô hiệu Admin cuối cùng đang hoạt động.</p>
            <div class="recovery-actions">
              <button
                v-if="selected.status !== 'ACTIVE'"
                type="button"
                :disabled="actionLoading"
                @click="changeStatus('ACTIVE')"
              >
                Kích hoạt lại
              </button>
              <button
                v-if="selected.status !== 'SUSPENDED'"
                type="button"
                :disabled="actionLoading"
                @click="changeStatus('SUSPENDED')"
              >
                Tạm khóa
              </button>
              <button
                v-if="selected.status !== 'DEACTIVATED'"
                class="danger-button"
                type="button"
                :disabled="actionLoading"
                @click="changeStatus('DEACTIVATED')"
              >
                Vô hiệu hóa
              </button>
            </div>
          </section>

          <p v-if="actionError" class="status-state error" role="alert">{{ actionError }}</p>
          <p v-if="actionSuccess" class="status-state success" role="status" aria-live="polite">{{ actionSuccess }}</p>
        </template>
      </aside>
    </div>

    <ReAuthChallenge
      :open="reAuthOpen"
      :reason="reAuthReason"
      @completed="handleReAuthCompleted"
      @dismissed="handleReAuthDismissed"
    />
  </section>
</template>
