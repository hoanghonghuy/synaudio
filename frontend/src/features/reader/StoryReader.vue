<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  completeProgress,
  getAudioURL,
  getChapterContent,
  listPublishedChapters,
} from '../../api/client'
import { useListenerStore } from '../../stores/listener'
import type { Chapter, ChapterContent } from '../../api/types'
import {
  createLatestChapterSelectionGuard,
  formatPlaybackTime,
  normalizeVolume,
  normalizePlaybackRate,
  toggleMuteState,
} from './readerSession.mjs'

const route = useRoute()
const storyID = computed(() => route.params.storyID as string)
const listener = useListenerStore()

const chapters = ref<Chapter[]>([])
const activeChapter = ref<Chapter | null>(null)
const content = ref<ChapterContent | null>(null)
const audioURL = ref('')
const audioChapterID = ref('')
const loading = ref(false)
const contentLoading = ref(false)
const audioLoading = ref(false)
const error = ref('')
const contentError = ref('')
const audioError = ref('')

const audioEl = ref<HTMLAudioElement | null>(null)
const isPlaying = ref(false)
const isBuffering = ref(false)
const currentTime = ref(0)
const duration = ref(0)
const playbackRate = ref(1)
const volume = ref(0.9)
const isMuted = ref(false)
const lastAudibleVolume = ref(0.9)
const sleepTimer = ref<'off' | '15' | '30' | '60'>('off')
const chapterDrawerOpen = ref(false)
const progressState = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')
const chapterSelection = createLatestChapterSelectionGuard()
const progressWriteIntervalMs = 15_000
const playbackRates = [0.75, 1, 1.25, 1.5, 2]
let lastProgressWriteAt = 0
let progressWrite: Promise<void> = Promise.resolve()
let progressStateTimer: number | undefined
let sleepTimerID: number | undefined
let activeSelectionMayCommit: () => boolean = () => false

const playbackPercent = computed(() => duration.value > 0 ? Math.min(100, (currentTime.value / duration.value) * 100) : 0)
const progressLabel = computed(() => {
  if (progressState.value === 'saving') return 'Đang lưu tiến độ…'
  if (progressState.value === 'saved') return 'Đã lưu tiến độ'
  if (progressState.value === 'error') return 'Chưa lưu được tiến độ'
  return ''
})

const currentChapterIndex = computed(() =>
  chapters.value.findIndex((c) => c.ID === activeChapter.value?.ID),
)
const prevChapter = computed(() =>
  currentChapterIndex.value > 0 ? chapters.value[currentChapterIndex.value - 1] : null,
)
const nextChapter = computed(() =>
  currentChapterIndex.value >= 0 && currentChapterIndex.value < chapters.value.length - 1
    ? chapters.value[currentChapterIndex.value + 1]
    : null,
)

function ownsCurrentAudio() {
  return Boolean(activeChapter.value && audioChapterID.value === activeChapter.value.ID)
}

function resetPlayerState() {
  isPlaying.value = false
  isBuffering.value = false
  currentTime.value = 0
  duration.value = 0
  progressState.value = 'idle'
  audioChapterID.value = ''
}

