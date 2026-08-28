import { defineStore } from 'pinia'
import {
  addFavorite,
  getProgress,
  listFavorites,
  removeFavorite,
  saveProgress,
} from '../api/client'
import type { Favorite, ListeningProgress } from '../api/types'

const GUEST_FAVORITES_KEY = 'synaudio.guest.favorites'
const GUEST_PROGRESS_KEY = 'synaudio.guest.progress'

interface GuestProgress {
  [chapterID: string]: { positionMs: number; audioAssetID: string }
}

function readGuestFavorites(): string[] {
  try {
    const raw = localStorage.getItem(GUEST_FAVORITES_KEY)
    return raw ? (JSON.parse(raw) as string[]) : []
  } catch {
    return []
  }
}

function writeGuestFavorites(ids: string[]) {
  localStorage.setItem(GUEST_FAVORITES_KEY, JSON.stringify(ids))
}

function readGuestProgress(): GuestProgress {
  try {
    const raw = localStorage.getItem(GUEST_PROGRESS_KEY)
    return raw ? (JSON.parse(raw) as GuestProgress) : {}
  } catch {
    return {}
  }
}

function writeGuestProgress(p: GuestProgress) {
  localStorage.setItem(GUEST_PROGRESS_KEY, JSON.stringify(p))
}

export const useListenerStore = defineStore('listener', {
  state: () => ({
    userID: '' as string,
    favorites: [] as string[],
    progress: {} as Record<string, ListeningProgress>,
  }),

  getters: {
    isGuest: (state) => state.userID === '',
    isFavorite: (state) => (storyID: string) => state.favorites.includes(storyID),
  },

  actions: {
    setUserID(id: string) {
      this.userID = id
    },

    async loadFavorites() {
      if (this.isGuest) {
        this.favorites = readGuestFavorites()
        return
      }
      const res = await listFavorites(this.userID)
      this.favorites = res.favorites.map((f: Favorite) => f.StoryID)
    },

    async toggleFavorite(storyID: string) {
      if (this.isGuest) {
        const ids = readGuestFavorites()
        const idx = ids.indexOf(storyID)
        if (idx >= 0) ids.splice(idx, 1)
        else ids.push(storyID)
        writeGuestFavorites(ids)
        this.favorites = ids
        return
      }

      if (this.favorites.includes(storyID)) {
        await removeFavorite(this.userID, storyID)
      } else {
        await addFavorite(this.userID, storyID)
      }
      await this.loadFavorites()
    },

    async loadProgress(chapterID: string) {
      if (this.isGuest) {
        const p = readGuestProgress()
        const g = p[chapterID]
        if (g) {
          this.progress[chapterID] = {
            UserID: '',
            ChapterID: chapterID,
            PositionMs: g.positionMs,
            CompletedAt: '',
            LastAudioAssetID: g.audioAssetID,
            LastPlaybackSessionID: '',
            Version: 0,
            RelistenStatus: 'NO_RELISTEN_NEEDED',
          }
        }
        return
      }
      const p = await getProgress(this.userID, chapterID)
      this.progress[chapterID] = p
    },

    async saveProgress(chapterID: string, positionMs: number, audioAssetID: string) {
      if (this.isGuest) {
        const p = readGuestProgress()
        p[chapterID] = { positionMs, audioAssetID }
        writeGuestProgress(p)
        this.progress[chapterID] = {
          UserID: '',
          ChapterID: chapterID,
          PositionMs: positionMs,
          CompletedAt: '',
          LastAudioAssetID: audioAssetID,
          LastPlaybackSessionID: '',
          Version: 0,
          RelistenStatus: 'NO_RELISTEN_NEEDED',
        }
        return
      }
      const saved = await saveProgress(this.userID, chapterID, {
        position_ms: positionMs,
        audio_asset_id: audioAssetID,
        playback_session_id: '',
      })
      this.progress[chapterID] = saved
    },

    // Merge guest progress into server on login (idempotent).
    async mergeGuestProgress() {
      if (this.isGuest) return
      const guest = readGuestProgress()
      for (const [chapterID, g] of Object.entries(guest)) {
        try {
          await this.loadProgress(chapterID)
        } catch {
          // no server progress yet
        }
        const existing = this.progress[chapterID]
        if (!existing || existing.PositionMs < g.positionMs) {
          await this.saveProgress(chapterID, g.positionMs, g.audioAssetID)
        }
      }
      localStorage.removeItem(GUEST_PROGRESS_KEY)
    },
  },
})
