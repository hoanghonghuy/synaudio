import { defineStore } from 'pinia'
import {
  addFavorite,
  getProgress,
  listFavorites,
  removeFavorite,
  saveProgress,
} from '../api/client'
import type { Favorite, ListeningProgress } from '../api/types'
import { mergeGuestProgressIfAbsent } from './guest-progress-merge.ts'

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
      const res = await listFavorites()
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
        await removeFavorite(storyID)
      } else {
        await addFavorite(storyID)
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
      const p = await getProgress(chapterID)
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
      const saved = await saveProgress(chapterID, {
        position_ms: positionMs,
        audio_asset_id: audioAssetID,
        playback_session_id: '',
      })
      this.progress[chapterID] = saved
    },

    // Guest records do not carry an authoritative server-comparable timestamp.
    // Import only on an explicit server "not found". Any network/auth/5xx uncertainty
    // leaves the guest record intact for a later retry and never writes server state.
    async mergeGuestProgress() {
      if (this.isGuest) return
      const guest = readGuestProgress()
      for (const [chapterID, g] of Object.entries(guest)) {
        const outcome = await mergeGuestProgressIfAbsent(
          () => this.loadProgress(chapterID),
          () => this.saveProgress(chapterID, g.positionMs, g.audioAssetID),
        )
        if (outcome !== 'deferred') {
          delete guest[chapterID]
        }
      }

      if (Object.keys(guest).length === 0) {
        localStorage.removeItem(GUEST_PROGRESS_KEY)
      } else {
        writeGuestProgress(guest)
      }
    },
  },
})
