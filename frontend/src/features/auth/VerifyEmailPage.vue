<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { resendEmailVerification, verifyEmail } from '../../api/client'

const route = useRoute()
const email = ref((route.query.email as string) ?? '')
const token = ref((route.query.token as string) ?? '')
const submitting = ref(false)
const resending = ref(false)
const verified = ref(false)
const message = ref('')
const error = ref('')

async function submit() {
  submitting.value = true
  error.value = ''
  message.value = ''
  try {
    await verifyEmail(email.value, token.value)
    verified.value = true
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Mã xác minh không hợp lệ hoặc đã hết hạn.'
  } finally {
    submitting.value = false
  }
}

async function resend() {
  resending.value = true
  error.value = ''
  message.value = ''
  try {
    await resendEmailVerification(email.value)
    message.value = 'Nếu email thuộc một tài khoản, mã xác minh mới sẽ được gửi đến bạn.'
  } catch {
    message.value = 'Nếu email thuộc một tài khoản, mã xác minh mới sẽ được gửi đến bạn.'
  } finally {
    resending.value = false
  }
}
</script>

<template>
  <section class="page auth-page">
    <RouterLink class="back-link" to="/auth">← Về đăng nhập</RouterLink>

    <div class="auth-panel">
      <p class="eyebrow">Bảo vệ tài khoản</p>
      <h1>Xác minh email</h1>
      <p class="auth-intro">Xác minh địa chỉ email để hoàn tất việc tạo tài khoản Synaudio.</p>

      <form v-if="!verified" class="auth-form" @submit.prevent="submit">
        <label for="verify-email">
          Email
          <input id="verify-email" v-model="email" type="email" autocomplete="email" required />
        </label>
        <label for="verify-token">
          Mã xác minh
          <input id="verify-token" v-model="token" type="text" autocomplete="one-time-code" required />
        </label>
        <p v-if="error" class="status-state error" role="alert">{{ error }}</p>
        <p v-if="message" class="status-state success" role="status" aria-live="polite">{{ message }}</p>
        <button type="submit" :disabled="submitting">
          {{ submitting ? 'Đang xác minh...' : 'Xác minh email' }}
        </button>
        <button class="text-button" type="button" :disabled="resending" @click="resend">
          {{ resending ? 'Đang gửi lại...' : 'Gửi lại mã xác minh' }}
        </button>
      </form>

      <div v-else class="status-state success" role="status" aria-live="polite">
        <strong>Email đã được xác minh.</strong>
        <p>Bạn có thể đăng nhập để tiếp tục.</p>
      </div>

      <RouterLink class="auth-primary-link" to="/auth">Đăng nhập</RouterLink>
    </div>
  </section>
</template>
