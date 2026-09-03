<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  activateStory,
  archiveStory,
  createStoryContentProfile,
  generateStoryFoundation,
  getActivationReadiness,
  getStoryBible,
  getStoryContentProfile,
  getStoryEnding,
  getStoryWorkflowSettings,
  listAdminChapters,
  listAdminStories,
  listCreativeDecisions,
  listStoryArcs,
  listStoryCharacters,
  makeStoryPrivate,
  makeStoryPublic,
  restoreStory,
  updateStoryWorkflowSettings,
} from '../../api/client'
import type {
  ActivationReadiness,
  EndingPlanVersion,
  PlanningCharacter,
  StoryArc,
  StoryBibleVersion,
  StoryContentProfile,
  StoryWorkflowSettings,
} from '../../api/client'
import type { Chapter, CreativeDecision, Story } from '../../api/types'

type PolicySnapshot = {
  story_id: string
  minimum_audio_duration_sec: number
  target_audio_duration_sec: number
  content_origin: string
  language: string
  narration_language: string
  policy_version: number
}

type ReadinessWithPolicy = ActivationReadiness & {
  generation_policy?: PolicySnapshot
}

const route = useRoute()
const storyID = computed(() => String(route.params.storyID ?? ''))

const story = ref<Story | null>(null)
const chapters = ref<Chapter[]>([])
const decisions = ref<CreativeDecision[]>([])
const bible = ref<StoryBibleVersion | null>(null)
const ending = ref<EndingPlanVersion | null>(null)
const arcs = ref<StoryArc[]>([])
const characters = ref<PlanningCharacter[]>([])
const contentProfile = ref<StoryContentProfile | null>(null)
const readiness = ref<ReadinessWithPolicy | null>(null)
const workflowLoaded = ref(false)

const loading = ref(false)
const mutating = ref(false)
const error = ref('')
const notice = ref('')
const premise = ref('')

const settings = reactive<StoryWorkflowSettings>({
  story_id: '',
  batch_generation_size: 0,
  creative_autonomy: '',
  preferred_text_provider: '',
  preferred_text_model: '',
  preferred_tts_provider: '',
  preferred_voice_id: '',
  pause_before_tts: false,
  auto_ai_review: false,
  planning_horizon: 0,
  fallback_policy: {},
})
const fallbackPolicyJSON = ref('{}')

const profileDraft = reactive({
  maturity_target: '',
  allowed_themes: '',
  disallowed_themes: '',
  violence_level: '',
  language_limits: '',
  romance_limits: '',
  constraints: '{}',
})

const unresolvedDecisions = computed(() =>
  decisions.value.filter((item) => item.Status !== 'SELECTED' && item.Status !== 'REJECTED'),
)
const canActivate = computed(() => story.value?.status === 'DRAFT' && readiness.value?.ready === true)
const canMakePublic = computed(() =>
  story.value?.visibility === 'PRIVATE' &&
  (story.value?.status === 'ACTIVE' || story.value?.status === 'COMPLETED'),
)

const readinessLabels: Record<string, string> = {
  planning_mode: 'Planning Mode',
  generation_policy: 'Generation Policy',
  content_profile: 'Content Profile',
  story_bible: 'Story Bible',
  ending_plan: 'Ending Plan',
  initial_arc: 'Initial Arc',
  main_character: 'Main Character',
  planning_foundation: 'Planning foundation',
}

function labelForMissing(item: string): string {
  return readinessLabels[item] ?? item.replaceAll('_', ' ')
}

function asJSON(value: unknown): string {
  return JSON.stringify(value ?? {}, null, 2)
}

function syncSettings(value: StoryWorkflowSettings | null) {
  workflowLoaded.value = value !== null
  settings.story_id = value?.story_id ?? storyID.value
  settings.batch_generation_size = value?.batch_generation_size ?? 0
  settings.creative_autonomy = value?.creative_autonomy ?? ''
  settings.preferred_text_provider = value?.preferred_text_provider ?? ''
  settings.preferred_text_model = value?.preferred_text_model ?? ''
  settings.preferred_tts_provider = value?.preferred_tts_provider ?? ''
  settings.preferred_voice_id = value?.preferred_voice_id ?? ''
  settings.pause_before_tts = value?.pause_before_tts ?? false
  settings.auto_ai_review = value?.auto_ai_review ?? false
  settings.planning_horizon = value?.planning_horizon ?? 0
  settings.fallback_policy = value?.fallback_policy ?? {}
  fallbackPolicyJSON.value = asJSON(settings.fallback_policy)
}

