<template>
  <div class="space-y-8 max-w-7xl mx-auto">
    <!-- Header Title -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-2xl font-bold tracking-tight text-white">FreeBSD System Overview</h2>
        <p class="text-sm text-slate-400 mt-1">Status node host, utilitas ZFS ARC, FreeBSD Jails, dan daemon hosting.</p>
      </div>
      <div class="flex items-center gap-2">
        <button @click="fetchData" class="px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium rounded-lg border border-slate-700 transition flex items-center gap-1.5">
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
          Refresh Data
        </button>
      </div>
    </div>

    <!-- Metric Cards Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5">
      <!-- OS & Kernel -->
      <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm">
        <div class="flex items-center justify-between text-slate-400 mb-2">
          <span class="text-xs font-semibold uppercase tracking-wider">Sistem Operasi</span>
          <span class="p-1.5 bg-rose-500/10 text-rose-400 rounded-md">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M12 5l7 7-7 7"/></svg>
          </span>
        </div>
        <div class="text-xl font-bold text-white tracking-tight">{{ stats.system.kernel_release || 'FreeBSD 14.1-RELEASE' }}</div>
        <div class="text-xs text-slate-500 mt-1 font-mono">Host: {{ stats.system.hostname || 'freebsd-panel.local' }}</div>
      </div>

      <!-- CPU Load -->
      <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm">
        <div class="flex items-center justify-between text-slate-400 mb-2">
          <span class="text-xs font-semibold uppercase tracking-wider">Load Average</span>
          <span class="p-1.5 bg-amber-500/10 text-amber-400 rounded-md">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
          </span>
        </div>
        <div class="text-xl font-bold text-white tracking-tight">
          {{ stats.system.load_avg_1?.toFixed(2) || '0.24' }}, {{ stats.system.load_avg_5?.toFixed(2) || '0.38' }}, {{ stats.system.load_avg_15?.toFixed(2) || '0.15' }}
        </div>
        <div class="text-xs text-slate-500 mt-1">1m, 5m, 15m sysctl load</div>
      </div>

      <!-- Memory & ZFS ARC -->
      <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm">
        <div class="flex items-center justify-between text-slate-400 mb-2">
          <span class="text-xs font-semibold uppercase tracking-wider">RAM & ZFS ARC</span>
          <span class="p-1.5 bg-blue-500/10 text-blue-400 rounded-md">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"/></svg>
          </span>
        </div>
        <div class="text-xl font-bold text-white tracking-tight">
          {{ (stats.system.memory_used_mb / 1024).toFixed(1) || '4.7' }} GB <span class="text-xs font-normal text-slate-500">/ {{ (stats.system.memory_total_mb / 1024).toFixed(1) || '16' }} GB</span>
        </div>
        <div class="text-xs text-blue-400 mt-1 font-mono">ARC Cache: {{ (stats.system.zfs_arc_size_mb / 1024).toFixed(1) || '2.5' }} GB</div>
      </div>

      <!-- Active Jails & Websites -->
      <div class="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-sm">
        <div class="flex items-center justify-between text-slate-400 mb-2">
          <span class="text-xs font-semibold uppercase tracking-wider">Jails & Websites</span>
          <span class="p-1.5 bg-emerald-500/10 text-emerald-400 rounded-md">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/></svg>
          </span>
        </div>
        <div class="text-xl font-bold text-white tracking-tight">
          {{ stats.system.active_jails || 3 }} <span class="text-sm font-normal text-slate-400">Jails</span>
        </div>
        <div class="text-xs text-slate-500 mt-1">Total Web: {{ stats.metrics?.total_websites || 2 }} domain</div>
      </div>
    </div>

    <!-- Daemon / Services Matrix -->
    <div class="bg-slate-900 border border-slate-800 rounded-xl p-6">
      <div class="flex items-center justify-between mb-5">
        <div>
          <h3 class="font-bold text-lg text-white">Status Layanan & Multi-PHP FreeBSD</h3>
          <p class="text-xs text-slate-400">Dikelola langsung via subsystem FreeBSD <code class="text-rose-400">/usr/sbin/service</code></p>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div
          v-for="svc in services"
          :key="svc.name"
          class="p-4 rounded-lg bg-slate-950 border border-slate-800/80 flex items-center justify-between"
        >
          <div class="flex items-center gap-3">
            <div
              class="w-2.5 h-2.5 rounded-full"
              :class="svc.running ? 'bg-emerald-400 shadow-sm shadow-emerald-400/50' : 'bg-rose-500'"
            ></div>
            <div>
              <p class="text-sm font-semibold text-slate-200 font-mono">{{ svc.name }}</p>
              <p class="text-xs text-slate-500">{{ svc.running ? 'Aktif (Running)' : 'Berhenti (Stopped)' }}</p>
            </div>
          </div>

          <div class="flex items-center gap-1">
            <button
              @click="serviceAction(svc.name, 'restart')"
              class="px-2.5 py-1 text-xs font-medium rounded bg-slate-800 hover:bg-slate-700 text-slate-300 transition"
            >
              Restart
            </button>
            <button
              @click="serviceAction(svc.name, 'reload')"
              class="px-2.5 py-1 text-xs font-medium rounded bg-slate-800 hover:bg-slate-700 text-slate-300 transition"
            >
              Reload
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const stats = ref({
  system: {
    kernel_release: 'FreeBSD 14.1-RELEASE',
    hostname: 'freebsd-panel.local',
    load_avg_1: 0.24,
    load_avg_5: 0.38,
    load_avg_15: 0.15,
    memory_used_mb: 4820,
    memory_total_mb: 16384,
    zfs_arc_size_mb: 2540,
    active_jails: 3,
  },
  metrics: {
    total_websites: 2,
    total_databases: 1,
  }
})

const services = ref([
  { name: 'nginx', running: true },
  { name: 'apache24', running: false },
  { name: 'php81_fpm', running: true },
  { name: 'php82_fpm', running: true },
  { name: 'php83_fpm', running: true },
  { name: 'mysql_server', running: true },
  { name: 'postgresql', running: true },
])

const fetchData = async () => {
  try {
    const res = await fetch('/api/dashboard/stats')
    if (res.ok) {
      stats.value = await res.json()
    }
  } catch (err) {
    console.log('Menggunakan data preview lokal', err)
  }
}

const serviceAction = async (service, action) => {
  try {
    const res = await fetch('/api/services/control', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ service, action }),
    })
    if (res.ok) {
      alert(`Berhasil mengeksekusi '${action}' pada service ${service}`)
    }
  } catch (err) {
    alert(`Aksi: service ${service} ${action}`)
  }
}

onMounted(() => {
  fetchData()
})
</script>
