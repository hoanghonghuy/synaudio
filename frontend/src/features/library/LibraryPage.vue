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
        <div class="library-hero-art" aria-hidden="true">
          <span class="hero-art-letter">{{ library.continue_listening.story_title.slice(0, 1).toUpperCase() }}</span>
          <span class="hero-art-pulse">▶</span>
        </div>
        <div class="library-hero-info">
          <p class="eyebrow">Đang nghe dở</p>
          <h2 class="library-hero-title">{{ library.continue_listening.story_title }}</h2>
          <p class="library-hero-chapter">
            Chương {{ library.continue_listening.chapter_number }} ·
            {{ library.continue_listening.chapter_title || 'Chưa đặt tên' }}
          </p>
          <p v-if="library.continue_listening.relisten_status !== 'NO_RELISTEN_NEEDED'" class="relisten-notice">
            {{ progressLabel(library.continue_listening) }} — tiến độ cũ vẫn được giữ.
          </p>
          <div class="library-hero-actions">
            <RouterLink class="primary-link hero-play-btn" :to="resumeTo(library.continue_listening)">
              ▶ {{ progressLabel(library.continue_listening) }}
            </RouterLink>
            <RouterLink class="secondary-link" :to="`/stories/${library.continue_listening.story_id}`">
              Chi tiết truyện
            </RouterLink>
          </div>
        </div>
      </article>
      <div v-else class="status-state continue-empty">
        <div class="empty-icon" aria-hidden="true">🎧</div>
        <strong>Chưa có nội dung đang nghe dở.</strong>
        <p>Mở một tác phẩm trong kho truyện và bắt đầu nghe để tiến độ tự động xuất hiện ở đây.</p>
        <RouterLink class="primary-link" to="/">Khám phá kho truyện</RouterLink>
      </div>

      <section class="library-section" aria-labelledby="favorites-heading">
        <div class="section-heading">
          <div class="section-heading-text">
            <h2 id="favorites-heading">Yêu thích</h2>
            <span class="count-badge">{{ library.favorites.length }} truyện</span>
          </div>
        </div>
        <ul v-if="library.favorites.length" class="story-grid favorites-grid">
          <li v-for="story in library.favorites" :key="story.story_id" class="story-card">
            <RouterLink :to="`/stories/${story.story_id}`" class="story-card-link" :aria-label="story.title">
              <div class="story-card-cover" aria-hidden="true">
                <span class="cover-letter">{{ story.title.slice(0, 1).toUpperCase() }}</span>
                <div class="cover-play-badge">▶</div>
              </div>
              <div class="story-card-content">
                <strong class="title">{{ story.title }}</strong>
                <p v-if="story.description" class="desc">{{ story.description }}</p>
                <div class="story-card-meta">
                  <span class="story-card-action">Nghe ngay →</span>
                </div>
              </div>
            </RouterLink>
            <div class="card-footer-action">
              <button
                class="unfavorite-btn"
                type="button"
                :disabled="removingStoryID === story.story_id"
                aria-label="Bỏ yêu thích"
                @click="unfavorite(story.story_id)"
              >
                {{ removingStoryID === story.story_id ? 'Đang bỏ…' : '♥ Bỏ lưu' }}
              </button>
            </div>
          </li>
        </ul>
        <p v-else class="empty-inline-note">Chưa có truyện nào trong danh sách yêu thích.</p>
      </section>

      <section class="library-section" aria-labelledby="recent-heading">
        <div class="section-heading">
          <div class="section-heading-text">
            <h2 id="recent-heading">Nghe gần đây</h2>
            <span class="count-badge">{{ library.recent.length }} chương</span>
          </div>
        </div>
        <div v-if="library.recent.length" class="library-track-list">
          <article v-for="item in library.recent" :key="item.chapter_id" class="library-track-row">
            <div class="track-row-art" aria-hidden="true">
              {{ item.story_title.slice(0, 1).toUpperCase() }}
            </div>
            <div class="track-row-info">
              <strong class="track-row-story">{{ item.story_title }}</strong>
              <span class="track-row-chapter">
                Chương {{ item.chapter_number }} · {{ item.chapter_title || 'Chưa đặt tên' }}
              </span>
              <span v-if="item.relisten_status !== 'NO_RELISTEN_NEEDED'" class="relisten-badge">
                {{ progressLabel(item) }}
              </span>
            </div>
            <div class="track-row-actions">
              <RouterLink class="track-play-btn" :to="resumeTo(item)">
                <span aria-hidden="true">▶</span> Mở
              </RouterLink>
            </div>
          </article>
        </div>
        <p v-else class="empty-inline-note">Chưa có lịch sử nghe gần đây.</p>
      </section>

      <section class="library-section" aria-labelledby="completed-heading">
        <div class="section-heading">
          <div class="section-heading-text">
            <h2 id="completed-heading">Đã hoàn thành</h2>
            <span class="count-badge">{{ library.completed.length }} chương</span>
          </div>
        </div>
        <div v-if="library.completed.length" class="library-track-list">
          <article v-for="item in library.completed" :key="item.chapter_id" class="library-track-row">
            <div class="track-row-art completed-art" aria-hidden="true">
              ✓
            </div>
            <div class="track-row-info">
              <strong class="track-row-story">{{ item.story_title }}</strong>
              <span class="track-row-chapter">
                Chương {{ item.chapter_number }} · {{ item.chapter_title || 'Chưa đặt tên' }}
              </span>
              <span v-if="item.relisten_status !== 'NO_RELISTEN_NEEDED'" class="relisten-badge">
                {{ progressLabel(item) }}
              </span>
            </div>
            <div class="track-row-actions">
              <RouterLink class="track-play-btn secondary" :to="resumeTo(item)">
                <span aria-hidden="true">↺</span> Nghe lại
              </RouterLink>
            </div>
          </article>
        </div>
        <p v-else class="empty-inline-note">Chưa có chương nào đã hoàn thành.</p>
      </section>
    </template>
  </section>
</template>