function syncProfile(value: StoryContentProfile | null) {
  contentProfile.value = value
  const profile = value?.profile ?? {}
  profileDraft.maturity_target = String(profile.maturity_target ?? '')
  profileDraft.allowed_themes = Array.isArray(profile.allowed_themes)
    ? profile.allowed_themes.join(', ')
    : ''
  profileDraft.disallowed_themes = Array.isArray(profile.disallowed_themes)
    ? profile.disallowed_themes.join(', ')
    : ''
  profileDraft.violence_level = String(profile.violence_level ?? '')
  profileDraft.language_limits = String(profile.language_limits ?? '')
  profileDraft.romance_limits = String(profile.romance_limits ?? '')
  profileDraft.constraints = asJSON(profile.constraints)
}

async function optional<T>(promise: Promise<T>): Promise<T | null> {
  try {
    return await promise
  } catch {
    return null
  }
}

async function loadWorkspace() {
  loading.value = true
  error.value = ''
  try {
    const [storyList, chapterList, decisionList, arcList, characterList, ready] = await Promise.all([
      listAdminStories(),
      listAdminChapters(storyID.value),
      listCreativeDecisions(storyID.value),
      listStoryArcs(storyID.value),
      listStoryCharacters(storyID.value),
      getActivationReadiness(storyID.value),
    ])

    story.value = storyList.stories.find((item) => item.id === storyID.value) ?? null
    chapters.value = chapterList.chapters
    decisions.value = decisionList.decisions
    arcs.value = arcList.arcs
    characters.value = characterList.characters
    readiness.value = ready as ReadinessWithPolicy

    if (!story.value) {
      error.value = 'Không tìm thấy Story trong workspace Admin.'
      return
    }

    const [workflow, currentProfile, currentBible, currentEnding] = await Promise.all([
      optional(getStoryWorkflowSettings(storyID.value)),
      optional(getStoryContentProfile(storyID.value)),
      optional(getStoryBible(storyID.value)),
      optional(getStoryEnding(storyID.value)),
    ])
    syncSettings(workflow)
    syncProfile(currentProfile)
    bible.value = currentBible
    ending.value = currentEnding
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Không thể tải Planning Studio.'
  } finally {
    loading.value = false
  }
}

async function runMutation(message: string, action: () => Promise<unknown>) {
  mutating.value = true
  error.value = ''
  notice.value = ''
  try {
    await action()
    notice.value = message
    await loadWorkspace()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Thao tác thất bại.'
  } finally {
    mutating.value = false
  }
}

async function saveWorkflowSettings() {
  let fallback: Record<string, unknown>
  try {
    fallback = JSON.parse(fallbackPolicyJSON.value) as Record<string, unknown>
  } catch {
    error.value = 'Fallback Policy phải là JSON hợp lệ.'
    return
  }

  await runMutation('Đã lưu Workflow Settings cho các work tương lai.', async () => {
    await updateStoryWorkflowSettings(storyID.value, {
      ...settings,
      story_id: storyID.value,
      fallback_policy: fallback,
    })
  })
}

async function saveContentProfileVersion() {
  let constraints: Record<string, unknown>
  try {
    constraints = JSON.parse(profileDraft.constraints || '{}') as Record<string, unknown>
  } catch {
    error.value = 'Content Profile constraints phải là JSON hợp lệ.'
    return
  }

  const split = (value: string) => value.split(',').map((item) => item.trim()).filter(Boolean)
  await runMutation('Đã tạo Content Profile version mới.', async () => {
    await createStoryContentProfile(storyID.value, {
      maturity_target: profileDraft.maturity_target,
      allowed_themes: split(profileDraft.allowed_themes),
      disallowed_themes: split(profileDraft.disallowed_themes),
      violence_level: profileDraft.violence_level,
      language_limits: profileDraft.language_limits,
      romance_limits: profileDraft.romance_limits,
      constraints,
    })
  })
}

async function generateFoundation() {
  if (!premise.value.trim()) {
    error.value = 'Cần nhập premise trước khi tạo planning foundation.'
    return
  }
  await runMutation('Đã tạo Story foundation theo contract hiện tại.', async () => {
    await generateStoryFoundation(storyID.value, premise.value.trim())
  })
}

async function confirmAndRun(prompt: string, message: string, action: () => Promise<unknown>) {
  if (!window.confirm(prompt)) return
  await runMutation(message, action)
}

onMounted(loadWorkspace)
</script>

