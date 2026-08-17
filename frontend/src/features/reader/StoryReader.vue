<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
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
const error = ref('')

const audioEl = ref<HTMLAudioElement | null>(null)

async function loadChapters() {
  loading.value = true
  error.value = ''
  try {
    const res = await listPublishedChapters(storyID.value)
    chapters.value = res.chapters
    if (chapters.value.length > 0) {
      await selectChapter(chapters.value[0])
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải danh sách chương.'
  } finally {
    loading.value = false
  }
}

async function selectChapter(chapter: Chapter) {
  activeChapter.value = chapter
  content.value = null
  audioURL.value = ''
  try {
    const [c, a] = await Promise.all([
      getChapterContent(chapter.ID),
      getAudioURL(chapter.ID),
    ])
    content.value = c
    audioURL.value = a.url
    await listener.loadProgress(chapter.ID)
    const saved = listener.progress[chapter.ID]
    if (saved && saved.PositionMs > 0) {
      // resume position after audio metadata loads
      const el = audioEl.value
      if (el) {
        el.currentTime = saved.PositionMs / 1000
      }
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải nội dung chương.'
  }
}

function onTimeUpdate() {
  const el = audioEl.value
  if (!el || !activeChapter.value) return
  const positionMs = Math.floor(el.currentTime * 1000)
  listener.saveProgress(activeChapter.value.ID, positionMs, '')
}

function onPause() {
  onTimeUpdate()
}

async function toggleFavorite() {
  await listener.toggleFavorite(storyID.value)
}

onMounted(async () => {
  await listener.loadFavorites()
  await loadChapters()
})
</script>

<template>
  <section class="page reader">
    <p class="eyebrow">Đọc & nghe</p>
    <div class="reader-head">
      <h1>Chương truyện</h1>
      <button class="fav-btn" :class="{ active: listener.isFavorite(storyID) }" @click="toggleFavorite">
        {{ listener.isFavorite(storyID) ? '★ Đã yêu thích' : '☆ Yêu thích' }}
      </button>
    </div>

    <p v-if="loading" class="note">Đang tải...</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <p v-else-if="chapters.length === 0" class="note">Chưa có chương nào được xuất bản.</p>

    <template v-else>
      <nav class="chapter-nav">
        <button
          v-for="c in chapters"
          :key="c.ID"
          class="chapter-tab"
          :class="{ active: activeChapter?.ID === c.ID }"
          @click="selectChapter(c)"
        >
          {{ c.ChapterNumber }}. {{ c.Title }}
        </button>
      </nav>

      <div v-if="activeChapter" class="reader-body">
        <h2>{{ activeChapter.Title }}</h2>

        <audio
          v-if="audioURL"
          ref="audioEl"
          class="player"
          :src="audioURL"
          controls
          preload="metadata"
          @timeupdate="onTimeUpdate"
          @pause="onPause"
        />

        <article v-if="content" class="prose">
          <p v-for="(para, i) in content.content_text.split(/\n+/)" :key="i">{{ para }}</p>
        </article>
      </div>
    </template>
  </section>
</template>
