<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { getPublicStory, listPublishedChapters } from '../../api/client'
import type { Chapter, Story } from '../../api/types'

const route = useRoute()
const storyID = computed(() => route.params.storyID as string)
const story = ref<Story | null>(null)
const chapters = ref<Chapter[]>([])
const loading = ref(true)
const error = ref('')
const chaptersLoading = ref(false)
const chaptersError = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    story.value = await getPublicStory(storyID.value)
    await loadChapters()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải thông tin truyện.'
  } finally {
    loading.value = false
  }
}

async function loadChapters() {
  chaptersLoading.value = true
  chaptersError.value = ''
  try {
    const response = await listPublishedChapters(storyID.value)
    chapters.value = response.chapters
  } catch (e) {
    chaptersError.value = e instanceof Error ? e.message : 'Không thể tải danh sách chương.'
  } finally {
    chaptersLoading.value = false
  }
}

function storyStatusLabel(status: Story['status']) {
  const labels: Record<Story['status'], string> = {
    DRAFT: 'Bản nháp',
    ACTIVE: 'Đang phát hành',
    COMPLETED: 'Đã hoàn thành',
    ARCHIVED: 'Đã lưu trữ',
  }
  return labels[status]
}

onMounted(load)
</script>

<template>
  <section class="page story-detail">
    <RouterLink class="back-link" to="/">← Quay lại khám phá</RouterLink>

    <p v-if="loading" class="status-state" role="status" aria-live="polite">Đang tải thông tin truyện...</p>
    <div v-else-if="error" class="status-state error" role="alert">
      <strong>Không thể tải truyện.</strong>
      <p>{{ error }}</p>
      <button class="secondary-link" type="button" @click="load">Thử lại</button>
    </div>
    <div v-else-if="!story" class="empty-state">
      <strong>Không tìm thấy truyện.</strong>
      <p>Truyện có thể đã được chuyển khỏi thư viện công khai.</p>
      <RouterLink class="secondary-link" to="/">Về trang khám phá</RouterLink>
    </div>
    <template v-else>
      <div class="detail-hero">
        <div class="detail-art" aria-hidden="true">
          <span class="detail-art-letter">{{ story.title.slice(0, 1).toUpperCase() }}</span>
          <span class="detail-art-seal">文</span>
        </div>
        <div class="detail-copy">
          <div class="meta" aria-label="Trạng thái truyện">
            <span class="badge badge-accent">{{ storyStatusLabel(story.status) }}</span>
            <span class="badge">Audio original</span>
          </div>
          <h1>{{ story.title }}</h1>
          <p class="detail-description">
            {{ story.description || 'Một câu chuyện đang chờ bạn khám phá.' }}
          </p>
          <div class="detail-facts" aria-label="Thông tin nhanh">
            <div><strong>{{ chapters.length || '—' }}</strong><span>chương</span></div>
            <div><strong>AI voice</strong><span>narration</span></div>
            <div><strong>Sync</strong><span>nghe & đọc</span></div>
          </div>
          <div class="detail-actions">
            <RouterLink class="primary-btn-cta" :to="`/stories/${story.id}/read`">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <polygon points="5 3 19 12 5 21 5 3"></polygon>
              </svg>
              Bắt đầu nghe & đọc
            </RouterLink>
            <RouterLink class="secondary-btn-pill" to="/">
              Khám phá thêm
            </RouterLink>
          </div>
        </div>
      </div>

      <section class="chapter-preview" aria-labelledby="chapter-preview-heading">
        <div class="section-heading">
          <div>
            <h2 id="chapter-preview-heading">Danh sách chương</h2>
            <p class="muted">Chọn một chương để bắt đầu — tiến độ sẽ tự động được lưu.</p>
          </div>
          <span v-if="chapters.length > 0" class="muted badge-subtle">{{ chapters.length }} chương</span>
        </div>

        <p v-if="chaptersLoading" class="status-state" role="status" aria-live="polite">
          <span class="spinner" aria-hidden="true"></span> Đang tải danh sách chương...
        </p>
        <div v-else-if="chaptersError" class="status-state error" role="alert">
          <strong>Không thể tải danh sách chương.</strong>
          <p>{{ chaptersError }}</p>
          <button class="secondary-link" type="button" @click="loadChapters">Thử lại</button>
        </div>
        <p v-else-if="chapters.length === 0" class="empty-state">
          <strong>Chưa có chương nào được xuất bản.</strong>
        </p>
        <ol v-else class="chapter-list">
          <li v-for="chapter in chapters" :key="chapter.ID" class="chapter-row">
            <RouterLink
              class="chapter-row-link"
              :to="{ path: `/stories/${story.id}/read`, query: { chapter: chapter.ID } }"
            >
              <span class="chapter-number" aria-hidden="true">
                {{ String(chapter.ChapterNumber).padStart(2, '0') }}
              </span>
              <div class="chapter-copy">
                <h3>{{ chapter.Title }}</h3>
                <span class="chapter-sub">Chương {{ chapter.ChapterNumber }} • Sẵn sàng nghe</span>
              </div>
              <div class="chapter-play-icon" aria-hidden="true">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
                  <polygon points="5 3 19 12 5 21 5 3"></polygon>
                </svg>
              </div>
            </RouterLink>
          </li>
        </ol>
      </section>
    </template>
  </section>
</template>
