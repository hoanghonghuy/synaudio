<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { listGenres, listPublicStories } from '../../api/client'
import type { Genre, Story } from '../../api/types'

const stories = ref<Story[]>([])
const genres = ref<Genre[]>([])
const loading = ref(false)
const error = ref('')

const q = ref('')
const genre = ref('')
const sort = ref('')
const isSearching = computed(() => Boolean(q.value || genre.value || sort.value))
const heroListenPath = computed(() => stories.value[0] ? `/stories/${stories.value[0].id}/read` : '#catalog-results')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [storyRes, genreRes] = await Promise.all([
      listPublicStories({ q: q.value, genre: genre.value, sort: sort.value }),
      listGenres(),
    ])
    stories.value = storyRes.stories
    genres.value = genreRes.genres
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải danh sách truyện.'
  } finally {
    loading.value = false
  }
}

function selectGenre(slug: string) {
  genre.value = genre.value === slug ? '' : slug
  load()
}

function selectSort(sortVal: string) {
  sort.value = sortVal
  load()
}

function statusLabel(status: Story['status']) {
  const map: Record<string, string> = {
    ACTIVE: 'Đang phát hành',
    COMPLETED: 'Hoàn thành',
    DRAFT: 'Bản thảo',
    ARCHIVED: 'Lưu trữ',
  }
  return map[status] || status
}

onMounted(load)
</script>

<template>
  <section class="page catalog">
    <div class="catalog-intro">
      <div class="hero-copy">
        <p class="eyebrow">
          <svg class="eyebrow-icon" width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
            <circle cx="12" cy="12" r="10"></circle>
          </svg>
          <span>Âm thanh kể chuyện thế hệ mới</span>
        </p>
        <h1>Chạm play. Bước vào một thế giới khác.</h1>
        <p class="lede">
          Khám phá những câu chuyện được tạo nên để nghe sâu, đọc chậm và nhớ lâu — từ những phút rảnh đến những đêm không ngủ.
        </p>
        <div class="hero-cta-row">
          <RouterLink class="primary-btn-cta" :to="heroListenPath">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <polygon points="5 3 19 12 5 21 5 3"></polygon>
            </svg>
            {{ stories.length ? 'Nghe ngay' : 'Khám phá bộ sưu tập' }}
          </RouterLink>
          <a class="hero-secondary-link" href="#catalog-results">Xem các tuyển tập <span aria-hidden="true">↓</span></a>
        </div>
        <div class="hero-stat-row" aria-label="Điểm nổi bật của Synaudio">
          <div><strong>24/7</strong><span>chuyện để nghe</span></div>
          <div><strong>AI + người</strong><span>kể chuyện giàu cảm xúc</span></div>
          <div><strong>1 nơi</strong><span>nghe, đọc, lưu lại</span></div>
        </div>
      </div>
    </div>

    <div class="search-section">
      <form class="searchbar" role="search" @submit.prevent="load">
        <div class="search-input-wrap">
          <svg class="search-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input id="story-search" v-model="q" type="search" placeholder="Tìm theo tên truyện, chủ đề…" autocomplete="off" />
          <button v-if="q" type="button" class="search-clear-btn" aria-label="Xóa từ khóa" @click="q = ''; load()">×</button>
        </div>
        <button type="submit" class="search-submit-btn" :disabled="loading">{{ loading ? 'Đang tìm…' : 'Tìm truyện' }}</button>
      </form>

      <div class="filter-chips-row" role="region" aria-label="Lọc theo thể loại">
        <button
          type="button"
          class="chip-pill"
          :class="{ active: genre === '' }"
          @click="selectGenre('')"
        >
          Tất cả
        </button>
        <button
          v-for="g in genres"
          :key="g.id"
          type="button"
          class="chip-pill"
          :class="{ active: genre === g.slug }"
          @click="selectGenre(g.slug)"
        >
          {{ g.name }}
        </button>
      </div>

      <div class="sort-chips-row" role="region" aria-label="Sắp xếp">
        <span class="sort-label">Sắp xếp:</span>
        <button
          type="button"
          class="sort-chip"
          :class="{ active: sort === '' }"
          @click="selectSort('')"
        >
          Mặc định
        </button>
        <button
          type="button"
          class="sort-chip"
          :class="{ active: sort === 'NEW' }"
          @click="selectSort('NEW')"
        >
          Mới nhất
        </button>
        <button
          type="button"
          class="sort-chip"
          :class="{ active: sort === 'TITLE' }"
          @click="selectSort('TITLE')"
        >
          Tên A-Z
        </button>
      </div>
    </div>

    <div id="catalog-results" class="section-heading">
      <div>
        <p class="eyebrow">Tuyển tập hôm nay</p>
        <h2>Chuyện đang chờ bạn</h2>
      </div>
      <span class="count-label">{{ stories.length }} tác phẩm{{ isSearching ? ' phù hợp' : '' }}</span>
    </div>

    <p v-if="loading" class="status-state" role="status" aria-live="polite">
      <span class="spinner" aria-hidden="true"></span>
      Đang tìm những câu chuyện phù hợp...
    </p>
    <div v-else-if="error" class="status-state error" role="alert">
      <strong>Không thể tải thư viện.</strong>
      <p>{{ error }}</p>
      <button class="secondary-link" type="button" @click="load">Thử lại</button>
    </div>
    <div v-else-if="stories.length === 0" class="empty-state">
      <div class="empty-icon" aria-hidden="true">📖</div>
      <strong>Chưa có truyện phù hợp.</strong>
      <p>Hãy thử một từ khóa hoặc thể loại khác.</p>
    </div>

    <ul v-else class="story-grid catalog-results" :class="{ 'search-mode': isSearching }">
      <li v-for="(s, index) in stories" :key="s.id" class="story-card">
        <RouterLink :to="`/stories/${s.id}`" class="story-card-link" :aria-label="s.title">
          <div class="story-card-cover" aria-hidden="true">
            <span class="cover-letter">{{ s.title.slice(0, 1).toUpperCase() }}</span>
            <span class="cover-seal-mark" aria-hidden="true">音</span>
            <div class="cover-play-badge">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <polygon points="5 3 19 12 5 21 5 3"></polygon>
              </svg>
            </div>
          </div>
          <div class="story-card-content">
            <div class="story-card-meta">
              <span class="badge badge-accent">{{ statusLabel(s.status) }}</span>
              <span v-if="index < 3" class="badge">Đề xuất</span>
            </div>
            <h3 class="story-title">{{ s.title }}</h3>
            <p class="desc">{{ s.description || 'Một câu chuyện đang chờ bạn khám phá.' }}</p>
            <div class="story-card-action">
              <span class="read-link">Khám phá ngay <span aria-hidden="true">→</span></span>
            </div>
          </div>
        </RouterLink>
      </li>
    </ul>
  </section>
</template>