<template>
  <section class="page admin planning-studio">
    <div class="section-heading">
      <div>
        <p class="eyebrow">Studio / Story Planning</p>
        <h1>{{ story?.title ?? 'Planning Studio' }}</h1>
        <p class="page-intro">Workspace quản trị planning, readiness và lifecycle theo backend/domain truth.</p>
      </div>
      <div class="story-row-actions">
        <RouterLink class="control-link" to="/admin">Danh sách truyện</RouterLink>
        <RouterLink class="control-link" :to="`/admin/stories/${storyID}/control`">Trung tâm điều khiển</RouterLink>
        <RouterLink class="control-link" :to="`/admin/stories/${storyID}/review`">Duyệt nội dung</RouterLink>
      </div>
    </div>

    <p v-if="loading" class="status-state" role="status">Đang tải Planning Studio...</p>
    <div v-else-if="error && !story" class="status-state error" role="alert">
      <strong>Không thể tải workspace.</strong>
      <p>{{ error }}</p>
      <button type="button" class="secondary-link" @click="loadWorkspace">Thử lại</button>
    </div>

    <template v-else-if="story">
      <p v-if="notice" class="status-state" role="status">{{ notice }}</p>
      <p v-if="error" class="status-state error" role="alert">{{ error }}</p>

      <div class="admin-workspace">
        <section class="admin-list-panel">
          <div class="section-heading">
            <div><p class="eyebrow">Lifecycle</p><h2>Trạng thái Story</h2></div>
          </div>
          <dl class="planning-summary">
            <div><dt>Slug</dt><dd>{{ story.slug }}</dd></div>
            <div><dt>Trạng thái</dt><dd><span class="badge">{{ story.status }}</span></dd></div>
            <div><dt>Hiển thị</dt><dd><span class="badge">{{ story.visibility }}</span></dd></div>
            <div><dt>Mô tả</dt><dd>{{ story.description || 'Chưa có mô tả.' }}</dd></div>
          </dl>
          <div class="action-row">
            <button type="button" :disabled="mutating || !canActivate" @click="runMutation('Story đã ACTIVE.', () => activateStory(storyID))">Activate</button>
            <button v-if="story.status !== 'ARCHIVED'" type="button" class="secondary-link" :disabled="mutating" @click="confirmAndRun('Archive Story này?', 'Story đã được archive.', () => archiveStory(storyID))">Archive</button>
            <button v-else type="button" class="secondary-link" :disabled="mutating" @click="confirmAndRun('Restore Story này?', 'Story đã được restore.', () => restoreStory(storyID))">Restore</button>
            <button v-if="story.visibility === 'PRIVATE'" type="button" class="secondary-link" :disabled="mutating || !canMakePublic" @click="confirmAndRun('Đưa Story ra PUBLIC?', 'Story đã PUBLIC.', () => makeStoryPublic(storyID))">Make public</button>
            <button v-else type="button" class="secondary-link" :disabled="mutating" @click="confirmAndRun('Chuyển Story về PRIVATE?', 'Story đã PRIVATE.', () => makeStoryPrivate(storyID))">Make private</button>
          </div>
        </section>

        <section class="admin-list-panel">
          <div class="section-heading">
            <div><p class="eyebrow">Activation Gate</p><h2>Readiness</h2></div>
            <span class="badge">{{ readiness?.ready ? 'READY' : 'BLOCKED' }}</span>
          </div>
          <p v-if="readiness?.ready" class="status-state">Backend xác nhận đủ điều kiện activation.</p>
          <ul v-else class="readiness-list">
            <li v-for="item in readiness?.missing ?? []" :key="item">
              <strong>{{ labelForMissing(item) }}</strong>
              <span>Cần hoàn tất trước khi Activate.</span>
            </li>
          </ul>
        </section>
      </div>

      <section class="admin-list-panel">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Immutable contract</p>
            <h2>Generation Policy</h2>
            <p class="muted">Snapshot read-only được resolve khi Story được tạo; workspace không cung cấp mutation.</p>
          </div>
        </div>
        <dl v-if="readiness?.generation_policy" class="planning-summary">
          <div><dt>Policy version</dt><dd>{{ readiness.generation_policy.policy_version }}</dd></div>
          <div><dt>Audio duration</dt><dd>{{ readiness.generation_policy.minimum_audio_duration_sec }}s – {{ readiness.generation_policy.target_audio_duration_sec }}s</dd></div>
          <div><dt>Content origin</dt><dd>{{ readiness.generation_policy.content_origin }}</dd></div>
          <div><dt>Language</dt><dd>{{ readiness.generation_policy.language }}</dd></div>
          <div><dt>Narration</dt><dd>{{ readiness.generation_policy.narration_language }}</dd></div>
        </dl>
        <p v-else class="empty-state">Generation Policy chưa tồn tại hoặc không thể đọc.</p>
      </section>

      <section class="admin-list-panel">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Mutable future-work controls</p>
            <h2>Workflow Settings</h2>
            <p class="muted">Các thay đổi này áp dụng theo contract backend cho work tương lai.</p>
          </div>
          <span class="badge">{{ workflowLoaded ? 'CONFIGURED' : 'NOT CONFIGURED' }}</span>
        </div>
        <form class="planning-form" @submit.prevent="saveWorkflowSettings">
          <label>Batch generation size<input v-model.number="settings.batch_generation_size" type="number" min="0"></label>
          <label>Creative autonomy<input v-model="settings.creative_autonomy" type="text"></label>
          <label>Text provider<input v-model="settings.preferred_text_provider" type="text"></label>
          <label>Text model<input v-model="settings.preferred_text_model" type="text"></label>
          <label>TTS provider<input v-model="settings.preferred_tts_provider" type="text"></label>
          <label>Voice ID<input v-model="settings.preferred_voice_id" type="text"></label>
          <label>Planning horizon<input v-model.number="settings.planning_horizon" type="number" min="0"></label>
          <label class="check-field"><input v-model="settings.pause_before_tts" type="checkbox"> Pause before TTS</label>
          <label class="check-field"><input v-model="settings.auto_ai_review" type="checkbox"> Auto AI review</label>
          <label class="full-field">Fallback Policy (JSON)<textarea v-model="fallbackPolicyJSON" rows="5" spellcheck="false"></textarea></label>
          <div class="full-field"><button type="submit" :disabled="mutating">Lưu Workflow Settings</button></div>
        </form>
      </section>

      <section class="admin-list-panel">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Versioned safety profile</p>
            <h2>Content Profile</h2>
            <p class="muted">Mỗi lần lưu tạo một version mới; không rewrite lịch sử.</p>
          </div>
          <span class="badge">v{{ contentProfile?.version_no ?? 0 }}</span>
        </div>
        <form class="planning-form" @submit.prevent="saveContentProfileVersion">
          <label>Maturity target<input v-model="profileDraft.maturity_target" type="text"></label>
          <label>Violence level<input v-model="profileDraft.violence_level" type="text"></label>
          <label>Allowed themes<input v-model="profileDraft.allowed_themes" type="text" placeholder="comma-separated"></label>
          <label>Disallowed themes<input v-model="profileDraft.disallowed_themes" type="text" placeholder="comma-separated"></label>
          <label>Language limits<input v-model="profileDraft.language_limits" type="text"></label>
          <label>Romance limits<input v-model="profileDraft.romance_limits" type="text"></label>
          <label class="full-field">Constraints (JSON)<textarea v-model="profileDraft.constraints" rows="5" spellcheck="false"></textarea></label>
          <div class="full-field"><button type="submit" :disabled="mutating">Tạo Content Profile version mới</button></div>
        </form>
      </section>

      <section class="admin-list-panel">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Planning foundation</p>
            <h2>Bible · Ending · Arcs · Characters</h2>
            <p class="muted">Foundation generation dùng backend Architect và persist qua versioned domain contracts hiện tại.</p>
          </div>
        </div>
        <form class="foundation-form" @submit.prevent="generateFoundation">
          <label>Premise<textarea v-model="premise" rows="4" placeholder="Premise dùng cho Story Architect"></textarea></label>
          <button type="submit" :disabled="mutating || !premise.trim()">Generate foundation</button>
        </form>

        <div class="admin-workspace artifact-grid">
          <article class="artifact-card">
            <div class="section-heading"><h3>Story Bible</h3><span class="badge">v{{ bible?.VersionNo ?? 0 }}</span></div>
            <pre v-if="bible">{{ asJSON(bible.Content) }}</pre>
            <p v-else class="empty-state">Chưa có Story Bible.</p>
          </article>
          <article class="artifact-card">
            <div class="section-heading"><h3>Ending Plan</h3><span class="badge">v{{ ending?.VersionNo ?? 0 }}</span></div>
            <pre v-if="ending">{{ asJSON(ending.Content) }}</pre>
            <p v-else class="empty-state">Chưa có Ending Plan.</p>
          </article>
        </div>

        <div class="story-table-wrap">
          <table class="story-table">
            <thead><tr><th>Arc</th><th>Status</th><th>Current version</th></tr></thead>
            <tbody>
              <tr v-for="arc in arcs" :key="arc.ID">
                <th scope="row">Arc {{ arc.Ordinal }}</th><td>{{ arc.Status }}</td><td>{{ arc.CurrentVersionID || '—' }}</td>
              </tr>
              <tr v-if="arcs.length === 0"><td colspan="3">Chưa có Arc.</td></tr>
            </tbody>
          </table>
        </div>

        <div class="story-table-wrap">
          <table class="story-table">
            <thead><tr><th>Character</th><th>Importance</th><th>Current profile</th></tr></thead>
            <tbody>
              <tr v-for="character in characters" :key="character.ID">
                <th scope="row">{{ character.CanonicalName }}</th><td>{{ character.Importance }}</td><td>{{ character.CurrentProfileVersionID || '—' }}</td>
              </tr>
              <tr v-if="characters.length === 0"><td colspan="3">Chưa có Character.</td></tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="admin-list-panel" aria-labelledby="planning-chapters-heading">
        <div class="section-heading">
          <div>
            <p class="eyebrow">Chapter plan surface</p>
            <h2 id="planning-chapters-heading">Chapter Plans</h2>
            <p class="muted">Hiển thị pointer tới current plan revision; workspace không flatten lịch sử versioned thành CRUD phá huỷ.</p>
          </div>
          <span class="count-label">{{ chapters.length }}</span>
        </div>
        <p v-if="chapters.length === 0" class="empty-state">Chưa có chapter.</p>
        <div v-else class="story-table-wrap">
          <table class="story-table">
            <thead><tr><th>Chapter</th><th>Trạng thái</th><th>Arc</th><th>Current plan</th></tr></thead>
            <tbody>
              <tr v-for="chapter in chapters" :key="chapter.ID">
                <th scope="row">#{{ chapter.ChapterNumber }} — {{ chapter.Title }}</th>
                <td><span class="badge">{{ chapter.Status }}</span></td>
                <td>{{ chapter.ArcID || '—' }}</td>
                <td>{{ (chapter as Chapter & { CurrentPlanRevisionID?: string }).CurrentPlanRevisionID || '—' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="admin-list-panel">
        <div class="section-heading">
          <div><p class="eyebrow">Decision pressure</p><h2>Creative Decisions</h2></div>
          <span class="count-label">{{ unresolvedDecisions.length }} pending</span>
        </div>
        <p class="muted">{{ decisions.length }} decision records hiện có. Control Center vẫn là entry point xử lý chi tiết.</p>
      </section>
    </template>
  </section>
</template>

<style scoped>
.planning-studio { display: grid; gap: 1.5rem; }
.planning-summary { display: grid; gap: 0.75rem; margin: 0; }
.planning-summary > div { display: grid; grid-template-columns: minmax(9rem, .35fr) 1fr; gap: 1rem; padding-block: .75rem; border-bottom: 1px solid var(--border, #d9d9d9); }
.planning-summary dt { font-weight: 700; }
.planning-summary dd { margin: 0; overflow-wrap: anywhere; }
.action-row { display: flex; flex-wrap: wrap; gap: .75rem; margin-top: 1rem; }
.readiness-list { display: grid; gap: .65rem; margin: 0; padding: 0; list-style: none; }
.readiness-list li { display: flex; justify-content: space-between; gap: 1rem; padding: .75rem; border: 1px solid var(--border, #d9d9d9); border-radius: .5rem; }
.planning-form { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; }
.planning-form label, .foundation-form label { display: grid; gap: .4rem; font-weight: 600; }
.planning-form input, .planning-form textarea, .foundation-form textarea { width: 100%; box-sizing: border-box; }
.full-field { grid-column: 1 / -1; }
.check-field { display: flex !important; align-items: center; gap: .5rem !important; }
.foundation-form { display: grid; gap: .75rem; margin-bottom: 1.25rem; }
.artifact-grid { margin-block: 1rem; }
.artifact-card { min-width: 0; padding: 1rem; border: 1px solid var(--border, #d9d9d9); border-radius: .65rem; }
.artifact-card pre { max-height: 22rem; overflow: auto; white-space: pre-wrap; overflow-wrap: anywhere; font-size: .82rem; }
@media (max-width: 760px) {
  .planning-form { grid-template-columns: 1fr; }
  .full-field { grid-column: auto; }
  .planning-summary > div { grid-template-columns: 1fr; gap: .25rem; }
  .readiness-list li { display: grid; }
}
</style>
