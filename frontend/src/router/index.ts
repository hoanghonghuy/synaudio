import { createRouter, createWebHistory } from 'vue-router'
import HomePage from '../features/catalog/HomePage.vue'
import AdminShell from '../features/admin/AdminShell.vue'
import StoryReader from '../features/reader/StoryReader.vue'
import StoryControlCenter from '../features/admin/StoryControlCenter.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomePage,
    },
    {
      path: '/admin',
      name: 'admin',
      component: AdminShell,
    },
    {
      path: '/stories/:storyID/read',
      name: 'reader',
      component: StoryReader,
    },
    {
      path: '/admin/stories/:storyID/control',
      name: 'control-center',
      component: StoryControlCenter,
    },
  ],
})
