<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { createStory, listAdminStories } from '../../api/client'
import { resolveAdminSecurityState, type AdminSecurityResolution } from '../../api/http-error'
import type { PrivilegedAssuranceReason } from '../../api/privileged-request'
import type { Story } from '../../api/types'
import { useAuthStore } from '../../stores/auth'
import ReAuthChallenge from './ReAuthChallenge.vue'
import AdminWorkspaceNav from './AdminWorkspaceNav.vue'

const auth = useAuthStore()
const stories = ref<Story[]>([])
const loading = ref(false)
const error = ref('')
const securityState = ref<AdminSecurityResolution | null>(null)

const reAuthOpen = ref(false)
const reAuthReason = ref<PrivilegedAssuranceReason | null>(null)

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

function openReAuth(reason: PrivilegedAssuranceReason | null = 'MFA_REQUIRED') {
  reAuthReason.value = reason
  reAuthOpen.value = true
}

async function handleReAuthCompleted(success: boolean) {
  reAuthOpen.value = false
  reAuthReason.value = null
  if (success) {
    await load()
  }
}

function handleReAuthDismissed() {
  reAuthOpen.value = false
  reAuthReason.value = null
}

async function load() {
  loading.value = true
  error.value = ''
  securityState.value = null
  try {
    const res = await listAdminStories()
    stories.value = res.stories
  } catch (e) {
    securityState.value = resolveAdminSecurityState(e, auth.user)
    error.value = securityState.value.message
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
    const res = resolveAdminSecurityState(e, auth.user)
    formError.value = res.message
    if (res.needsReAuth) {
      openReAuth(res.assuranceReason)
    }
  } finally {
    submitting.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="page admin">
    <AdminWorkspaceNav />
    <p class="eyebrow">Studio / Quản lý truyện</p>
    <h1>Quản lý truyện</h1>
    <p class="page-intro">Khởi tạo và theo dõi những câu chuyện đang được xây dựng trong Synaudio.</p>
    <div class="story-row-actions">
      <RouterLink class="control-link" to="/admin/audit">Audit & Provenance</RouterLink>
      <RouterLink class="control-link" to="/admin/security">Quản lý bảo mật</RouterLink>
      <RouterLink class="control-link" to="/account/security">Bảo mật tài khoản</RouterLink>
    </div>

    <!-- Cảnh báo bắt buộc kích hoạt MFA cho Admin -->
    <div v-if="auth.isAdmin && auth.user?.admin_mfa_required !== false && !auth.user?.mfa_enabled" class="admin-mfa-banner panel" role="alert">
      <div class="admin-mfa-body">
        <div class="admin-mfa-badge">Bảo Mật Bắt Buộc</div>
        <h2>Kích hoạt xác thực hai yếu tố (MFA)</h2>
        <p class="muted">
          Tài khoản quản trị cần bật MFA để bảo vệ dữ liệu truyện và phân quyền hệ thống. Toàn bộ thao tác trong Studio sẽ được mở khóa ngay sau khi bạn thiết lập ứng dụng xác thực (TOTP).
        </p>
      </div>
      <RouterLink class="primary-link" to="/account/security">Thiết lập MFA ngay</RouterLink>
    </div>

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
          <div class="status-actions">
            <RouterLink
              v-if="securityState?.needsMfaSetup || (auth.isAdmin && auth.user?.admin_mfa_required !== false && !auth.user?.mfa_enabled)"
              class="primary-link"
              to="/account/security"
            >
              Thiết lập MFA ngay
            </RouterLink>
            <button
              v-else-if="securityState?.needsReAuth"
              class="primary-link"
              type="button"
              @click="openReAuth(securityState?.assuranceReason)"
            >
              Xác minh MFA
            </button>
            <button class="secondary-link" type="button" @click="load">Thử lại</button>
          </div>
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
          <span>Tiêu đề <span class="required" aria-hidden="true">*</span></span>
          <input id="story-title" v-model="form.title" type="text" required placeholder="Nhập tên truyện..." />
        </label>
        <label for="story-description">
          <span>Mô tả</span>
          <textarea id="story-description" v-model="form.description" rows="3" placeholder="Tóm tắt ngắn gọn nội dung truyện..."></textarea>
        </label>
        <p class="field-help">
          Người tạo: {{ auth.user?.email ?? 'tài khoản admin hiện tại' }}. Hệ thống tự ghi nhận từ phiên đăng nhập.
        </p>

        <fieldset>
          <legend>Chính sách / âm thanh</legend>
          <div class="row">
            <label for="minimum-audio-duration">
              <span>Thời lượng tối thiểu (giây)</span>
              <input id="minimum-audio-duration" v-model.number="form.minimum_audio_duration_sec" type="number" min="0" />
            </label>
            <label for="target-audio-duration">
              <span>Thời lượng mục tiêu (giây)</span>
              <input id="target-audio-duration" v-model.number="form.target_audio_duration_sec" type="number" min="0" />
            </label>
          </div>
        </fieldset>

        <fieldset>
          <legend>Nội dung</legend>
          <div class="row">
            <label for="content-origin">
              <span>Nguồn nội dung</span>
              <input id="content-origin" v-model="form.content_origin" type="text" placeholder="Sáng tác / AI..." />
            </label>
            <label for="story-language">
              <span>Ngôn ngữ</span>
              <input id="story-language" v-model="form.language" type="text" placeholder="vi" />
            </label>
            <label for="narration-language">
              <span>Ngôn ngữ kể chuyện</span>
              <input id="narration-language" v-model="form.narration_language" type="text" placeholder="vi-VN" />
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

    <ReAuthChallenge
      :open="reAuthOpen"
      :reason="reAuthReason"
      @completed="handleReAuthCompleted"
      @dismissed="handleReAuthDismissed"
    />
  </section>
</template>
