import { defineStore } from 'pinia'
import {
  clearAccessToken,
  getCurrentUser,
  logout as logoutRequest,
  logoutAll as logoutAllRequest,
} from '../api/client'
import { useListenerStore } from './listener'
import type { AuthUser } from '../api/types'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as AuthUser | null,
    loading: false,
    initialized: false,
  }),

  getters: {
    isAuthenticated: (state) => state.user !== null,
    isAdmin: (state) => state.user?.roles.includes('ADMIN') ?? false,
  },

  actions: {
    async loadCurrentUser() {
      this.loading = true
      try {
        this.user = await getCurrentUser()
        await this.syncListener()
      } catch {
        this.user = null
        clearAccessToken()
        useListenerStore().setUserID('')
      } finally {
        this.loading = false
        this.initialized = true
      }
    },

    setUser(user: AuthUser) {
      this.user = user
      this.initialized = true
    },

    async syncListener() {
      const listener = useListenerStore()
      listener.setUserID(this.user?.id ?? '')
      try {
        if (this.user) {
          await listener.mergeGuestProgress()
        }
        await listener.loadFavorites()
      } catch {
        // Listener data must not prevent authentication from completing.
      }
    },

    async clearLocalSession() {
      clearAccessToken()
      this.user = null
      const listener = useListenerStore()
      listener.setUserID('')
      await listener.loadFavorites()
    },

    async logout() {
      try {
        await logoutRequest()
      } finally {
        await this.clearLocalSession()
      }
    },

    async logoutAll() {
      try {
        await logoutAllRequest()
      } finally {
        await this.clearLocalSession()
      }
    },
  },
})
