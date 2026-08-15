import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './app/App.vue'
import { router } from './router'
import './styles.css'

createApp(App).use(createPinia()).use(router).mount('#app')
