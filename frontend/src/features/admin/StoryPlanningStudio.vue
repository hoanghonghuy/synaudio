<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { listAdminChapters, listAdminStories, listCreativeDecisions } from '../../api/client'
import type { Chapter, CreativeDecision, Story } from '../../api/types'

const route = useRoute()
const storyID = computed(() => String(route.params.storyID ?? ''))
const story = ref<Story | null>(null)
const chapters = ref<Chapter[]>([])
const decisions = ref<CreativeDecision[]>([])
const loading = ref(false)
const error = ref('')

const unresolvedDecisions = computed(() => decisions.value.filter((item) => item.Status !== 'SELECTED' && item.Status !== 'REJECTED'))

async function loadWorkspace() {
  loading.value = true
  error.value = ''
  try {
    const [storyList, chapterList, decisionList] = await Promise.all([
      listAdminStories(),
      listAdminChapters(storyID.value),
      listCreativeDecisions(storyID.value),
    ])
    story.value = storyList.stories.find((item) => item.id === storyID.value) ?? null
    chapters.value = chapterList.chapters
    decisions.value = decisionList.decisions
    if (!story.value) {
      error.value = 'Không tìm thấy story trong workspace Admin.'
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải Planning Studio.'
  } finally {
    loading.value = false
  }
}

onMounted(loadWorkspace)
</script>

<template>
  <section class="page admin planning-studio">
    <div class="section-heading">
      <div>
        <p class="eyebrow">Studio / Story Planning</p>
        <h1>{{ story?.title ?? 'Planning Studio' }}</h1>
        <p class="page-intro">Workspace quản trị planning và lifecycle của một Story.</p>
      </div>
      <div class="story-row-actions">
        <RouterLink class="control-link" to="/admin">Danh sách truyện</RouterLink>
        <RouterLink class="control-link" :to="`/admin/stories/${storyID}/control`">Trung tâm điều khiển</RouterLink>
        <RouterLink class="control-link" :to="`/admin/stories/${storyID}/review`">Duyệt nội dung</RouterLink>
      </div>
    </div>

    <p v-if="loading" class="status-state" role="status">Đang tải Planning Studio...</p>
    <div v-else-if="error" class="status-state error" role="alert">
      <strong>Không thể tải workspace.</strong>
      <p>{{ error }}</p>
      <button type="button" class="secondary-link" @click="loadWorkspace">Thử lại</button>
    </div>

    <template v-else-if="story">
      <div class="admin-workspace">
        <section class="admin-list-panel">
          <div class="section-heading">
            <div>
              <p class="eyebrow">Lifecycle</p>
              <h2>Trạng thái Story</h2>
            </div>
          </div>
          <dl class="planning-summary">
            <div><dt>Slug</dt><dd>{{ story.slug }}</dd></div>
            <div><dt>Trạng thái</dt><dd><span class="badge">{{ story.status }}</span></dd></div>
            <div><dt>Hiển thị</dt><dd><span class="badge">{{ story.visibility }}</span></dd></div>
            <div><dt>Mô tả</dt><dd>{{ story.description || 'Chưa có mô tả.' }}</dd></div>
          </dl>
        </section>

        <section class="admin-list-panel">
          <div class="section-heading">
            <div>
              <p class="eyebrow">Planning state</p>
              <h2>Chapter & quyết định sáng tạo</h2>
            </div>
          </div>
          <dl class="planning-summary">
            <div><dt>Chapters</dt><dd>{{ chapters.length }}</dd></div>
            <div><dt>Creative decisions</dt><dd>{{ decisions.length }}</dd></div>
            <div><dt>Đang chờ xử lý</dt><dd>{{ unresolvedDecisions.length }}</dd></div>
          </dl>
        </section>
      </div>

      <section class="admin-list-panel" aria-labelledby="planning-chapters-heading">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Chapter plan surface</p>
            <h2 id="planning-chapters-heading">Chapters</h2>
            <p class="muted">Danh sách server-backed hiện tại; versioned planning controls sẽ được đặt tại workspace này.</p>
          </div>
          <span class="count-label">{{ chapters.length }}</span>
        </div>
        <p v-if="chapters.length === 0" class="empty-state">Chưa có chapter.</p>
        <div v-else class="story-table-wrap">
          <table class="story-table">
            <thead><tr><th>Chapter</th><th>Trạng thái</th><th>Arc</th></tr></thead>
            <tbody>
              <tr v-for="chapter in chapters" :key="chapter.ID">
                <th scope="row">#{{ chapter.ChapterNumber }} — {{ chapter.Title }}</th>
                <td><span class="badge">{{ chapter.Status }}</span></td>
                <td>{{ chapter.ArcID || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>
  </section>
</template>

<style scoped>
.planning-studio {
  display: grid;
  gap: 1.5rem;
}

.planning-summary {
  display: grid;
  gap: 0.75rem;
  margin: 0;
}

.planning-summary > div {
  display: grid;
  grid-template-columns: minmax(9rem, 0.35fr) 1fr;
  gap: 1rem;
  padding-block: 0.75rem;
  border-bottom: 1px solid var(--border, #d9d9d9);
}

.planning-summary dt {
  font-weight: 700;
}

.planning-summary dd {
  margin: 0;
}
</style>
