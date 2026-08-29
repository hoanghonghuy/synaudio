<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createStory, listAdminStories } from '../../api/client'
import type { Story } from '../../api/types'
import { useAuthStore } from '../../stores/auth'

const auth = useAuthStore()
const stories = ref<Story[]>([])
const loading = ref(false)
const error = ref('')

const form = ref({
  title: '',
  description: '',
  minimum_audio_duration_sec: 1200,
  target_audio_duration_sec: 1800,
  content_origin: 'ORIGINAL',
  language: 'vi',
  narration_language: 'vi',
})

const submitting = ref(false)
const formError = ref('')
const formSuccess = ref('')

function statusLabel(status: Story['status']) {
  const labels: Record<Story['status'], string> = {
    DRAFT: 'Bản nháp',
    ACTIVE: 'Đang phát hành',
    COMPLETED: 'Đã hoàn thành',
    ARCHIVED: 'Đã lưu trữ',
  }
  return labels[status]
}

function visibilityLabel(visibility: Story['visibility']) {
  return visibility === 'PUBLIC' ? 'Công khai' : 'Riêng tư'
}

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
    <p class="eyebrow">Studio / Quản lý truyện</p>
    <h1>Quản lý truyện</h1>
    <p class="page-intro">Khởi tạo và theo dõi những câu chuyện đang được xây dựng trong Synaudio.</p>

    <div class="admin-workspace">
      <section class="admin-list-panel" aria-labelledby="story-list-heading">
        <div class="section-heading">
          <div>
            <h2 id="story-list-heading">Danh sách truyện</h2>
            <p class="muted">Các story hiện có trong workspace.</p>
          </div>
          <span class="count-label">{{ stories.length }} truyện</span>
        </div>

        <p v-if="loading" class="status-state" role="status" aria-live="polite">Đang tải danh sách truyện...</p>
        <div v-else-if="error" class="status-state error" role="alert">
          <strong>Không thể tải danh sách truyện.</strong>
          <p>{{ error }}</p>
          <button class="secondary-link" type="button" @click="load">Thử lại</button>
        </div>
        <p v-else-if="stories.length === 0" class="empty-state">
          <strong>Chưa có truyện nào.</strong>
          <span>Bắt đầu bằng cách tạo story đầu tiên ở bên cạnh.</span>
        </p>

        <div v-else class="story-table-wrap">
          <table class="story-table">
            <caption class="sr-only">Danh sách truyện trong workspace</caption>
            <thead>
              <tr>
                <th scope="col">Truyện</th>
                <th scope="col">Trạng thái</th>
                <th scope="col">Hiển thị</th>
                <th scope="col"><span class="sr-only">Thao tác</span></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="s in stories" :key="s.id">
                <th scope="row" data-label="Truyện">
                  <strong>{{ s.title }}</strong>
                  <span class="slug">{{ s.slug }}</span>
                </th>
                <td data-label="Trạng thái">
                  <span class="badge">{{ statusLabel(s.status) }}</span>
                </td>
                <td data-label="Hiển thị">
                  <span class="badge">{{ visibilityLabel(s.visibility) }}</span>
                </td>
                <td data-label="Thao tác">
                  <div class="story-row-actions">
                    <RouterLink class="control-link" :to="`/admin/stories/${s.id}/control`">
                      Trung tâm điều khiển
                    </RouterLink>
                    <RouterLink class="control-link" :to="`/admin/stories/${s.id}/review`">
                      Duyệt nội dung
                    </RouterLink>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <form class="create-form" aria-labelledby="create-story-heading" @submit.prevent="submit">
        <div>
          <p class="eyebrow">Khởi tạo</p>
          <h2 id="create-story-heading">Tạo truyện mới</h2>
          <p class="muted">Thiết lập nền tảng để bắt đầu phát triển một story.</p>
        </div>

        <label for="story-title">
          Tiêu đề <span aria-hidden="true">*</span>
          <input id="story-title" v-model="form.title" required />
        </label>
        <label for="story-description">
          Mô tả
          <textarea id="story-description" v-model="form.description" rows="3"></textarea>
        </label>
        <p class="field-help">
          Người tạo: {{ auth.user?.email ?? 'tài khoản admin hiện tại' }}. Hệ thống tự ghi nhận từ phiên đăng nhập.
        </p>

        <fieldset>
          <legend>Chính sách / âm thanh</legend>
          <div class="row">
            <label for="minimum-audio-duration">
              Thời lượng tối thiểu (giây)
              <input id="minimum-audio-duration" v-model.number="form.minimum_audio_duration_sec" type="number" min="0" />
            </label>
            <label for="target-audio-duration">
              Thời lượng mục tiêu (giây)
              <input id="target-audio-duration" v-model.number="form.target_audio_duration_sec" type="number" min="0" />
            </label>
          </div>
        </fieldset>

        <fieldset>
          <legend>Nội dung</legend>
          <div class="row">
            <label for="content-origin">
              Nguồn nội dung
              <input id="content-origin" v-model="form.content_origin" />
            </label>
            <label for="story-language">
              Ngôn ngữ
              <input id="story-language" v-model="form.language" />
            </label>
            <label for="narration-language">
              Ngôn ngữ kể chuyện
              <input id="narration-language" v-model="form.narration_language" />
            </label>
          </div>
        </fieldset>

        <p v-if="formError" class="status-state error" role="alert">{{ formError }}</p>
        <p v-if="formSuccess" class="status-state success" role="status" aria-live="polite">{{ formSuccess }}</p>
        <button type="submit" :disabled="submitting">
          {{ submitting ? 'Đang tạo...' : 'Tạo truyện' }}
        </button>
      </form>
    </div>
  </section>
</template>
