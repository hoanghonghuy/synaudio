<script setup lang="ts">
import { ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const menuOpen = ref(false)

watch(
  () => route.path,
  () => {
    menuOpen.value = false
  },
)

async function signOut() {
  try {
    await auth.logout()
    await router.push('/')
  } catch {
    // Keep the current session state visible if the server is unavailable.
  }
}
</script>

<template>
  <div class="shell">
    <header class="topbar">
      <div class="topbar-inner">
        <RouterLink class="brand" to="/" aria-label="Synaudio - Trang chủ">
          <span class="brand-mark" aria-hidden="true">S</span>
          <span>Synaudio</span>
        </RouterLink>

        <button
          class="menu-toggle"
          type="button"
          :aria-expanded="menuOpen"
          aria-controls="main-navigation"
          :aria-label="menuOpen ? 'Đóng menu điều hướng' : 'Mở menu điều hướng'"
          @click="menuOpen = !menuOpen"
        >
          <span aria-hidden="true">{{ menuOpen ? '×' : '☰' }}</span>
        </button>

        <nav id="main-navigation" class="main-nav" :class="{ open: menuOpen }" aria-label="Điều hướng chính">
          <RouterLink to="/">Khám phá</RouterLink>
          <RouterLink v-if="!auth.isAuthenticated" to="/auth">Đăng nhập</RouterLink>
          <template v-else>
            <RouterLink class="nav-cta" to="/admin">Studio</RouterLink>
            <RouterLink to="/account/security">Bảo mật</RouterLink>
            <button class="nav-sign-out" type="button" @click="signOut">Đăng xuất</button>
          </template>
        </nav>
      </div>
    </header>
    <a class="skip-link" href="#main-content">Bỏ qua đến nội dung chính</a>
    <main id="main-content">
      <RouterView />
    </main>
  </div>
</template>
