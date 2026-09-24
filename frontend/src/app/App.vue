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
          <span class="brand-mark" aria-hidden="true">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M3 18v-6a9 9 0 0 1 18 0v6"></path>
              <path d="M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3zM3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z"></path>
            </svg>
          </span>
          <span class="brand-text">Synaudio</span>
        </RouterLink>

        <nav id="main-navigation" class="main-nav desktop-only" aria-label="Điều hướng chính">
          <RouterLink to="/">Khám phá</RouterLink>
          <template v-if="auth.isAuthenticated">
            <RouterLink to="/library">Thư viện</RouterLink>
            <RouterLink v-if="auth.isAdmin" class="nav-cta" to="/admin">Creator Studio</RouterLink>
            <RouterLink class="account-menu-link" to="/account/security" aria-label="Mở cài đặt tài khoản">
              <span class="account-avatar" aria-hidden="true">{{ auth.user?.email.slice(0, 1).toUpperCase() }}</span>
              <span class="account-menu-copy"><strong>{{ auth.user?.email }}</strong><small>Tài khoản</small></span>
            </RouterLink>
            <button class="nav-sign-out" type="button" @click="signOut">Đăng xuất</button>
          </template>
          <RouterLink v-else class="nav-login-btn" to="/auth">Đăng nhập</RouterLink>
        </nav>

        <div class="mobile-top-actions">
          <RouterLink v-if="!auth.isAuthenticated" class="mobile-login-pill" to="/auth">Đăng nhập</RouterLink>
          <button v-else class="mobile-logout-icon" type="button" aria-label="Đăng xuất" title="Đăng xuất" @click="signOut">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
              <polyline points="16 17 21 12 16 7"></polyline>
              <line x1="21" y1="12" x2="9" y2="12"></line>
            </svg>
          </button>
        </div>
      </div>
    </header>

    <a class="skip-link" href="#main-content">Bỏ qua đến nội dung chính</a>

    <main id="main-content" class="main-content">
      <RouterView />
    </main>

    <nav class="bottom-nav" aria-label="Điều hướng ứng dụng">
      <RouterLink to="/" class="bottom-nav-item" exact-active-class="active">
        <svg class="bottom-nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path>
          <polyline points="9 22 9 12 15 12 15 22"></polyline>
        </svg>
        <span class="bottom-nav-label">Khám phá</span>
      </RouterLink>

      <RouterLink :to="auth.isAuthenticated ? '/library' : '/auth'" class="bottom-nav-item" active-class="active">
        <svg class="bottom-nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>
          <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>
        </svg>
        <span class="bottom-nav-label">Thư viện</span>
      </RouterLink>

      <RouterLink v-if="auth.isAdmin" to="/admin" class="bottom-nav-item" active-class="active">
        <svg class="bottom-nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>
        </svg>
        <span class="bottom-nav-label">Studio</span>
      </RouterLink>

      <RouterLink :to="auth.isAuthenticated ? '/account/security' : '/auth'" class="bottom-nav-item" active-class="active">
        <svg class="bottom-nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
          <circle cx="12" cy="7" r="4"></circle>
        </svg>
        <span class="bottom-nav-label">{{ auth.isAuthenticated ? 'Tài khoản' : 'Đăng nhập' }}</span>
      </RouterLink>
    </nav>
  </div>
</template>
