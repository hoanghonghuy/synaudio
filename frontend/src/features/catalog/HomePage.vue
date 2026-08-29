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

onMounted(load)
</script>

<template>
  <section class="page catalog">
    <div class="catalog-intro">
      <div class="hero-copy">
        <p class="eyebrow">Thư viện sách nói tiếng Việt</p>
        <h1>Một câu chuyện hay có thể thay đổi nhịp của một ngày.</h1>
        <p class="lede">
          Khám phá những truyện nói được tạo ra để đọc chậm, nghe sâu và quay lại bất cứ
          lúc nào.
        </p>
      </div>
      <aside class="hero-note" aria-label="Giới thiệu Synaudio">
        <strong>Đọc bằng mắt. Nghe bằng tâm trí.</strong>
        <span>Một không gian yên tĩnh cho những câu chuyện mới.</span>
      </aside>
    </div>

    <form class="searchbar" role="search" @submit.prevent="load">
      <label class="search-field search-field-wide" for="story-search">
        <span>Tìm kiếm trong thư viện</span>
        <input id="story-search" v-model="q" type="search" placeholder="Tìm theo tiêu đề hoặc mô tả…" />
      </label>
      <label class="search-field" for="story-genre">
        <span>Thể loại</span>
        <select id="story-genre" v-model="genre">
          <option value="">Tất cả thể loại</option>
          <option v-for="g in genres" :key="g.id" :value="g.slug">{{ g.name }}</option>
        </select>
      </label>
      <label class="search-field" for="story-sort">
        <span>Sắp xếp</span>
        <select id="story-sort" v-model="sort">
          <option value="">Mới cập nhật</option>
          <option value="NEW">Mới nhất</option>
          <option value="RECENTLY_UPDATED">Cập nhật gần đây</option>
          <option value="TITLE">Theo tiêu đề</option>
        </select>
      </label>
      <button type="submit" :disabled="loading">Tìm truyện</button>
    </form>

    <div class="section-heading">
      <h2>Đang được khám phá</h2>
      <span class="muted">{{ stories.length }} truyện{{ isSearching ? ' phù hợp' : '' }}</span>
    </div>

    <p v-if="loading" class="status-state" role="status" aria-live="polite">Đang tìm những câu chuyện phù hợp...</p>
    <div v-else-if="error" class="status-state error" role="alert">
      <strong>Không thể tải thư viện.</strong>
      <p>{{ error }}</p>
      <button class="secondary-link" type="button" @click="load">Thử lại</button>
    </div>
    <div v-else-if="stories.length === 0" class="empty-state">
      <strong>Chưa có truyện phù hợp.</strong>
      <p>Hãy thử một từ khóa hoặc bộ lọc khác.</p>
    </div>

    <ul v-else class="story-grid catalog-results" :class="{ 'search-mode': isSearching }">
      <li v-for="s in stories" :key="s.id" class="story-card">
        <div class="story-card-cover" aria-hidden="true">
          <span>{{ s.title.slice(0, 1).toUpperCase() }}</span>
        </div>
        <div class="story-card-content">
          <div class="story-card-kicker">{{ s.status }}</div>
          <h2>
            <RouterLink :to="`/stories/${s.id}`">{{ s.title }}</RouterLink>
          </h2>
          <p class="desc">{{ s.description || 'Một câu chuyện đang chờ bạn khám phá.' }}</p>
          <div class="meta">
            <span class="badge">{{ s.status }}</span>
          </div>
          <RouterLink class="read-link" :to="`/stories/${s.id}`">
            Xem truyện <span aria-hidden="true">→</span>
          </RouterLink>
        </div>
      </li>
    </ul>
  </section>
</template>
