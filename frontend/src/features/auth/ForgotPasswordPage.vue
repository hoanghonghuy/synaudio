<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { requestPasswordReset } from '../../api/client'

const email = ref('')
const submitting = ref(false)
const submitted = ref(false)
const error = ref('')

async function submit() {
  submitting.value = true
  error.value = ''
  try {
    await requestPasswordReset(email.value)
    submitted.value = true
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể gửi yêu cầu lúc này.'
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
      <h1>Quên mật khẩu?</h1>
      <p class="auth-intro">Nhập email của bạn. Nếu tài khoản tồn tại, hướng dẫn đặt lại mật khẩu sẽ được gửi đến email đó.</p>

      <form v-if="!submitted" class="auth-form" @submit.prevent="submit">
        <label for="reset-email">
          Email
          <input id="reset-email" v-model="email" type="email" autocomplete="email" required />
        </label>
        <p v-if="error" class="status-state error" role="alert">{{ error }}</p>
        <button type="submit" :disabled="submitting">
          {{ submitting ? 'Đang gửi...' : 'Gửi hướng dẫn' }}
        </button>
      </form>

      <div v-else class="status-state success" role="status" aria-live="polite">
        <strong>Hãy kiểm tra email của bạn.</strong>
        <p>Nếu email phù hợp với một tài khoản, bạn sẽ nhận được hướng dẫn tiếp theo.</p>
      </div>

      <RouterLink class="auth-primary-link" to="/auth">Quay lại đăng nhập</RouterLink>
    </div>
  </section>
</template>
