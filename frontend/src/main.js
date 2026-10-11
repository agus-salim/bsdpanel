import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Dashboard from './views/Dashboard.vue'
import Websites from './views/Websites.vue'
import Terminal from './views/Terminal.vue'
import './style.css'

const routes = [
  { path: '/', name: 'Dashboard', component: Dashboard },
  { path: '/websites', name: 'Websites', component: Websites },
  { path: '/terminal', name: 'Terminal', component: Terminal },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
