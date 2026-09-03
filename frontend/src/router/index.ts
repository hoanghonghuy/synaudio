import { createRouter, createWebHistory } from 'vue-router'
import HomePage from '../features/catalog/HomePage.vue'
import StoryDetail from '../features/catalog/StoryDetail.vue'
import AdminShell from '../features/admin/AdminShell.vue'
import AuditPage from '../features/admin/AuditPage.vue'
import StoryReader from '../features/reader/StoryReader.vue'
import StoryControlCenter from '../features/admin/StoryControlCenter.vue'
import StoryPlanningStudio from '../features/admin/StoryPlanningStudio.vue'
import AuthPage from '../features/auth/AuthPage.vue'
import ForgotPasswordPage from '../features/auth/ForgotPasswordPage.vue'
import ResetPasswordPage from '../features/auth/ResetPasswordPage.vue'
import VerifyEmailPage from '../features/auth/VerifyEmailPage.vue'
import SecurityPage from '../features/auth/SecurityPage.vue'
import ContentReviewPage from '../features/admin/ContentReviewPage.vue'
import { useAuthStore } from '../stores/auth'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomePage,
    },
    {
      path: '/admin',
      name: 'admin',
      component: AdminShell,
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/audit',
      name: 'admin-audit',
      component: AuditPage,
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/auth',
      name: 'auth',
      component: AuthPage,
    },
    {
      path: '/auth/forgot-password',
      name: 'forgot-password',
      component: ForgotPasswordPage,
    },
    {
      path: '/auth/reset-password',
      name: 'reset-password',
      component: ResetPasswordPage,
    },
    {
      path: '/auth/verify-email',
      name: 'verify-email',
      component: VerifyEmailPage,
    },
    {
      path: '/account/security',
      name: 'account-security',
      component: SecurityPage,
      meta: { requiresAuth: true },
    },
    {
      path: '/stories/:storyID/read',
      name: 'reader',
      component: StoryReader,
    },
    {
      path: '/stories/:storyID',
      name: 'story-detail',
      component: StoryDetail,
    },
    {
      path: '/admin/stories/:storyID/planning',
      name: 'story-planning',
      component: StoryPlanningStudio,
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/stories/:storyID/control',
      name: 'control-center',
      component: StoryControlCenter,
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/admin/stories/:storyID/review',
      name: 'content-review',
      component: ContentReviewPage,
      meta: { requiresAuth: true, requiresAdmin: true },
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.initialized) {
    await auth.loadCurrentUser()
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { path: '/auth', query: { redirect: to.fullPath } }
  }
  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return { path: '/', query: { error: 'forbidden' } }
  }
  return true
})
