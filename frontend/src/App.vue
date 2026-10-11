<template>
  <div class="app-layout" :data-theme="currentTheme">
    <!-- Mobile Backdrop -->
    <div
      v-if="isSidebarOpen"
      class="sidebar-backdrop"
      @click="isSidebarOpen = false"
    ></div>

    <!-- Sidebar Navigation -->
    <aside class="sidebar" :class="{ 'is-open': isSidebarOpen }">
      <div class="sidebar-header">
        <div class="brand">
          <img src="/freebsd-logo.svg" alt="FreeBSD Logo" class="brand-logo" />
          <div class="brand-text">
            <span class="brand-title">BSD Panel</span>
            <span class="brand-badge font-mono">FreeBSD 14</span>
          </div>
        </div>
        <button class="mobile-close-btn" @click="isSidebarOpen = false">✕</button>
      </div>

      <nav class="sidebar-nav">
        <div class="nav-section-label">NAVIGASI SISTEM</div>

        <router-link to="/" class="nav-item" @click="closeMobileNav">
          <AppIcon name="LayoutDashboard" :size="18" />
          <span>Dashboard &amp; Telemetri</span>
        </router-link>

        <router-link to="/webservers" class="nav-item" @click="closeMobileNav">
          <AppIcon name="Globe" :size="18" />
          <span>Multi Web Server</span>
          <span class="nav-tag font-mono">4</span>
        </router-link>

        <router-link to="/php" class="nav-item" @click="closeMobileNav">
          <AppIcon name="Code" :size="18" />
          <span>PHP Multi-Version</span>
          <span class="nav-tag font-mono">8.2-8.6</span>
        </router-link>

        <router-link to="/databases" class="nav-item" @click="closeMobileNav">
          <AppIcon name="Database" :size="18" />
          <span>Multi Database</span>
          <span class="nav-tag font-mono">PG/Maria</span>
        </router-link>

        <div class="nav-section-label" style="margin-top: 14px;">KEAMANAN &amp; USER</div>

        <router-link to="/firewall" class="nav-item" @click="closeMobileNav">
          <AppIcon name="Shield" :size="18" />
          <span>Firewall (PF) &amp; Fail2ban</span>
          <span class="nav-tag font-mono" style="color: #34d399;">PF</span>
        </router-link>

        <router-link to="/users" class="nav-item" @click="closeMobileNav">
          <AppIcon name="Users" :size="18" />
          <span>User &amp; Reseller Tier</span>
        </router-link>

        <router-link to="/services" class="nav-item" @click="closeMobileNav">
          <AppIcon name="Terminal" :size="18" />
          <span>FreeBSD rc.d &amp; Logs</span>
        </router-link>
      </nav>

      <!-- Sidebar Footer Info -->
      <div class="sidebar-footer">
        <div class="footer-stats font-mono">
          <div class="fs-line">
            <span class="fs-dot"></span>
            <span>pf: ACTIVE ({{ store.stats.pf_states_count }} states)</span>
          </div>
          <div class="fs-line text-muted">
            <span>zpool: ONLINE</span>
          </div>
        </div>
      </div>
    </aside>

    <!-- Main Content Area -->
    <div class="main-wrapper">
      <!-- Top Header Bar -->
      <header class="topbar">
        <div class="topbar-left">
          <button class="mobile-toggle-btn" @click="isSidebarOpen = true">
            <AppIcon name="Menu" :size="20" />
          </button>
          <div class="breadcrumb font-mono">
            <span>FreeBSD Server</span>
            <span class="sep">/</span>
            <span class="current">{{ currentRouteName }}</span>
          </div>
        </div>

        <div class="topbar-right">
          <!-- Theme Toggle -->
          <button
            class="topbar-btn"
            @click="toggleTheme"
            :title="currentTheme === 'dark' ? 'Switch to Light Theme' : 'Switch to Dark Theme'"
          >
            <AppIcon :name="currentTheme === 'dark' ? 'Sun' : 'Moon'" :size="17" />
          </button>

          <!-- Current User Profile Pill -->
          <div class="user-pill font-mono">
            <span class="user-avatar">
              <AppIcon name="Shield" :size="14" class="text-red" />
            </span>
            <span class="user-name">{{ store.currentUser.username }}</span>
            <span class="badge badge-red font-mono" style="font-size: 0.65rem;">
              {{ store.currentUser.role.toUpperCase() }}
            </span>
          </div>
        </div>
      </header>

      <!-- Router View Body -->
      <main class="page-body">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useBsdStore } from './stores/bsdStore'