async function loadChapters() {
  loading.value = true
  error.value = ''
  try {
    const res = await listPublishedChapters(storyID.value)
    chapters.value = res.chapters
    if (chapters.value.length > 0) {
      const requestedChapterID = typeof route.query.chapter === 'string' ? route.query.chapter : ''
      const requested = chapters.value.find((chapter) => chapter.ID === requestedChapterID)
      await selectChapter(requested ?? chapters.value[0])
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải danh sách chương.'
  } finally {
    loading.value = false
  }
}

async function selectChapter(chapter: Chapter) {
  chapterDrawerOpen.value = false
  persistCurrentPosition(true)
  const mayCommit = chapterSelection.begin(chapter.ID)
  activeSelectionMayCommit = mayCommit
  activeChapter.value = chapter
  lastProgressWriteAt = 0
  content.value = null
  audioURL.value = ''
  contentError.value = ''
  audioError.value = ''
  contentLoading.value = true
  audioLoading.value = true
  resetPlayerState()

  const contentRequest = getChapterContent(chapter.ID)
    .then((result) => {
      if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
      content.value = result
    })
    .catch((e) => {
      if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
      contentError.value = e instanceof Error ? e.message : 'Không thể tải nội dung chương.'
    })
    .finally(() => {
      if (mayCommit() && activeChapter.value?.ID === chapter.ID) contentLoading.value = false
    })

  const audioRequest = getAudioURL(chapter.ID)
    .then((result) => {
      if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
      audioChapterID.value = chapter.ID
      audioURL.value = result.url
      audioError.value = ''
    })
    .catch((e) => {
      if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
      audioError.value = e instanceof Error ? e.message : 'Không thể tải audio chương này.'
    })
    .finally(() => {
      if (mayCommit() && activeChapter.value?.ID === chapter.ID) audioLoading.value = false
    })

  await Promise.all([contentRequest, audioRequest])
  if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return

  try {
    await listener.loadProgress(chapter.ID)
    if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
    const saved = listener.progress[chapter.ID]
    if (saved && saved.PositionMs > 0) {
      await nextTick()
      if (!mayCommit() || activeChapter.value?.ID !== chapter.ID || audioChapterID.value !== chapter.ID) return
      const el = audioEl.value
      if (el) {
        el.currentTime = saved.PositionMs / 1000
        currentTime.value = el.currentTime
      }
    }
  } catch {
    // Guest progress remains local; unavailable authenticated progress must not
    // prevent the reader from opening public content.
  }
}

async function retryAudio() {
  const chapter = activeChapter.value
  if (!chapter || audioLoading.value) return
  const requestChapterID = chapter.ID
  const mayCommit = activeSelectionMayCommit
  audioLoading.value = true
  audioError.value = ''
  try {
    const result = await getAudioURL(requestChapterID)
    if (!mayCommit() || activeChapter.value?.ID !== requestChapterID) return
    audioChapterID.value = requestChapterID
    audioURL.value = result.url
    await nextTick()
    if (mayCommit() && activeChapter.value?.ID === requestChapterID && audioChapterID.value === requestChapterID) audioEl.value?.load()
  } catch (e) {
    if (mayCommit() && activeChapter.value?.ID === requestChapterID) {
      audioError.value = e instanceof Error ? e.message : 'Không thể tải audio chương này.'
    }
  } finally {
    if (mayCommit() && activeChapter.value?.ID === requestChapterID) audioLoading.value = false
  }
}

function markProgressState(state: 'idle' | 'saving' | 'saved' | 'error') {
  progressState.value = state
  if (progressStateTimer) window.clearTimeout(progressStateTimer)
  if (state === 'saved') {
    progressStateTimer = window.setTimeout(() => {
      progressState.value = 'idle'
    }, 2400)
  }
}

function persistCurrentPosition(force = false) {
  const el = audioEl.value
  const chapter = activeChapter.value
  if (!el || !chapter || audioChapterID.value !== chapter.ID) return

  const chapterID = chapter.ID
  const now = Date.now()
  if (!force && now - lastProgressWriteAt < progressWriteIntervalMs) return
  const positionMs = Math.max(0, Math.floor(el.currentTime * 1000))
  lastProgressWriteAt = now
  markProgressState('saving')

  progressWrite = progressWrite
    .catch(() => undefined)
    .then(() => listener.saveProgress(chapterID, positionMs, ''))
    .then(() => {
      if (activeChapter.value?.ID === chapterID && audioChapterID.value === chapterID) markProgressState('saved')
    })
    .catch(() => {
      if (activeChapter.value?.ID === chapterID && audioChapterID.value === chapterID) markProgressState('error')
    })
}

async function togglePlayback() {
  const el = audioEl.value
  if (!el || !audioURL.value || !ownsCurrentAudio()) return
  audioError.value = ''
  try {
    if (el.paused) {
      isBuffering.value = true
      await el.play()
    } else {
      el.pause()
    }
  } catch {
    isBuffering.value = false
    audioError.value = 'Không thể bắt đầu phát audio. Hãy thử lại hoặc kiểm tra kết nối mạng.'
    isPlaying.value = false
  }
}

function seekTo(value: number, persist = false) {
  const el = audioEl.value
  if (!el || !ownsCurrentAudio() || !Number.isFinite(value)) return
  const next = Math.max(0, Math.min(value, duration.value || value))
  el.currentTime = next
  currentTime.value = next
  if (persist) persistCurrentPosition(true)
}

function seekBy(deltaSeconds: number) {
  seekTo(currentTime.value + deltaSeconds, true)
}

function changePlaybackRate(value: unknown) {
  const rate = normalizePlaybackRate(value)
  playbackRate.value = rate
  if (audioEl.value && ownsCurrentAudio()) audioEl.value.playbackRate = rate
}

function changeVolume(value: unknown) {
  const next = normalizeVolume(value)
  volume.value = next
  if (next > 0) {
    lastAudibleVolume.value = next
    isMuted.value = false
  } else {
    isMuted.value = true
  }
  if (audioEl.value) {
    audioEl.value.volume = next
    audioEl.value.muted = isMuted.value
  }
}

function toggleMute() {
  const state = toggleMuteState(isMuted.value, volume.value, lastAudibleVolume.value)
  isMuted.value = state.muted
  volume.value = state.volume
  lastAudibleVolume.value = state.lastAudibleVolume
  if (audioEl.value) {
    audioEl.value.volume = state.volume
    audioEl.value.muted = state.muted
  }
}

function setSleepTimer(value: string) {
  if (sleepTimerID) window.clearTimeout(sleepTimerID)
  sleepTimerID = undefined
  sleepTimer.value = value === '15' || value === '30' || value === '60' ? value : 'off'
  if (sleepTimer.value === 'off') return
  sleepTimerID = window.setTimeout(() => {
    audioEl.value?.pause()
    sleepTimer.value = 'off'
    sleepTimerID = undefined
  }, Number(sleepTimer.value) * 60_000)
}

function onLoadedMetadata() {
  const el = audioEl.value
  if (!el || !ownsCurrentAudio()) return
  duration.value = Number.isFinite(el.duration) ? el.duration : 0
  currentTime.value = el.currentTime
  el.playbackRate = playbackRate.value
  el.volume = volume.value
  el.muted = isMuted.value
}

function onAudioLoadStart() {
  if (!ownsCurrentAudio()) return
  isBuffering.value = true
}

function onAudioPlaying() {
  if (!ownsCurrentAudio()) return
  isPlaying.value = true
  isBuffering.value = false
}

function onAudioCanPlay() {
  if (!ownsCurrentAudio()) return
  isBuffering.value = false
}

function onTimeUpdate() {
  const el = audioEl.value
  if (!el || !ownsCurrentAudio()) return
  currentTime.value = el.currentTime
  persistCurrentPosition(false)
}

function onPauseOrSeek() {
  const el = audioEl.value
  if (!el || !ownsCurrentAudio()) return
  currentTime.value = el.currentTime
  persistCurrentPosition(true)
}

function onAudioError() {
  if (!ownsCurrentAudio()) return
  isPlaying.value = false
  isBuffering.value = false
  audioLoading.value = false
  audioError.value = 'Không thể phát audio lúc này. Hãy thử tải lại audio.'
}

async function onEnded() {
  const chapter = activeChapter.value
  if (!chapter || audioChapterID.value !== chapter.ID) return
  const chapterID = chapter.ID
  isPlaying.value = false
  persistCurrentPosition(true)
  if (listener.isGuest) return
  try {
    // Completion must be ordered after the final position write; otherwise a
    // brand-new progress row could race the completion mutation and return 404.
    await progressWrite
    if (activeChapter.value?.ID !== chapterID || audioChapterID.value !== chapterID) return
    const completed = await completeProgress(chapterID)
    if (activeChapter.value?.ID === chapterID && audioChapterID.value === chapterID) listener.progress[chapterID] = completed
  } catch {
    // The final persisted position is still useful if completion marking is
    // temporarily unavailable; the next interaction can retry naturally.
  }
}

function onPageHide() {
  persistCurrentPosition(true)
}

async function toggleFavorite() {
  await listener.toggleFavorite(storyID.value)
}

onMounted(async () => {
  window.addEventListener('pagehide', onPageHide)
  await listener.loadFavorites()
  await loadChapters()
})

onBeforeUnmount(() => {
  persistCurrentPosition(true)
  if (progressStateTimer) window.clearTimeout(progressStateTimer)
  if (sleepTimerID) window.clearTimeout(sleepTimerID)
  window.removeEventListener('pagehide', onPageHide)
})
</script>

<template>
  <section class="page reader">
    <RouterLink class="back-link" :to="`/stories/${storyID}`">← Về chi tiết truyện</RouterLink>
    <div class="reader-head">
      <div>
        <p class="eyebrow">Synaudio / immersive listening</p>
        <h1>{{ activeChapter?.Title || 'Chọn một chương để bắt đầu' }}</h1>
        <p class="reader-story-context">Đọc cùng nhịp kể chuyện. Tiến độ sẽ được lưu tự động trên thiết bị của bạn.</p>
      </div>
      <button
        class="fav-btn"
        :class="{ active: listener.isFavorite(storyID) }"
        type="button"
        :aria-pressed="listener.isFavorite(storyID)"
        @click="toggleFavorite"
      >
        {{ listener.isFavorite(storyID) ? '★ Đã yêu thích' : '☆ Yêu thích' }}
      </button>
    </div>

    <p v-if="loading" class="status-state" role="status" aria-live="polite">Đang mở thư viện chương...</p>
    <div v-else-if="error" class="status-state error" role="alert">
      <strong>Không thể tải các chương.</strong>
      <p>{{ error }}</p>
      <button class="secondary-link" type="button" @click="loadChapters">Thử lại</button>
    </div>
    <p v-else-if="chapters.length === 0" class="note">Chưa có chương nào được xuất bản.</p>

    <template v-else>
      <button class="chapter-drawer-toggle" type="button" :aria-expanded="chapterDrawerOpen" @click="chapterDrawerOpen = !chapterDrawerOpen">
        <span><span class="drawer-toggle-dot" aria-hidden="true"></span> Danh sách chương</span>
        <span>{{ chapters.length }} chương · {{ chapterDrawerOpen ? 'Đóng' : 'Mở' }}</span>
      </button>
      <div class="reader-layout">
        <nav class="chapter-nav" :class="{ 'drawer-open': chapterDrawerOpen }" aria-labelledby="chapter-nav-heading">
          <div class="chapter-nav-heading">
            <div>
              <p class="eyebrow">Playlist</p>
              <h2 id="chapter-nav-heading">Các chương</h2>
            </div>
            <div class="chapter-nav-meta">
              <span class="chapter-count">{{ chapters.length }}</span>
              <button class="drawer-close" type="button" aria-label="Đóng danh sách chương" @click="chapterDrawerOpen = false">×</button>
            </div>
          </div>
          <div class="chapter-nav-list">
            <button
              v-for="c in chapters"
              :key="c.ID"
              class="chapter-tab"
              :class="{ active: activeChapter?.ID === c.ID }"
              type="button"
              :aria-current="activeChapter?.ID === c.ID ? 'page' : undefined"
              @click="selectChapter(c)"
            >
              <span class="chapter-number">Chương {{ c.ChapterNumber }}</span>
              <strong>{{ c.Title }}</strong>
              <span v-if="activeChapter?.ID === c.ID" class="chapter-current">Đang nghe</span>
            </button>
          </div>
        </nav>

        <div v-if="activeChapter" class="reader-body">
          <p v-if="contentLoading || audioLoading" class="muted loading-line" role="status" aria-live="polite">
            {{ contentLoading ? 'Đang tải nội dung' : '' }}{{ contentLoading && audioLoading ? ' · ' : '' }}{{ audioLoading ? 'Đang chuẩn bị audio' : '' }}...
          </p>

          <section class="audio-section" aria-labelledby="audio-heading">
            <div class="audio-heading-row">
              <div>
                <p class="eyebrow">Now playing · chapter {{ activeChapter.ChapterNumber }}</p>
                <h2 id="audio-heading">Nghe chương này</h2>
              </div>
              <span v-if="progressLabel" class="save-state" role="status" aria-live="polite">{{ progressLabel }}</span>
            </div>

            <audio
              v-if="audioURL"
              ref="audioEl"
              :key="`${audioChapterID}:${audioURL}`"
              class="native-audio"
              :src="audioURL"
              preload="metadata"
              @loadstart="onAudioLoadStart"
              @loadedmetadata="onLoadedMetadata"
              @timeupdate="onTimeUpdate"
              @play="isPlaying = ownsCurrentAudio()"
              @playing="onAudioPlaying"
              @pause="isPlaying = false; onPauseOrSeek()"
              @seeking="isBuffering = ownsCurrentAudio()"
              @seeked="isBuffering = false; onPauseOrSeek()"
              @waiting="isBuffering = ownsCurrentAudio()"
              @stalled="isBuffering = ownsCurrentAudio()"
              @canplay="onAudioCanPlay"
              @error="onAudioError"
              @ended="onEnded"
            />

            <div v-if="audioURL" class="player-shell" :aria-busy="isBuffering" :class="{ buffering: isBuffering }">
              <div class="player-primary">
                <button class="skip-button" type="button" aria-label="Lùi 15 giây" title="Lùi 15 giây" @click="seekBy(-15)">
                  <span aria-hidden="true">↶</span><span>15s</span>
                </button>
                <button
                  class="play-button"
                  type="button"
                  :aria-label="isPlaying ? 'Tạm dừng' : 'Phát audio'"
                  :aria-busy="isBuffering"
                  @click="togglePlayback"
                >
                  <span v-if="isBuffering" class="spinner player-spinner" aria-hidden="true"></span>
                  <svg v-else-if="isPlaying" aria-hidden="true" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="5" width="4" height="14" rx="1"></rect><rect x="14" y="5" width="4" height="14" rx="1"></rect></svg>
                  <svg v-else aria-hidden="true" viewBox="0 0 24 24" fill="currentColor"><polygon points="7 4 19 12 7 20 7 4"></polygon></svg>
                </button>
                <button class="skip-button" type="button" aria-label="Tua tới 15 giây" title="Tua tới 15 giây" @click="seekBy(15)">
                  <span>15s</span><span aria-hidden="true">↷</span>
                </button>
                <div class="now-playing">
                  <strong>{{ activeChapter.Title }}</strong>
                  <span><span v-if="isBuffering" class="spinner inline-spinner" aria-hidden="true"></span>{{ isBuffering ? 'Đang tải audio…' : isPlaying ? 'Đang phát' : 'Sẵn sàng nghe' }}</span>
                </div>
              </div>

              <div class="timeline-row">
                <span>{{ formatPlaybackTime(currentTime) }}</span>
                <input
                  class="timeline"
                  type="range"
                  min="0"
                  :max="Math.max(duration, 0)"
                  step="1"
                  :value="currentTime"
                  :style="{ '--playback-progress': `${playbackPercent}%` }"
                  :aria-label="`Tiến độ audio ${Math.round(playbackPercent)}%`"
                  @input="seekTo(Number(($event.target as HTMLInputElement).value))"
                >
                <span>{{ formatPlaybackTime(duration) }}</span>
              </div>

              <div class="player-secondary">
                <label class="rate-control">
                  <span>Tốc độ</span>
                  <select :value="playbackRate" aria-label="Tốc độ phát" @change="changePlaybackRate(($event.target as HTMLSelectElement).value)">
                    <option v-for="rate in playbackRates" :key="rate" :value="rate">{{ rate }}×</option>
                  </select>
                </label>
                <label class="volume-control">
                  <span class="volume-label">Âm lượng</span>
                  <button
                    class="mute-button"
                    type="button"
                    :aria-pressed="isMuted"
                    :aria-label="isMuted ? 'Bật âm lượng' : 'Tắt âm lượng'"
                    :title="isMuted ? 'Bật âm lượng' : 'Tắt âm lượng'"
                    @click="toggleMute"
                  >
                    <svg v-if="isMuted" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 9 4.5 4.5"></path><path d="m13.5 9-4.5 4.5"></path><path d="M4 9v6h4l5 4V5L8 9H4z"></path><path d="m19 9-4 6"></path><path d="m15 9 4 6"></path></svg>
                    <svg v-else aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 9v6h4l5 4V5L8 9H4z"></path><path d="M17 9.5a4 4 0 0 1 0 5"></path><path d="M19.5 7a7 7 0 0 1 0 10"></path></svg>
                  </button>
                  <input
                    type="range"
                    min="0"
                    max="1"
                    step="0.05"
                    :value="volume"
                    :aria-valuetext="`${Math.round(volume * 100)}%`"
                    aria-label="Âm lượng"
                    @input="changeVolume(($event.target as HTMLInputElement).value)"
                  >
                </label>
                <label class="sleep-control">
                  <span>Hẹn giờ</span>
                  <select :value="sleepTimer" aria-label="Hẹn giờ tắt" @change="setSleepTimer(($event.target as HTMLSelectElement).value)">
                    <option value="off">Tắt</option>
                    <option value="15">15 phút</option>
                    <option value="30">30 phút</option>
                    <option value="60">60 phút</option>
                  </select>
                </label>
                <span class="player-progress-text">{{ Math.round(playbackPercent) }}% chương</span>
              </div>
            </div>

            <div v-else-if="audioLoading" class="player-placeholder" role="status">Đang chuẩn bị audio…</div>
          </section>

          <div
            v-if="listener.progress[activeChapter.ID]?.RelistenStatus && listener.progress[activeChapter.ID]?.RelistenStatus !== 'NO_RELISTEN_NEEDED'"
            class="relisten-notice"
            role="status"
          >
            <strong>
              {{ listener.progress[activeChapter.ID]?.RelistenStatus === 'RELISTEN_REQUIRED' ? 'Nên nghe lại chương này' : 'Có bản cập nhật cho chương này' }}
            </strong>
            <span>Tiến độ nghe trước đây vẫn được giữ nguyên.</span>
          </div>

          <div v-if="audioError" class="status-state audio-state" role="alert">
            <strong>Audio tạm thời chưa sẵn sàng.</strong>
            <p>{{ audioError }}</p>
            <div class="error-actions">
              <button type="button" @click="retryAudio">Thử lại</button>
              <span class="muted">Bạn vẫn có thể đọc nội dung chương này.</span>
            </div>
          </div>

          <div v-if="contentError" class="status-state error" role="alert">
            <strong>Không thể tải nội dung chương.</strong>
            <p>{{ contentError }}</p>
          </div>
          <article v-else-if="content" class="prose" aria-label="Nội dung chương">
            <p v-for="(para, i) in content.content_text.split(/\n+/)" :key="i">{{ para }}</p>
          </article>

          <nav v-if="prevChapter || nextChapter" class="reader-footer-nav" aria-label="Chuyển chương">
            <button
              v-if="prevChapter"
              type="button"
              class="reader-nav-btn prev"
              @click="selectChapter(prevChapter)"
            >
              ← Chương {{ prevChapter.ChapterNumber }}: {{ prevChapter.Title }}
            </button>
            <div v-else class="nav-spacer"></div>
            <button
              v-if="nextChapter"
              type="button"
              class="reader-nav-btn next"
              @click="selectChapter(nextChapter)"
            >
              Chương {{ nextChapter.ChapterNumber }}: {{ nextChapter.Title }} →
            </button>
          </nav>
        </div>
      </div>
    </template>
  </section>
</template>

<style scoped>
.reader { max-width: var(--content-max); margin: 0 auto; }
.reader-head, .chapter-nav-heading, .audio-heading-row, .player-primary, .player-secondary { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.reader-head { margin: 18px 0 24px; align-items: flex-start; }
.reader-head h1 { margin: 4px 0 0; overflow-wrap: anywhere; }
.eyebrow { margin: 0; font-size: 12px; font-weight: 800; letter-spacing: .1em; text-transform: uppercase; opacity: .68; }
.reader-layout { display: grid; grid-template-columns: minmax(230px, 290px) minmax(0, 1fr); gap: 24px; align-items: start; }
.chapter-nav { position: sticky; top: 20px; border: 1px solid var(--line); border-radius: var(--radius-lg); padding: 16px; max-height: calc(100vh - 40px); overflow: hidden; background: var(--surface); }
.chapter-nav-heading h2 { margin: 3px 0 0; }
.chapter-count { min-width: 30px; height: 30px; display: inline-grid; place-items: center; border-radius: var(--radius-full); background: var(--accent-soft); color: var(--accent-strong); font-weight: 700; }
.chapter-nav-list { display: grid; gap: 8px; margin-top: 14px; max-height: calc(100vh - 130px); overflow: auto; padding-right: 4px; }
.chapter-tab { min-height: 56px; width: 100%; display: grid; gap: 4px; text-align: left; padding: 11px 12px; border: 1px solid transparent; border-radius: var(--radius-md); background: transparent; color: inherit; cursor: pointer; }
.chapter-tab:hover { background: var(--surface-soft); }
.chapter-tab.active { border-color: var(--accent); background: var(--accent-soft); }
.chapter-number, .chapter-current { font-size: 12px; color: var(--muted); }
.chapter-current { font-weight: 800; color: var(--accent-strong); }
.reader-body { min-width: 0; display: grid; gap: 18px; }
.loading-line { min-height: 24px; }
.audio-section { border: 1px solid var(--accent-border); border-radius: var(--radius-lg); padding: 20px; display: grid; gap: 16px; background: radial-gradient(circle at 82% 12%, rgba(217, 119, 6, 0.14), transparent 16rem), var(--surface); box-shadow: var(--shadow-lg); }
.audio-heading-row h2 { margin: 3px 0 0; }
.save-state { font-size: 13px; font-weight: 700; color: var(--muted); }
.native-audio { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
.player-shell { display: grid; gap: 18px; }
.player-shell.buffering { cursor: progress; }
.player-primary { justify-content: flex-start; }
.play-button, .skip-button, .fav-btn, .error-actions button { min-width: 44px; min-height: 44px; border-radius: var(--radius-full); border: 1px solid var(--accent); background: transparent; color: var(--accent-strong); cursor: pointer; }
.play-button { width: 58px; height: 58px; font-size: 22px; background: var(--accent); color: var(--surface); }
.play-button svg { width: 24px; height: 24px; }
.play-button .player-spinner { width: 22px; height: 22px; margin: 0; border-width: 3px; border-color: rgba(16, 21, 29, .32); border-top-color: var(--surface); }
.skip-button { padding: 0 12px; font-weight: 800; }
.skip-button span { display: inline-flex; align-items: center; }
.now-playing { min-width: 0; display: grid; gap: 3px; }
.now-playing strong { overflow-wrap: anywhere; }
.now-playing span, .player-progress-text { font-size: 13px; color: var(--muted); }
.inline-spinner { width: .8rem; height: .8rem; margin-right: .35rem; border-width: 2px; vertical-align: -0.12rem; }
.timeline-row { display: grid; grid-template-columns: max-content minmax(0, 1fr) max-content; gap: 10px; align-items: center; font-variant-numeric: tabular-nums; font-size: 13px; }
.timeline { --playback-progress: 0%; width: 100%; min-height: 44px; appearance: none; cursor: pointer; accent-color: var(--accent); background: transparent; }
.timeline::-webkit-slider-runnable-track { height: 6px; border-radius: var(--radius-full); background: linear-gradient(90deg, var(--accent) var(--playback-progress), rgba(255, 255, 255, 0.13) var(--playback-progress)); }
.timeline::-moz-range-track { height: 6px; border-radius: var(--radius-full); background: linear-gradient(90deg, var(--accent) var(--playback-progress), rgba(255, 255, 255, 0.13) var(--playback-progress)); }
.timeline::-webkit-slider-thumb { width: 18px; height: 18px; margin-top: -6px; appearance: none; border: 2px solid var(--accent-strong); border-radius: 50%; background: var(--surface); box-shadow: 0 0 0 4px var(--accent-soft); }
.timeline::-moz-range-thumb { width: 14px; height: 14px; border: 2px solid var(--accent-strong); border-radius: 50%; background: var(--surface); box-shadow: 0 0 0 4px var(--accent-soft); }
.rate-control { display: flex; align-items: center; gap: 8px; font-weight: 700; }
.rate-control select { min-height: 44px; border: 1px solid var(--line); border-radius: var(--radius-full); padding: 0 2.75rem 0 14px; background-color: var(--surface); color: var(--ink); }
.volume-control { display: flex; align-items: center; gap: 8px; min-width: 0; }
.volume-label { white-space: nowrap; }
.mute-button { display: grid; width: 44px; min-width: 44px; height: 44px; place-items: center; padding: 0; border: 1px solid var(--line); border-radius: var(--radius-full); background: var(--surface); color: var(--muted); cursor: pointer; }
.mute-button:hover, .mute-button[aria-pressed="true"] { border-color: var(--accent-border); background: var(--accent-soft); color: var(--accent-strong); }
.mute-button svg { width: 18px; height: 18px; }
.player-placeholder { min-height: 112px; display: grid; place-items: center; border-radius: var(--radius-md); background: var(--surface-soft); }
.relisten-notice, .status-state { border-radius: var(--radius-md); padding: 14px 16px; }
.relisten-notice { display: grid; gap: 4px; border: 1px solid var(--accent-border); border-left: 4px solid var(--amber); background: var(--accent-soft); color: var(--ink); }
.error-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; }
.error-actions button { padding: 0 16px; border-radius: var(--radius-full); }
.prose { max-width: 76ch; font-family: var(--font-reading); font-size: 1.05rem; line-height: 1.78; overflow-wrap: anywhere; }
.prose p { margin: 0 0 1.1em; }

.reader-footer-nav {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-top: 2.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--line);
}

.reader-nav-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 48px;
  padding: 0.65rem 1.25rem;
  border: 1px solid var(--line);
  border-radius: var(--radius-full);
  background: var(--surface);
  color: var(--ink);
  font-family: var(--font-heading);
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 160ms ease;
  box-shadow: var(--shadow);
}

