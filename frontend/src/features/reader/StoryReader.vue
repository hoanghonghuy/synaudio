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
const progressWriteIntervalMs = 15_000
let lastProgressWriteAt = 0
let progressWrite: Promise<void> = Promise.resolve()

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
  activeChapter.value = chapter
  lastProgressWriteAt = 0
  content.value = null
  audioURL.value = ''
  contentError.value = ''
  audioError.value = ''
  contentLoading.value = true
  audioLoading.value = true

  const contentRequest = getChapterContent(chapter.ID)
    .then((result) => {
      content.value = result
    })
    .catch((e) => {
      contentError.value = e instanceof Error ? e.message : 'Không thể tải nội dung chương.'
    })
    .finally(() => {
      contentLoading.value = false
    })

  const audioRequest = getAudioURL(chapter.ID)
    .then((result) => {
      audioURL.value = result.url
    })
    .catch((e) => {
      audioError.value = e instanceof Error ? e.message : 'Không thể tải audio chương này.'
    })
    .finally(() => {
      audioLoading.value = false
    })

  await Promise.all([contentRequest, audioRequest])

  try {
    await listener.loadProgress(chapter.ID)
    const saved = listener.progress[chapter.ID]
    if (saved && saved.PositionMs > 0) {
      await nextTick()
      const el = audioEl.value
      if (el) {
        el.currentTime = saved.PositionMs / 1000
      }
    }
  } catch {
    // Guest progress remains local; unavailable authenticated progress must not
    // prevent the reader from opening public content.
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

  progressWrite = progressWrite
    .catch(() => undefined)
    .then(() => listener.saveProgress(chapter.ID, positionMs, ''))
    .then(() => undefined)
    .catch(() => undefined)
}

function onTimeUpdate() {
  persistCurrentPosition(false)
}

function onPauseOrSeek() {
  persistCurrentPosition(true)
}

async function onEnded() {
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
          <h2 id="chapter-nav-heading">Các chương</h2>
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
              <span>{{ c.ChapterNumber }}.</span>
              <span>{{ c.Title }}</span>
              <span v-if="activeChapter?.ID === c.ID" class="chapter-current">Đang đọc</span>
            </button>
          </div>
        </nav>

        <div v-if="activeChapter" class="reader-body">
          <p v-if="contentLoading || audioLoading" class="muted" role="status" aria-live="polite">
            {{ contentLoading ? 'Đang tải nội dung' : '' }}{{ contentLoading && audioLoading ? ' · ' : '' }}{{ audioLoading ? 'Đang chuẩn bị audio' : '' }}...
          </p>

          <div class="audio-section">
            <h2>Nghe chương này</h2>
            <audio
              v-if="audioURL"
              ref="audioEl"
              class="player"
              :src="audioURL"
              controls
              preload="metadata"
              @timeupdate="onTimeUpdate"
              @pause="onPauseOrSeek"
              @seeked="onPauseOrSeek"
              @ended="onEnded"
            />
          </div>
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
          <div v-if="audioError" class="status-state audio-state" role="status">
            <strong>Audio tạm thời chưa sẵn sàng.</strong>
            <p>{{ audioError }}</p>
            <span class="muted">Bạn vẫn có thể đọc nội dung chương này.</span>
          </div>

          <div v-if="contentError" class="status-state error" role="alert">
            <strong>Không thể tải nội dung chương.</strong>
            <p>{{ contentError }}</p>
          </div>
          <article v-else-if="content" class="prose">
            <p v-for="(para, i) in content.content_text.split(/\n+/)" :key="i">{{ para }}</p>
          </article>
        </div>
      </div>
    </template>
  </section>
</template>