import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import WebserversView from '../views/WebserversView.vue'
import PhpManagerView from '../views/PhpManagerView.vue'
import DatabasesView from '../views/DatabasesView.vue'
import FirewallView from '../views/FirewallView.vue'
import UsersView from '../views/UsersView.vue'
import ServicesView from '../views/ServicesView.vue'

const routes = [
  { path: '/', name: 'Dashboard', component: DashboardView },
  { path: '/webservers', name: 'Webservers', component: WebserversView },
  { path: '/php', name: 'PhpManager', component: PhpManagerView },
  { path: '/databases', name: 'Databases', component: DatabasesView },
  { path: '/firewall', name: 'Firewall', component: FirewallView },
  { path: '/users', name: 'Users', component: UsersView },
  { path: '/services', name: 'Services', component: ServicesView },
  { path: '/:pathMatch(.*)*', redirect: '/' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
