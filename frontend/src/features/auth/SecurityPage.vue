<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import {
  cancelAccountDeletion,
  confirmTOTP,
  disableTOTP,
  listAuthSessions,
  requestAccountDeletion,
  revokeAuthSession,
  setupTOTP,
  type AuthSession,
} from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const router = useRouter()
const secret = ref('')
const code = ref('')
const recoveryCodes = ref<string[]>([])
const sessions = ref<AuthSession[]>([])
const loading = ref(false)
const disabling = ref(false)
const deleting = ref(false)
const sessionsLoading = ref(false)
const error = ref('')
const success = ref('')

const mfaEnabled = computed(() => auth.user?.mfa_enabled ?? false)
const hasRecoveryCodes = computed(() => recoveryCodes.value.length > 0)
const accountDeactivated = computed(() => auth.user?.status === 'DEACTIVATED')

async function beginSetup() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    const response = await setupTOTP()
    secret.value = response.secret
    code.value = ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể bắt đầu thiết lập MFA.'
  } finally {
    loading.value = false
  }
}

async function confirmSetup() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    const response = await confirmTOTP(code.value)
    recoveryCodes.value = response.recovery_codes
    secret.value = ''
    code.value = ''
    if (auth.user) auth.user.mfa_enabled = true
    success.value = 'MFA đã được bật cho tài khoản của bạn.'
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Mã xác thực không hợp lệ.'
  } finally {
    loading.value = false
  }
}

async function disable() {
  if (!window.confirm('Bạn có chắc muốn tắt MFA cho tài khoản này không?')) return
  disabling.value = true
  error.value = ''
  success.value = ''
  try {
    await disableTOTP()
    if (auth.user) auth.user.mfa_enabled = false
    success.value = 'MFA đã được tắt.'
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tắt MFA.'
  } finally {
    disabling.value = false
  }
}

async function loadSessions() {
  if (!auth.isAuthenticated) return
  sessionsLoading.value = true
  try {
    sessions.value = (await listAuthSessions()).items
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải danh sách phiên đăng nhập.'
  } finally {
    sessionsLoading.value = false
  }
}

async function revokeSession(session: AuthSession) {
  const label = session.current ? 'phiên hiện tại' : 'phiên đăng nhập này'
  if (!window.confirm(`Bạn có chắc muốn thu hồi ${label} không?`)) return
  sessionsLoading.value = true
  error.value = ''
  try {
    await revokeAuthSession(session.id)
    if (session.current) {
      await auth.clearLocalSession()
      await router.replace('/auth')
      return
    }
    await loadSessions()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể thu hồi phiên đăng nhập.'
  } finally {
    sessionsLoading.value = false
  }
}

async function logoutEverywhere() {
  if (!window.confirm('Đăng xuất khỏi tất cả thiết bị và thu hồi toàn bộ phiên đăng nhập?')) return
  sessionsLoading.value = true
  error.value = ''
  try {
    await auth.logoutAll()
    await router.replace('/auth')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể đăng xuất khỏi tất cả thiết bị.'
  } finally {
    sessionsLoading.value = false
  }
}

async function requestDeletion() {
  if (!window.confirm('Tài khoản sẽ bị vô hiệu hóa ngay lập tức. Bạn có chắc muốn bắt đầu yêu cầu xóa tài khoản không?')) return
  deleting.value = true
  error.value = ''
  success.value = ''
  try {
    await requestAccountDeletion()
    if (auth.user) auth.user.status = 'DEACTIVATED'
    success.value = 'Tài khoản đã được vô hiệu hóa.'
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể yêu cầu xóa tài khoản.'
  } finally {
    deleting.value = false
  }
}

async function cancelDeletion() {
  deleting.value = true
  error.value = ''
  success.value = ''
  try {
    await cancelAccountDeletion()
    if (auth.user) auth.user.status = 'ACTIVE'
    success.value = 'Yêu cầu xóa tài khoản đã được hủy.'
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể hủy yêu cầu xóa tài khoản.'
  } finally {
    deleting.value = false
  }
}

function copyRecoveryCodes() {
  void navigator.clipboard?.writeText(recoveryCodes.value.join('\n'))
}

function finishRecoveryCodes() {
  recoveryCodes.value = []
}

onMounted(async () => {
  if (!auth.initialized) await auth.loadCurrentUser()
  if (!auth.isAuthenticated) {
    await router.replace('/auth')
    return
  }
  await loadSessions()
})
</script>

