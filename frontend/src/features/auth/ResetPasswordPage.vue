<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { resetPassword } from '../../api/client'

const route = useRoute()
const router = useRouter()
const email = ref((route.query.email as string) ?? '')
const token = ref((route.query.token as string) ?? '')
const newPassword = ref('')
const submitting = ref(false)
const completed = ref(false)
const error = ref('')

async function submit() {
  submitting.value = true
  error.value = ''
  try {
    await resetPassword(email.value, token.value, newPassword.value)
    completed.value = true
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Liên kết đặt lại mật khẩu không hợp lệ hoặc đã hết hạn.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="page auth-page">
    <RouterLink class="back-link" to="/auth">← Về đăng nhập</RouterLink>

    <div class="auth-panel">
      <p class="eyebrow">Khôi phục tài khoản</p>
      <h1>Đặt lại mật khẩu</h1>
      <p class="auth-intro">Tạo mật khẩu mới để tiếp tục sử dụng Synaudio.</p>

      <form v-if="!completed" class="auth-form" @submit.prevent="submit">
        <label for="reset-account-email">
          Email
          <input id="reset-account-email" v-model="email" type="email" autocomplete="email" required />
        </label>
        <label for="reset-token">
          Mã xác nhận
          <input id="reset-token" v-model="token" type="text" autocomplete="one-time-code" required />
        </label>
        <label for="new-password">
          Mật khẩu mới
          <input id="new-password" v-model="newPassword" type="password" autocomplete="new-password" required />
        </label>
        <p v-if="error" class="status-state error" role="alert">{{ error }}</p>
        <button type="submit" :disabled="submitting">
          {{ submitting ? 'Đang cập nhật...' : 'Đặt lại mật khẩu' }}
        </button>
      </form>

      <div v-else class="status-state success" role="status" aria-live="polite">
        <strong>Mật khẩu đã được cập nhật.</strong>
        <p>Bạn có thể đăng nhập bằng mật khẩu mới.</p>
      </div>

      <button v-if="completed" class="auth-primary-link text-button" type="button" @click="router.push('/auth')">
        Đăng nhập ngay
      </button>
    </div>
  </section>
</template>