import AppIcon from './components/AppIcon.vue'

const store = useBsdStore()
const route = useRoute()

const isSidebarOpen = ref(false)
const currentTheme = ref('dark')

function closeMobileNav() {
  isSidebarOpen.value = false
}

function toggleTheme() {
  currentTheme.value = currentTheme.value === 'dark' ? 'light' : 'dark'
  document.documentElement.setAttribute('data-theme', currentTheme.value)
}

const currentRouteName = computed(() => {
  return route.name || 'Dashboard'
})
</script>

<style scoped>
.app-layout {
  display: flex;
  min-height: 100vh;
  background-color: var(--bg-primary);
  color: var(--text-primary);
}

/* Sidebar */
.sidebar {
  width: var(--sidebar-width);
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  position: fixed;
  top: 0;
  bottom: 0;
  left: 0;
  z-index: 100;
  transition: transform 0.3s ease;
}

.sidebar-header {
  height: var(--header-height);
  padding: 0 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-color);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-logo {
  width: 32px;
  height: 32px;
}

.brand-text {
  display: flex;
  flex-direction: column;
}

.brand-title {
  font-size: 1.15rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #fff;
}

.brand-badge {
  font-size: 0.675rem;
  color: var(--accent-red);
  font-weight: 600;
}

.mobile-close-btn {
  display: none;
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 1.25rem;
  cursor: pointer;
}

.sidebar-nav {
  flex: 1;
  padding: 16px 12px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.nav-section-label {
  font-size: 0.675rem;
  font-weight: 700;
  color: var(--text-muted);
  letter-spacing: 0.08em;
  padding: 8px 12px 4px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s ease;
}

.nav-item:hover {
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.nav-item.router-link-active {
  background: var(--accent-red-subtle);
  color: #fff;
  border-left: 3px solid var(--accent-red);
  font-weight: 600;
}

.nav-tag {
  margin-left: auto;
  font-size: 0.7rem;
  background: var(--bg-primary);
  padding: 2px 6px;
  border-radius: 4px;
  color: var(--text-muted);
}

.sidebar-footer {
  padding: 16px;
  border-top: 1px solid var(--border-color);
  background: rgba(10, 13, 20, 0.4);
}

.footer-stats {
  font-size: 0.725rem;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.fs-line {
  display: flex;
  align-items: center;
  gap: 6px;
}

.fs-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #10b981;
}

/* Main Content Wrapper */
.main-wrapper {
  flex: 1;
  margin-left: var(--sidebar-width);
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 100vh;
}

.topbar {
  height: var(--header-height);
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  position: sticky;
  top: 0;
  z-index: 50;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.mobile-toggle-btn {
  display: none;
  background: none;
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: var(--radius-sm);
  padding: 6px;
  cursor: pointer;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.825rem;
  color: var(--text-muted);
}

.breadcrumb .current {
  color: var(--text-primary);
  font-weight: 600;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.topbar-btn {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: var(--radius-sm);
  padding: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.topbar-btn:hover {
  border-color: var(--text-muted);
}

.user-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  padding: 5px 12px;
  border-radius: 9999px;
  font-size: 0.8rem;
}

.user-avatar {
  display: flex;
  align-items: center;
}

.user-name {
  font-weight: 600;
}

/* Mobile Responsiveness */
@media (max-width: 900px) {
  .sidebar {
    transform: translateX(-100%);
  }

  .sidebar.is-open {
    transform: translateX(0);
  }

  .mobile-close-btn {
    display: block;
  }

  .mobile-toggle-btn {
    display: flex;
  }

  .main-wrapper {
    margin-left: 0;
  }

  .sidebar-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    z-index: 99;
  }
}
</style>
