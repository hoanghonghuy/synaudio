<script setup lang="ts">
import { computed } from 'vue'
import { presentCanonWorkspace } from './canonWorkspacePresentation.mjs'

const props = defineProps<{
  state: Record<string, unknown>
  disabled?: boolean
}>()

const emit = defineEmits<{
  commit: []
  retry: []
}>()

const view = computed(() => presentCanonWorkspace(props.state))
const commitDisabled = computed(() => props.disabled || !view.value.canCommit)
const retryDisabled = computed(() => props.disabled || !view.value.canRetry)
</script>

<template>
  <section class="canon-panel" aria-labelledby="canon-memory-title" :aria-busy="view.busy">
    <div class="canon-heading">
      <div>
        <p class="eyebrow">Story authority</p>
        <h3 id="canon-memory-title">Canon / Memory</h3>
      </div>
      <span :class="['canon-badge', `canon-${view.status.toLowerCase()}`]">{{ view.badge }}</span>
    </div>

    <p class="canon-summary" :role="view.status === 'BLOCKED' ? 'alert' : 'status'">{{ view.summary }}</p>
    <p class="canon-detail">{{ view.detail }}</p>

    <div v-if="view.canCommit || view.canRetry || view.busy" class="canon-actions">
      <button v-if="view.canCommit || view.busy" type="button" class="canon-primary" :disabled="commitDisabled" @click="emit('commit')">
        {{ view.busy ? 'Đang commit Canon/Memory…' : 'Commit Canon / Memory' }}
      </button>
      <button v-if="view.canRetry" type="button" class="canon-secondary" :disabled="retryDisabled" @click="emit('retry')">Retry authority</button>
    </div>
  </section>
</template>

<style scoped>
.canon-panel { display: grid; gap: 12px; padding: 18px; border: 1px solid var(--border-color, #d8d8d8); border-radius: 16px; background: var(--surface-color, #fff); }
.canon-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.canon-heading h3, .canon-heading p, .canon-summary, .canon-detail { margin: 0; }
.eyebrow { font-size: 0.72rem; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; opacity: 0.68; }
.canon-badge { flex: 0 0 auto; max-width: 48%; padding: 5px 9px; border: 1px solid currentColor; border-radius: 999px; font-size: 0.72rem; font-weight: 750; text-align: center; overflow-wrap: anywhere; }
.canon-summary { font-weight: 650; line-height: 1.45; overflow-wrap: anywhere; }
.canon-detail { line-height: 1.5; opacity: 0.78; overflow-wrap: anywhere; }
.canon-actions { display: flex; flex-wrap: wrap; gap: 10px; }
.canon-actions button { min-height: 44px; padding: 10px 14px; border-radius: 10px; font: inherit; font-weight: 700; cursor: pointer; }
.canon-actions button:disabled { cursor: not-allowed; opacity: 0.55; }
.canon-primary { border: 1px solid transparent; background: var(--accent-color, #315efb); color: #fff; }
.canon-secondary { border: 1px solid var(--border-color, #c8c8c8); background: transparent; color: inherit; }
.canon-actions button:focus-visible { outline: 3px solid currentColor; outline-offset: 3px; }
.canon-current { opacity: 0.82; }
.canon-blocked { font-weight: 800; }
@media (max-width: 640px) {
  .canon-panel { padding: 15px; }
  .canon-heading { align-items: stretch; flex-direction: column; gap: 10px; }
  .canon-badge { max-width: 100%; align-self: flex-start; }
  .canon-actions { display: grid; }
  .canon-actions button { width: 100%; }
}
</style>
