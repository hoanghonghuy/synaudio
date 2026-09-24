<script setup lang="ts">
import { RouterLink } from 'vue-router'

defineProps<{
  storyID?: string
}>()
</script>

<template>
  <section class="admin-workspace-nav" aria-label="Creator Studio workspace">
    <div class="admin-workspace-nav-brand">
      <span class="admin-workspace-nav-mark" aria-hidden="true">✦</span>
      <div>
        <p class="eyebrow">Synaudio / Creator Studio</p>
        <strong>{{ storyID ? 'Story workspace' : 'Operations workspace' }}</strong>
      </div>
    </div>

    <nav v-if="storyID" class="admin-workspace-nav-links" aria-label="Điều hướng story workspace">
      <RouterLink :to="`/admin/stories/${storyID}/control`">Control</RouterLink>
      <RouterLink :to="`/admin/stories/${storyID}/planning`">Planning</RouterLink>
      <RouterLink :to="`/admin/stories/${storyID}/production`">Production</RouterLink>
      <RouterLink :to="`/admin/stories/${storyID}/review`">Review</RouterLink>
      <RouterLink :to="`/admin/stories/${storyID}/retcons`">Retcons</RouterLink>
    </nav>

    <div class="admin-workspace-nav-tools">
      <RouterLink to="/admin">Stories</RouterLink>
      <RouterLink to="/admin/audit">Audit</RouterLink>
      <RouterLink to="/admin/security">Security</RouterLink>
    </div>
  </section>
</template>