<template>
  <section class="page security-page">
    <RouterLink class="back-link" to="/admin">← Về Studio</RouterLink>

    <div class="security-layout">
      <header class="security-heading">
        <p class="eyebrow">Tài khoản / Bảo mật</p>
        <h1>Bảo mật tài khoản</h1>
        <p class="page-intro">Quản lý MFA và các phiên đăng nhập đang còn hiệu lực của tài khoản.</p>
      </header>

      <section class="security-card" aria-labelledby="sessions-heading">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Phiên đăng nhập</p>
            <h2 id="sessions-heading">Thiết bị đang đăng nhập</h2>
          </div>
          <button class="danger-button" type="button" :disabled="sessionsLoading" @click="logoutEverywhere">
            Đăng xuất tất cả
          </button>
        </div>
        <p class="muted">Thu hồi một phiên sẽ làm access token của phiên đó mất hiệu lực ngay.</p>
        <p v-if="sessionsLoading && sessions.length === 0" class="muted">Đang tải phiên đăng nhập...</p>
        <div v-for="session in sessions" :key="session.id" class="security-action">
          <div>
            <strong>{{ session.current ? 'Thiết bị hiện tại' : session.user_agent_summary || 'Thiết bị khác' }}</strong>
            <p class="muted">
              Hoạt động gần nhất: {{ session.last_used_at || session.created_at }}
              <span v-if="session.safe_ip_metadata"> · IP {{ session.safe_ip_metadata }}</span>
            </p>
          </div>
          <button class="secondary-button" type="button" :disabled="sessionsLoading" @click="revokeSession(session)">
            {{ session.current ? 'Đăng xuất' : 'Thu hồi' }}
          </button>
        </div>
      </section>

      <section class="security-card" aria-labelledby="mfa-heading">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Xác thực hai bước</p>
            <h2 id="mfa-heading">Ứng dụng xác thực</h2>
          </div>
          <span class="badge" :class="{ 'badge-active': mfaEnabled }">{{ mfaEnabled ? 'Đang bật' : 'Chưa bật' }}</span>
        </div>

        <p class="muted">Dùng Google Authenticator, 1Password hoặc ứng dụng TOTP tương thích để tạo mã gồm 6 chữ số.</p>

        <div v-if="!mfaEnabled && !secret" class="security-action">
          <p>Thiết lập MFA để bảo vệ các thao tác quản trị và tài khoản của bạn.</p>
          <button type="button" :disabled="loading" @click="beginSetup">{{ loading ? 'Đang chuẩn bị...' : 'Thiết lập MFA' }}</button>
        </div>

        <form v-if="secret" class="auth-form mfa-form" @submit.prevent="confirmSetup">
          <div class="instruction-block">
            <h3>1. Thêm tài khoản vào ứng dụng</h3>
            <p>Nhập thủ công secret key này vào ứng dụng xác thực của bạn:</p>
            <code class="totp-secret" aria-label="Secret key để thiết lập MFA">{{ secret }}</code>
            <p class="field-help">Không chia sẻ secret key. Bạn chỉ thấy key này trong bước thiết lập hiện tại.</p>
          </div>
          <label for="mfa-code">
            2. Nhập mã 6 chữ số
            <input id="mfa-code" v-model="code" type="text" inputmode="numeric" autocomplete="one-time-code" pattern="[0-9]{6}" maxlength="6" required />
          </label>
          <p v-if="error" class="status-state error" role="alert">{{ error }}</p>
          <button type="submit" :disabled="loading || code.length !== 6">{{ loading ? 'Đang xác minh...' : 'Xác nhận và bật MFA' }}</button>
        </form>

        <div v-if="mfaEnabled" class="security-action">
          <p>MFA đang bảo vệ tài khoản của bạn.</p>
          <button class="danger-button" type="button" :disabled="disabling" @click="disable">{{ disabling ? 'Đang tắt...' : 'Tắt MFA' }}</button>
        </div>

        <p v-if="error && !secret" class="status-state error" role="alert">{{ error }}</p>
        <p v-if="success" class="status-state success" role="status" aria-live="polite">{{ success }}</p>
      </section>

      <section v-if="hasRecoveryCodes" class="security-card recovery-card" aria-labelledby="recovery-heading">
        <p class="eyebrow">Lưu ngay</p>
        <h2 id="recovery-heading">Mã khôi phục</h2>
        <p>Đây là lần duy nhất các mã này được hiển thị. Lưu chúng ở nơi an toàn; mỗi mã chỉ dùng được một lần.</p>
        <div class="recovery-codes" aria-label="Danh sách mã khôi phục">
          <code v-for="recoveryCode in recoveryCodes" :key="recoveryCode">{{ recoveryCode }}</code>
        </div>
        <div class="recovery-actions">
          <button class="secondary-button" type="button" @click="copyRecoveryCodes">Sao chép mã</button>
          <button type="button" @click="finishRecoveryCodes">Tôi đã lưu</button>
        </div>
      </section>

      <section class="security-card account-danger-zone" aria-labelledby="account-lifecycle-heading">
        <p class="eyebrow">Quản lý tài khoản</p>
        <h2 id="account-lifecycle-heading">Xóa tài khoản</h2>
        <p>Yêu cầu này vô hiệu hóa tài khoản ngay lập tức. Dữ liệu cá nhân chỉ được ẩn danh sau thời gian chờ theo chính sách hệ thống.</p>
        <div v-if="accountDeactivated" class="status-state error" role="status">
          <strong>Tài khoản đang chờ xóa.</strong>
        </div>
        <div class="recovery-actions">
          <button v-if="accountDeactivated" class="secondary-button" type="button" :disabled="deleting" @click="cancelDeletion">{{ deleting ? 'Đang xử lý...' : 'Hủy yêu cầu xóa' }}</button>
          <button v-else class="danger-button" type="button" :disabled="deleting" @click="requestDeletion">{{ deleting ? 'Đang xử lý...' : 'Yêu cầu xóa tài khoản' }}</button>
        </div>
      </section>
    </div>
  </section>
</template>
