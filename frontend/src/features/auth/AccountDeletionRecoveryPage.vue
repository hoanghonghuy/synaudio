<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import {
  confirmAccountDeletionRecovery,
  requestAccountDeletionRecovery,
} from '../../api/client'

const route = useRoute()
const router = useRouter()
const email = ref(typeof route.query.email === 'string' ? route.query.email : '')
const token = ref(typeof route.query.token === 'string' ? route.query.token : '')
const requesting = ref(false)
const confirming = ref(false)
const requestSent = ref(false)
const error = ref('')
const hasToken = computed(() => token.value.trim() !== '')

async function requestRecovery() {
  requesting.value = true
  error.value = ''
  try {
    await requestAccountDeletionRecovery(email.value)
    requestSent.value = true
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể gửi liên kết khôi phục lúc này.'
  } finally {
    requesting.value = false
  }
}

async function confirmRecovery() {
  confirming.value = true
  error.value = ''
  try {
    await confirmAccountDeletionRecovery(email.value, token.value)
    await router.replace({ path: '/auth', query: { recovered: 'account-deletion' } })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Liên kết khôi phục không hợp lệ hoặc đã hết hạn.'
  } finally {
    confirming.value = false
  }
}
</script>

<template>
  <section class="page auth-page">
    <div class="auth-card">
      <p class="eyebrow">Khôi phục tài khoản</p>
      <h1>Hủy yêu cầu xóa tài khoản</h1>
      <p class="muted">
        Tài khoản đang chờ xóa không thể dùng phiên đăng nhập cũ. Xác minh quyền sở hữu qua liên kết một lần được gửi đến email của tài khoản.
      </p>

      <form v-if="!hasToken" class="auth-form" @submit.prevent="requestRecovery">
        <label for="deletion-recovery-email">
          Email
          <input id="deletion-recovery-email" v-model.trim="email" type="email" autocomplete="email" required />
        </label>
        <button type="submit" :disabled="requesting">
          {{ requesting ? 'Đang gửi...' : 'Gửi liên kết khôi phục' }}
        </button>
        <p v-if="requestSent" class="status-state success" role="status">
          Nếu tài khoản đủ điều kiện khôi phục, liên kết xác minh đã được gửi. Hãy kiểm tra email.
        </p>
      </form>

      <form v-else class="auth-form" @submit.prevent="confirmRecovery">
        <label for="deletion-recovery-confirm-email">
          Email
          <input id="deletion-recovery-confirm-email" v-model.trim="email" type="email" autocomplete="email" required />
        </label>
        <button type="submit" :disabled="confirming || !email.trim()">
          {{ confirming ? 'Đang khôi phục...' : 'Hủy yêu cầu xóa và kích hoạt lại' }}
        </button>
      </form>

      <p v-if="error" class="status-state error" role="alert">{{ error }}</p>
      <RouterLink class="back-link" to="/auth">← Quay lại đăng nhập</RouterLink>
    </div>
  </section>
</template>
