<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { getCurrentUser, login, register } from '../../api/client'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const mode = ref(route.query.mode === 'register' ? 'register' : 'login')
const email = ref('')
const password = ref('')
const submitting = ref(false)
const error = ref('')
const success = ref('')

const isRegister = computed(() => mode.value === 'register')

watch(
  () => route.query.mode,
  (queryMode) => {
    mode.value = queryMode === 'register' ? 'register' : 'login'
  },
)

function setMode(nextMode: 'login' | 'register') {
  mode.value = nextMode
  error.value = ''
  success.value = ''
  password.value = ''
  void router.replace({ query: nextMode === 'register' ? { mode: nextMode } : {} })
}

async function submit() {
  submitting.value = true
  error.value = ''
  success.value = ''

  try {
    if (isRegister.value) {
      const account = await register(email.value, password.value)
      await router.push({ path: '/auth/verify-email', query: { email: account.email } })
      return
    }

        await login(email.value, password.value)
        auth.setUser(await getCurrentUser())
        await auth.syncListener()
    await router.push('/admin')
  } catch (e) {
    error.value = e instanceof Error ? e.message : isRegister.value ? 'Không thể tạo tài khoản.' : 'Email hoặc mật khẩu chưa đúng.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="page auth-page">
    <RouterLink class="back-link" to="/">← Về khám phá</RouterLink>

    <div class="auth-panel">
      <p class="eyebrow">{{ isRegister ? 'Bắt đầu hành trình' : 'Chào mừng trở lại' }}</p>
      <h1>{{ isRegister ? 'Tạo tài khoản' : 'Đăng nhập Synaudio' }}</h1>
      <p class="auth-intro">
        {{ isRegister ? 'Lưu tiến độ nghe và quay lại đúng nơi bạn đã dừng.' : 'Tiếp tục câu chuyện bạn đang nghe dở.' }}
      </p>

      <form class="auth-form" @submit.prevent="submit">
        <label for="auth-email">
          Email
          <input id="auth-email" v-model="email" type="email" autocomplete="email" required />
        </label>
        <label for="auth-password">
          Mật khẩu
          <input
            id="auth-password"
            v-model="password"
            type="password"
            :autocomplete="isRegister ? 'new-password' : 'current-password'"
            required
          />
        </label>

        <p v-if="error" class="status-state error" role="alert">{{ error }}</p>
        <p v-if="success" class="status-state success" role="status" aria-live="polite">{{ success }}</p>
        <p v-if="submitting" class="muted" role="status" aria-live="polite">
          {{ isRegister ? 'Đang tạo tài khoản...' : 'Đang đăng nhập...' }}
        </p>

        <button type="submit" :disabled="submitting">
          {{ isRegister ? 'Đăng ký' : 'Đăng nhập' }}
        </button>
      </form>

      <div class="auth-links">
        <button v-if="!isRegister" class="text-button" type="button" @click="setMode('register')">
          Chưa có tài khoản? Đăng ký
        </button>
        <button v-else class="text-button" type="button" @click="setMode('login')">
          Đã có tài khoản? Đăng nhập
        </button>
        <RouterLink v-if="!isRegister" to="/auth/forgot-password">Quên mật khẩu?</RouterLink>
      </div>
    </div>
  </section>
</template>
