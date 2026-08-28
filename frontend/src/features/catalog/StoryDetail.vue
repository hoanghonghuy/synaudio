<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { getPublicStory } from '../../api/client'
import type { Story } from '../../api/types'

const route = useRoute()
const storyID = computed(() => route.params.storyID as string)
const story = ref<Story | null>(null)
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    story.value = await getPublicStory(storyID.value)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải thông tin truyện.'
  } finally {
    loading.value = false
  }
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
          <p class="eyebrow">Chi tiết truyện · Synaudio</p>
          <h1>{{ story.title }}</h1>
          <div class="meta" aria-label="Trạng thái truyện">
            <span class="badge">{{ story.status }}</span>
            <span class="badge">{{ story.visibility }}</span>
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
    </template>
  </section>
</template>
