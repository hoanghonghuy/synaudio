<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createStory, listAdminStories } from '../../api/client'
import type { Story } from '../../api/types'

const stories = ref<Story[]>([])
const loading = ref(false)
const error = ref('')

const form = ref({
  title: '',
  description: '',
  created_by: '',
  minimum_audio_duration_sec: 1200,
  target_audio_duration_sec: 1800,
  content_origin: 'ORIGINAL',
  language: 'vi',
  narration_language: 'vi',
})

const submitting = ref(false)
const formError = ref('')
const formSuccess = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await listAdminStories()
    stories.value = res.stories
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải danh sách truyện.'
  } finally {
    loading.value = false
  }
}

async function submit() {
  submitting.value = true
  formError.value = ''
  formSuccess.value = ''
  try {
    await createStory({
      title: form.value.title,
      description: form.value.description,
      created_by: form.value.created_by,
      policy: {
        minimum_audio_duration_sec: form.value.minimum_audio_duration_sec,
        target_audio_duration_sec: form.value.target_audio_duration_sec,
        content_origin: form.value.content_origin,
        language: form.value.language,
        narration_language: form.value.narration_language,
      },
    })
    formSuccess.value = 'Đã tạo truyện thành công.'
    form.value.title = ''
    form.value.description = ''
    await load()
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Không thể tạo truyện.'
  } finally {
    submitting.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="page admin">
    <p class="eyebrow">Admin</p>
    <h1>Quản lý truyện</h1>

    <form class="create-form" @submit.prevent="submit">
      <h2>Tạo truyện mới</h2>
      <label>
        Tiêu đề
        <input v-model="form.title" required />
      </label>
      <label>
        Mô tả
        <textarea v-model="form.description" rows="3"></textarea>
      </label>
      <label>
        Created by (user ID)
        <input v-model="form.created_by" placeholder="UUID của admin" />
      </label>
      <div class="row">
        <label>
          Thời lượng tối thiểu (giây)
          <input v-model.number="form.minimum_audio_duration_sec" type="number" min="0" />
        </label>
        <label>
          Thời lượng mục tiêu (giây)
          <input v-model.number="form.target_audio_duration_sec" type="number" min="0" />
        </label>
      </div>
      <div class="row">
        <label>
          Nguồn nội dung
          <input v-model="form.content_origin" />
        </label>
        <label>
          Ngôn ngữ
          <input v-model="form.language" />
        </label>
        <label>
          Ngôn ngữ kể chuyện
          <input v-model="form.narration_language" />
        </label>
      </div>
      <p v-if="formError" class="error">{{ formError }}</p>
      <p v-if="formSuccess" class="success">{{ formSuccess }}</p>
      <button type="submit" :disabled="submitting">
        {{ submitting ? 'Đang tạo...' : 'Tạo truyện' }}
      </button>
    </form>

    <h2>Danh sách truyện</h2>
    <p v-if="loading" class="note">Đang tải...</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <p v-else-if="stories.length === 0" class="note">Chưa có truyện nào.</p>

    <ul v-else class="story-list">
      <li v-for="s in stories" :key="s.id" class="story-row">
        <div>
          <strong>{{ s.title }}</strong>
          <span class="slug">{{ s.slug }}</span>
        </div>
        <div class="meta">
          <span class="badge">{{ s.status }}</span>
          <span class="badge">{{ s.visibility }}</span>
          <RouterLink class="control-link" :to="`/admin/stories/${s.id}/control`">Điều khiển</RouterLink>
        </div>
      </li>
    </ul>
  </section>
</template>
