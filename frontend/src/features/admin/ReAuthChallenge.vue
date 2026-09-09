<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { reAuth } from '../../api/client'
import { adminSecurityErrorMessage } from '../../api/http-error'
import type { PrivilegedAssuranceReason } from '../../api/privileged-request'

const props = defineProps<{
  open: boolean
  reason: PrivilegedAssuranceReason | null
}>()

const emit = defineEmits<{
  completed: [success: boolean]
  dismissed: []
}>()

const mode = ref<'totp' | 'recovery'>('totp')
const code = ref('')
const recoveryCode = ref('')
const submitting = ref(false)
const error = ref('')

const title = computed(() =>
  props.reason === 'RECENT_AUTH_REQUIRED'
    ? 'Xác thực gần đây bắt buộc'
    : 'Xác minh MFA cho phiên hiện tại',
)

const intro = computed(() =>
  props.reason === 'RECENT_AUTH_REQUIRED'
    ? 'Thao tác quản trị này yêu cầu bạn xác minh lại danh tính trên phiên đăng nhập hiện tại. Nhập mã TOTP hoặc mã khôi phục — không dùng mật khẩu.'
    : 'Phiên đăng nhập hiện tại chưa được đảm bảo MFA. Xác minh bằng mã TOTP hoặc mã khôi phục của tài khoản này trên phiên này.',
)

watch(
  () => props.open,
  (open) => {
    if (!open) return
    mode.value = 'totp'
    code.value = ''
    recoveryCode.value = ''
    error.value = ''
    submitting.value = false
  },
)

function dismiss() {
  if (submitting.value) return
  emit('dismissed')
}

async function submit() {
  submitting.value = true
  error.value = ''
  try {
    if (mode.value === 'totp') {
      await reAuth({ code: code.value.trim() })
    } else {
      await reAuth({ recovery_code: recoveryCode.value.trim() })
    }
    code.value = ''
    recoveryCode.value = ''
    emit('completed', true)
  } catch (e) {
    error.value = adminSecurityErrorMessage(e)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div v-if="open" class="reauth-overlay" role="presentation" @click.self="dismiss">
    <section
      class="reauth-dialog security-card"
      role="dialog"
      aria-modal="true"
      aria-labelledby="reauth-heading"
      @click.stop
    >
      <div class="section-heading">
        <div>
          <p class="eyebrow">Bảo mật phiên</p>
          <h2 id="reauth-heading">{{ title }}</h2>
        </div>
        <button class="secondary-button" type="button" :disabled="submitting" @click="dismiss">Đóng</button>
      </div>

      <p class="muted">{{ intro }}</p>

      <div class="reauth-mode-toggle" role="tablist" aria-label="Phương thức xác minh">
        <button
          type="button"
          role="tab"
          :aria-selected="mode === 'totp'"
          :class="{ active: mode === 'totp' }"
          @click="mode = 'totp'"
        >
          Mã TOTP
        </button>
        <button
          type="button"
          role="tab"
          :aria-selected="mode === 'recovery'"
          :class="{ active: mode === 'recovery' }"
          @click="mode = 'recovery'"
        >
          Mã khôi phục
        </button>
      </div>

      <form class="auth-form mfa-form" @submit.prevent="submit">
        <label v-if="mode === 'totp'" for="reauth-totp">
          Mã 6 chữ số
          <input
            id="reauth-totp"
            v-model="code"
            type="text"
            inputmode="numeric"
            autocomplete="one-time-code"
            pattern="[0-9]{6}"
            maxlength="6"
            required
          />
        </label>
        <label v-else for="reauth-recovery">
          Mã khôi phục
          <input id="reauth-recovery" v-model="recoveryCode" type="text" autocomplete="off" required />
        </label>

        <p v-if="error" class="status-state error" role="alert">{{ error }}</p>

        <div class="recovery-actions">
          <button
            type="submit"
            :disabled="submitting || (mode === 'totp' ? code.trim().length !== 6 : recoveryCode.trim().length === 0)"
          >
            {{ submitting ? 'Đang xác minh...' : 'Xác minh và tiếp tục' }}
          </button>
        </div>
      </form>
    </section>
  </div>
</template>