.reader-nav-btn:hover {
  border-color: var(--accent);
  color: var(--accent);
  background: var(--surface-soft);
  transform: translateY(-1px);
}

.reader-nav-btn.next {
  margin-left: auto;
  background: var(--accent-soft);
  border-color: var(--accent-border);
  color: var(--accent);
}

.nav-spacer {
  flex: 1;
}

@media (max-width: 1024px) {
  .reader-layout { grid-template-columns: minmax(210px, 250px) minmax(0, 1fr); gap: 18px; }
  .chapter-nav { position: static; max-height: none; }
  .chapter-nav-list { max-height: 520px; }
}

@media (max-width: 760px) {
  .reader-head { align-items: stretch; flex-direction: column; margin: 12px 0 16px; }
  .reader-head .fav-btn { align-self: flex-start; }
  .reader-layout { grid-template-columns: 1fr; gap: 16px; }
  .chapter-nav { padding: 12px; }
  .chapter-nav-list { display: flex; max-height: none; overflow-x: auto; overflow-y: hidden; scroll-snap-type: x proximity; padding-bottom: 4px; gap: 8px; }
  .chapter-tab { flex: 0 0 min(75vw, 260px); scroll-snap-align: start; min-height: 56px; padding: 10px 12px; }
  .audio-section {
    position: sticky;
    top: calc(60px + env(safe-area-inset-top));
    z-index: 30;
    padding: 16px;
    border-radius: var(--radius-lg);
    background: rgba(16, 21, 29, 0.96);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    box-shadow: 0 8px 24px rgba(35, 28, 22, 0.08);
  }
  .audio-heading-row { margin-bottom: 4px; }
  .player-primary { display: grid; grid-template-columns: auto auto auto; justify-content: center; gap: 16px; }
  .now-playing { grid-column: 1 / -1; text-align: center; }
  .timeline-row { grid-template-columns: max-content minmax(0, 1fr) max-content; }
  .player-secondary { align-items: center; justify-content: space-between; }
  .prose { font-size: 1.08rem; line-height: 1.82; padding: 12px 0; }
  .reader-footer-nav { flex-direction: column; align-items: stretch; }
  .reader-nav-btn { width: 100%; justify-content: center; text-align: center; }
}

@media (max-width: 430px) {
  .reader-head { margin-top: 10px; }
  .audio-section { margin-inline: -8px; border-radius: var(--radius-md); padding: 14px; }
  .timeline-row { grid-template-columns: 1fr 1fr; gap: 4px; }
  .timeline { grid-column: 1 / -1; grid-row: 1; }
  .timeline-row span:last-child { text-align: right; }
  .player-secondary { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); align-items: center; }
  .volume-control { justify-content: flex-end; }
  .sleep-control { justify-content: flex-start; }
  .player-progress-text { grid-column: 1 / -1; }
  .rate-control { justify-content: flex-start; }
}
</style>
