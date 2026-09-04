<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { getListenerLibrary, removeFavorite } from '../../api/client'
import type { ListenerLibrary, LibraryItem } from '../../api/types'

const library = ref<ListenerLibrary | null>(null)
const loading = ref(true)
const error = ref('')
const removingStoryID = ref('')

function resumeTo(item: LibraryItem) {
  return {
    name: 'reader',
    params: { storyID: item.story_id },
    query: { chapter: item.chapter_id },
  }
}

function progressLabel(item: LibraryItem) {
  if (item.completed_at && item.relisten_status === 'NO_RELISTEN_NEEDED') return 'Đã nghe xong'
  if (item.relisten_status === 'RELISTEN_REQUIRED') return 'Cần nghe lại'
  if (item.relisten_status === 'RELISTEN_RECOMMENDED') return 'Có bản cập nhật'
  const seconds = Math.max(0, Math.floor(item.position_ms / 1000))
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  return `Tiếp tục từ ${minutes}:${String(rest).padStart(2, '0')}`
}

async function loadLibrary() {
  loading.value = true
  error.value = ''
  try {
    library.value = await getListenerLibrary()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải thư viện.'
  } finally {
    loading.value = false
  }
}

async function unfavorite(storyID: string) {
  removingStoryID.value = storyID
  try {
    await removeFavorite(storyID)
    await loadLibrary()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể bỏ yêu thích.'
  } finally {
    removingStoryID.value = ''
  }
}

onMounted(loadLibrary)
</script>

<template>
  <section class="page listener-library">
    <div class="page-heading">
      <div>
        <p class="eyebrow">Thư viện của bạn</p>
        <h1>Tiếp tục nghe</h1>
        <p class="muted">Tiến độ được đồng bộ theo tài khoản trên các thiết bị.</p>
      </div>
      <RouterLink class="secondary-link" to="/">Khám phá thêm truyện</RouterLink>
    </div>

    <p v-if="loading" class="status-state" role="status">Đang tải thư viện...</p>
    <div v-else-if="error" class="status-state error" role="alert">
      <strong>Không thể tải thư viện.</strong>
      <p>{{ error }}</p>
      <button class="secondary-link" type="button" @click="loadLibrary">Thử lại</button>
    </div>

    <template v-else-if="library">
      <article v-if="library.continue_listening" class="library-hero panel">
        <p class="eyebrow">Nghe tiếp</p>
        <h2>{{ library.continue_listening.story_title }}</h2>
        <p>
          Chương {{ library.continue_listening.chapter_number }} ·
          {{ library.continue_listening.chapter_title || 'Chưa đặt tên' }}
        </p>
        <p v-if="library.continue_listening.relisten_status !== 'NO_RELISTEN_NEEDED'" class="relisten-notice">
          {{ progressLabel(library.continue_listening) }} — tiến độ cũ vẫn được giữ.
        </p>
        <RouterLink class="primary-link" :to="resumeTo(library.continue_listening)">
          {{ progressLabel(library.continue_listening) }}
        </RouterLink>
      </article>
      <div v-else class="status-state">
        <strong>Chưa có nội dung đang nghe dở.</strong>
        <p>Mở một truyện và bắt đầu nghe để tiến độ xuất hiện ở đây.</p>
      </div>

      <section class="library-section" aria-labelledby="favorites-heading">
        <div class="section-heading">
          <h2 id="favorites-heading">Yêu thích</h2>
          <span class="muted">{{ library.favorites.length }} truyện</span>
        </div>
        <div v-if="library.favorites.length" class="card-grid">
          <article v-for="story in library.favorites" :key="story.story_id" class="story-card">
            <div>
              <h3>{{ story.title }}</h3>
              <p>{{ story.description || 'Chưa có mô tả.' }}</p>
            </div>
            <div class="card-actions">
              <RouterLink class="secondary-link" :to="`/stories/${story.story_id}`">Chi tiết</RouterLink>
              <button
                class="secondary-link"
                type="button"
                :disabled="removingStoryID === story.story_id"
                @click="unfavorite(story.story_id)"
              >
                Bỏ yêu thích
              </button>
            </div>
          </article>
        </div>
        <p v-else class="muted">Chưa có truyện yêu thích.</p>
      </section>

      <section class="library-section" aria-labelledby="recent-heading">
        <div class="section-heading">
          <h2 id="recent-heading">Nghe gần đây</h2>
          <span class="muted">{{ library.recent.length }} chương</span>
        </div>
        <div class="library-list">
          <article v-for="item in library.recent" :key="item.chapter_id" class="library-row">
            <div>
              <strong>{{ item.story_title }}</strong>
              <p>Chương {{ item.chapter_number }} · {{ item.chapter_title || 'Chưa đặt tên' }}</p>
              <small v-if="item.relisten_status !== 'NO_RELISTEN_NEEDED'" class="relisten-label">
                {{ progressLabel(item) }}
              </small>
            </div>
            <RouterLink class="secondary-link" :to="resumeTo(item)">Mở</RouterLink>
          </article>
        </div>
      </section>

      <section class="library-section" aria-labelledby="completed-heading">
        <div class="section-heading">
          <h2 id="completed-heading">Đã hoàn thành</h2>
          <span class="muted">{{ library.completed.length }} chương</span>
        </div>
        <div v-if="library.completed.length" class="library-list">
          <article v-for="item in library.completed" :key="item.chapter_id" class="library-row">
            <div>
              <strong>{{ item.story_title }}</strong>
              <p>Chương {{ item.chapter_number }} · {{ item.chapter_title || 'Chưa đặt tên' }}</p>
              <small v-if="item.relisten_status !== 'NO_RELISTEN_NEEDED'" class="relisten-label">
                {{ progressLabel(item) }}
              </small>
            </div>
            <RouterLink class="secondary-link" :to="resumeTo(item)">Nghe lại</RouterLink>
          </article>
        </div>
        <p v-else class="muted">Chưa có chương đã hoàn thành.</p>
      </section>
    </template>
  </section>
</template>
