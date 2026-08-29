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
        <div class="detail-copy">
          <p class="eyebrow">Chi tiết truyện</p>
          <h1>{{ story.title }}</h1>
          <div class="meta" aria-label="Trạng thái truyện">
            <span class="badge">{{ storyStatusLabel(story.status) }}</span>
            <span class="badge">Công khai</span>
          </div>
          <p class="detail-description">
            {{ story.description || 'Một câu chuyện đang chờ bạn khám phá.' }}
          </p>
          <div class="detail-actions">
            <RouterLink class="primary-link" :to="`/stories/${story.id}/read`">
              Đọc & nghe <span aria-hidden="true">→</span>
            </RouterLink>
            <RouterLink class="secondary-link" to="/">Khám phá thêm</RouterLink>
          </div>
        </div>
        <div class="detail-art" aria-hidden="true">
          <span>{{ story.title.slice(0, 1).toUpperCase() }}</span>
        </div>
      </div>

      <section class="chapter-preview" aria-labelledby="chapter-preview-heading">
        <div class="section-heading">
          <div>
            <h2 id="chapter-preview-heading">Các chương đã xuất bản</h2>
            <p class="muted">Chọn một chương để bắt đầu đọc và nghe.</p>
          </div>
          <span v-if="chapters.length > 0" class="muted">{{ chapters.length }} chương</span>
        </div>

        <p v-if="chaptersLoading" class="status-state" role="status" aria-live="polite">
          Đang tải danh sách chương...
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
            <span class="chapter-number" aria-hidden="true">
              {{ String(chapter.ChapterNumber).padStart(2, '0') }}
            </span>
            <div class="chapter-copy">
              <h3>{{ chapter.Title }}</h3>
              <span class="muted">Đã xuất bản</span>
            </div>
            <RouterLink class="chapter-link" :to="`/stories/${story.id}/read`">
              Đọc & nghe <span aria-hidden="true">→</span>
            </RouterLink>
          </li>
        </ol>
      </section>
    </template>
  </section>
</template>
