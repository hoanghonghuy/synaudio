<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listGenres, listPublicStories } from '../../api/client'
import type { Genre, Story } from '../../api/types'

const stories = ref<Story[]>([])
const genres = ref<Genre[]>([])
const loading = ref(false)
const error = ref('')

const q = ref('')
const genre = ref('')
const sort = ref('')

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
    <p class="eyebrow">Thư viện công khai</p>
    <h1>Khám phá truyện</h1>

    <form class="searchbar" @submit.prevent="load">
      <input v-model="q" type="search" placeholder="Tìm theo tiêu đề, mô tả..." />
      <select v-model="genre">
        <option value="">Tất cả thể loại</option>
        <option v-for="g in genres" :key="g.id" :value="g.slug">{{ g.name }}</option>
      </select>
      <select v-model="sort">
        <option value="">Mặc định</option>
        <option value="NEW">Mới nhất</option>
        <option value="UPDATED">Cập nhật gần đây</option>
        <option value="TITLE">Theo tiêu đề</option>
      </select>
      <button type="submit">Tìm</button>
    </form>

    <p v-if="loading" class="note">Đang tải...</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <p v-else-if="stories.length === 0" class="note">Chưa có truyện công khai nào.</p>

    <ul v-else class="story-grid">
      <li v-for="s in stories" :key="s.id" class="story-card">
        <h2>{{ s.title }}</h2>
        <p class="desc">{{ s.description }}</p>
        <div class="meta">
          <span class="badge">{{ s.status }}</span>
          <span class="badge">{{ s.visibility }}</span>
        </div>
      </li>
    </ul>
  </section>
</template>
