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
  normalizePlaybackRate,
} from './readerSession.mjs'

const route = useRoute()
const storyID = computed(() => route.params.storyID as string)
const listener = useListenerStore()

const chapters = ref<Chapter[]>([])
const activeChapter = ref<Chapter | null>(null)
const content = ref<ChapterContent | null>(null)
const audioURL = ref('')
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
const progressState = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')
const chapterSelection = createLatestChapterSelectionGuard()
const progressWriteIntervalMs = 15_000
const playbackRates = [0.75, 1, 1.25, 1.5, 1.75, 2]
let lastProgressWriteAt = 0
let progressWrite: Promise<void> = Promise.resolve()
let progressStateTimer: number | undefined

const playbackPercent = computed(() => duration.value > 0 ? Math.min(100, (currentTime.value / duration.value) * 100) : 0)
const progressLabel = computed(() => {
  if (progressState.value === 'saving') return 'Đang lưu tiến độ…'
  if (progressState.value === 'saved') return 'Đã lưu tiến độ'
  if (progressState.value === 'error') return 'Chưa lưu được tiến độ'
  return ''
})

function resetPlayerState() {
  isPlaying.value = false
  isBuffering.value = false
  currentTime.value = 0
  duration.value = 0
  progressState.value = 'idle'
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
  persistCurrentPosition(true)
  const mayCommit = chapterSelection.begin(chapter.ID)
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
      if (!mayCommit() || activeChapter.value?.ID !== chapter.ID) return
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
  audioLoading.value = true
  audioError.value = ''
  try {
    const result = await getAudioURL(chapter.ID)
    if (activeChapter.value?.ID !== chapter.ID) return
    audioURL.value = result.url
    await nextTick()
    audioEl.value?.load()
  } catch (e) {
    if (activeChapter.value?.ID === chapter.ID) {
      audioError.value = e instanceof Error ? e.message : 'Không thể tải audio chương này.'
    }
  } finally {
    if (activeChapter.value?.ID === chapter.ID) audioLoading.value = false
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
  if (!el || !chapter) return

  const now = Date.now()
  if (!force && now - lastProgressWriteAt < progressWriteIntervalMs) return
  const positionMs = Math.max(0, Math.floor(el.currentTime * 1000))
  lastProgressWriteAt = now
  markProgressState('saving')

  progressWrite = progressWrite
    .catch(() => undefined)
    .then(() => listener.saveProgress(chapter.ID, positionMs, ''))
    .then(() => markProgressState('saved'))
    .catch(() => markProgressState('error'))
}

async function togglePlayback() {
  const el = audioEl.value
  if (!el || !audioURL.value) return
  audioError.value = ''
  try {
    if (el.paused) {
      await el.play()
    } else {
      el.pause()
    }
  } catch (e) {
    audioError.value = e instanceof Error ? e.message : 'Không thể bắt đầu phát audio.'
    isPlaying.value = false
  }
}

function seekTo(value: number) {
  const el = audioEl.value
  if (!el || !Number.isFinite(value)) return
  const next = Math.max(0, Math.min(value, duration.value || value))
  el.currentTime = next
  currentTime.value = next
  persistCurrentPosition(true)
}

function seekBy(deltaSeconds: number) {
  seekTo(currentTime.value + deltaSeconds)
}

function changePlaybackRate(value: unknown) {
  const rate = normalizePlaybackRate(value)
  playbackRate.value = rate
  if (audioEl.value) audioEl.value.playbackRate = rate
}

function onLoadedMetadata() {
  const el = audioEl.value
  if (!el) return
  duration.value = Number.isFinite(el.duration) ? el.duration : 0
  currentTime.value = el.currentTime
  el.playbackRate = playbackRate.value
}

function onTimeUpdate() {
  const el = audioEl.value
  if (el) currentTime.value = el.currentTime
  persistCurrentPosition(false)
}

function onPauseOrSeek() {
  const el = audioEl.value
  if (el) currentTime.value = el.currentTime
  persistCurrentPosition(true)
}

function onAudioError() {
  isPlaying.value = false
  isBuffering.value = false
  audioError.value = 'Không thể phát audio lúc này. Hãy thử tải lại audio.'
}

async function onEnded() {
  isPlaying.value = false
  persistCurrentPosition(true)
  const chapter = activeChapter.value
  if (!chapter || listener.isGuest) return
  try {
    // Completion must be ordered after the final position write; otherwise a
    // brand-new progress row could race the completion mutation and return 404.
    await progressWrite
    const completed = await completeProgress(chapter.ID)
    listener.progress[chapter.ID] = completed
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
  window.removeEventListener('pagehide', onPageHide)
})
</script>

<template>
  <section class="page reader">
    <RouterLink class="back-link" :to="`/stories/${storyID}`">← Về chi tiết truyện</RouterLink>
    <div class="reader-head">
      <div>
        <p class="eyebrow">Đọc & nghe · Synaudio</p>
        <h1>{{ activeChapter?.Title || 'Chọn một chương để bắt đầu' }}</h1>
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
      <div class="reader-layout">
        <nav class="chapter-nav" aria-labelledby="chapter-nav-heading">
          <div class="chapter-nav-heading">
            <div>
              <p class="eyebrow">Thư viện</p>
              <h2 id="chapter-nav-heading">Các chương</h2>
            </div>
            <span class="chapter-count">{{ chapters.length }}</span>
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
                <p class="eyebrow">Audiobook player</p>
                <h2 id="audio-heading">Nghe chương này</h2>
              </div>
              <span v-if="progressLabel" class="save-state" role="status" aria-live="polite">{{ progressLabel }}</span>
            </div>

            <audio
              v-if="audioURL"
              ref="audioEl"
              class="native-audio"
              :src="audioURL"
              preload="metadata"
              @loadedmetadata="onLoadedMetadata"
              @timeupdate="onTimeUpdate"
              @play="isPlaying = true"
              @playing="isBuffering = false"
              @pause="isPlaying = false; onPauseOrSeek()"
              @seeking="isBuffering = true"
              @seeked="isBuffering = false; onPauseOrSeek()"
              @waiting="isBuffering = true"
              @canplay="isBuffering = false"
              @error="onAudioError"
              @ended="onEnded"
            />

            <div v-if="audioURL" class="player-shell" :aria-busy="isBuffering">
              <div class="player-primary">
                <button class="skip-button" type="button" aria-label="Lùi 15 giây" @click="seekBy(-15)">−15s</button>
                <button class="play-button" type="button" :aria-label="isPlaying ? 'Tạm dừng' : 'Phát audio'" @click="togglePlayback">
                  {{ isPlaying ? '❚❚' : '▶' }}
                </button>
                <button class="skip-button" type="button" aria-label="Tua tới 30 giây" @click="seekBy(30)">+30s</button>
                <div class="now-playing">
                  <strong>{{ activeChapter.Title }}</strong>
                  <span>{{ isBuffering ? 'Đang tải audio…' : isPlaying ? 'Đang phát' : 'Sẵn sàng nghe' }}</span>
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
              <button type="button" @click="retryAudio">Thử tải lại audio</button>
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
        </div>
      </div>
    </template>
  </section>
</template>

<style scoped>
.reader { max-width: 1280px; margin: 0 auto; }
.reader-head, .chapter-nav-heading, .audio-heading-row, .player-primary, .player-secondary { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.reader-head { margin: 18px 0 24px; align-items: flex-start; }
.reader-head h1 { margin: 4px 0 0; overflow-wrap: anywhere; }
.eyebrow { margin: 0; font-size: 12px; font-weight: 800; letter-spacing: .1em; text-transform: uppercase; opacity: .68; }
.reader-layout { display: grid; grid-template-columns: minmax(230px, 290px) minmax(0, 1fr); gap: 24px; align-items: start; }
.chapter-nav { position: sticky; top: 20px; border: 1px solid var(--border-color, #d8d8d8); border-radius: 18px; padding: 16px; max-height: calc(100vh - 40px); overflow: hidden; }
.chapter-nav-heading h2 { margin: 3px 0 0; }
.chapter-count { min-width: 30px; height: 30px; display: inline-grid; place-items: center; border-radius: 999px; background: color-mix(in srgb, currentColor 10%, transparent); font-weight: 700; }
.chapter-nav-list { display: grid; gap: 8px; margin-top: 14px; max-height: calc(100vh - 130px); overflow: auto; padding-right: 4px; }
.chapter-tab { min-height: 56px; width: 100%; display: grid; gap: 4px; text-align: left; padding: 11px 12px; border: 1px solid transparent; border-radius: 12px; background: transparent; color: inherit; cursor: pointer; }
.chapter-tab:hover { background: color-mix(in srgb, currentColor 5%, transparent); }
.chapter-tab:focus-visible, button:focus-visible, select:focus-visible, input:focus-visible, .back-link:focus-visible { outline: 3px solid currentColor; outline-offset: 3px; }
.chapter-tab.active { border-color: currentColor; background: color-mix(in srgb, currentColor 7%, transparent); }
.chapter-number, .chapter-current { font-size: 12px; opacity: .72; }
.chapter-current { font-weight: 800; opacity: 1; }
.reader-body { min-width: 0; display: grid; gap: 18px; }
.loading-line { min-height: 24px; }
.audio-section { border: 1px solid var(--border-color, #d8d8d8); border-radius: 20px; padding: 20px; display: grid; gap: 16px; }
.audio-heading-row h2 { margin: 3px 0 0; }
.save-state { font-size: 13px; font-weight: 700; }
.native-audio { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
.player-shell { display: grid; gap: 18px; }
.player-primary { justify-content: flex-start; }
.play-button, .skip-button, .fav-btn, .error-actions button { min-width: 44px; min-height: 44px; border-radius: 999px; border: 1px solid currentColor; background: transparent; color: inherit; cursor: pointer; }
.play-button { width: 58px; height: 58px; font-size: 22px; background: currentColor; color: Canvas; }
.skip-button { padding: 0 12px; font-weight: 800; }
.now-playing { min-width: 0; display: grid; gap: 3px; }
.now-playing strong { overflow-wrap: anywhere; }
.now-playing span, .player-progress-text { font-size: 13px; opacity: .72; }
.timeline-row { display: grid; grid-template-columns: max-content minmax(0, 1fr) max-content; gap: 10px; align-items: center; font-variant-numeric: tabular-nums; font-size: 13px; }
.timeline { width: 100%; min-height: 44px; cursor: pointer; accent-color: currentColor; }
.rate-control { display: flex; align-items: center; gap: 8px; font-weight: 700; }
.rate-control select { min-height: 44px; border: 1px solid var(--border-color, #d8d8d8); border-radius: 10px; padding: 0 10px; background: Canvas; color: CanvasText; }
.player-placeholder { min-height: 112px; display: grid; place-items: center; border-radius: 14px; background: color-mix(in srgb, currentColor 5%, transparent); }
.relisten-notice, .status-state { border-radius: 14px; padding: 14px 16px; }
.relisten-notice { display: grid; gap: 4px; border: 1px solid currentColor; }
.error-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; }
.error-actions button { padding: 0 16px; border-radius: 10px; }
.prose { max-width: 76ch; font-size: 1.05rem; line-height: 1.78; overflow-wrap: anywhere; }
.prose p { margin: 0 0 1.1em; }

@media (max-width: 1024px) {
  .reader-layout { grid-template-columns: minmax(210px, 250px) minmax(0, 1fr); gap: 18px; }
  .chapter-nav { position: static; max-height: none; }
  .chapter-nav-list { max-height: 520px; }
}

@media (max-width: 760px) {
  .reader-head { align-items: stretch; }
  .reader-head, .audio-heading-row { flex-direction: column; }
  .reader-head .fav-btn { align-self: flex-start; }
  .reader-layout { grid-template-columns: 1fr; }
  .chapter-nav { padding: 14px; }
  .chapter-nav-list { display: flex; max-height: none; overflow-x: auto; overflow-y: hidden; scroll-snap-type: x proximity; padding-bottom: 4px; }
  .chapter-tab { flex: 0 0 min(78vw, 280px); scroll-snap-align: start; min-height: 64px; }
  .audio-section { padding: 16px; }
  .player-primary { display: grid; grid-template-columns: auto auto auto; justify-content: center; }
  .now-playing { grid-column: 1 / -1; text-align: center; }
  .timeline-row { grid-template-columns: max-content minmax(0, 1fr) max-content; }
  .player-secondary { align-items: flex-end; }
}

@media (max-width: 430px) {
  .reader-head { margin-top: 14px; }
  .audio-section { margin-inline: -4px; border-radius: 16px; }
  .timeline-row { grid-template-columns: 1fr 1fr; }
  .timeline { grid-column: 1 / -1; grid-row: 1; }
  .timeline-row span:last-child { text-align: right; }
  .player-secondary { align-items: stretch; flex-direction: column; }
  .rate-control { justify-content: space-between; }
}
</style>
