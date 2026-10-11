<template>
  <div class="services-page">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">
          <AppIcon name="Terminal" :size="24" class="text-green" />
          FreeBSD rc.d Daemons & System Logs
        </h1>
        <p class="page-subtitle">
          Kontrol service sistem FreeBSD (/usr/sbin/service) dan live monitoring log audit keamanan
        </p>
      </div>

      <div class="header-actions">
        <button class="btn btn-secondary" @click="refreshLogs">
          <AppIcon name="RefreshCw" :size="15" />
          Segarkan Log
        </button>
      </div>
    </div>

    <!-- FreeBSD Services Table -->
    <div class="bsd-card">
      <div class="bsd-card-header">
        <h2 class="bsd-card-title">
          <AppIcon name="Server" :size="18" class="text-red" />
          Daftar Daemon / rc.d Services FreeBSD
        </h2>
        <span class="text-muted font-mono" style="font-size: 0.8rem;">
          Tersinkron dengan /etc/rc.conf via sysrc
        </span>
      </div>

      <div class="table-responsive">
        <table class="bsd-table">
          <thead>
            <tr>
              <th>Nama Service</th>
              <th>Deskripsi Daemon</th>
              <th>Status Service</th>
              <th>Boot Startup (/etc/rc.conf)</th>
              <th style="text-align: right;">Aksi Kontrol</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="s in store.systemServices" :key="s.name">
              <td>
                <span class="font-mono text-bold">{{ s.name }}</span>
              </td>
              <td class="text-muted">{{ s.description }}</td>
              <td>
                <span class="badge" :class="s.is_running ? 'badge-green' : 'badge-red'">
                  {{ s.is_running ? 'RUNNING' : 'STOPPED' }}
                </span>
              </td>
              <td>
                <span class="badge badge-purple font-mono">
                  {{ s.is_enabled_boot ? 'YES' : 'NO' }}
                </span>
              </td>
              <td style="text-align: right;">
                <div class="srv-btn-group">
                  <button
                    class="btn btn-sm btn-secondary"
                    title="Reload Service"
                    @click="executeAction(s.name, 'reload')"
                  >
                    <AppIcon name="RotateCw" :size="12" />
                    Reload
                  </button>
                  <button
                    class="btn btn-sm"
                    :class="s.is_running ? 'btn-danger' : 'btn-success'"
                    @click="store.toggleService(s.name)"
                  >
                    {{ s.is_running ? 'Stop' : 'Start' }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Live System Log Viewer -->
    <div class="bsd-card">
      <div class="bsd-card-header">
        <h2 class="bsd-card-title">
          <AppIcon name="FileText" :size="18" class="text-blue" />
          Live Log Viewer: FreeBSD Kernel & Security
        </h2>

        <div class="log-tabs">
          <button
            class="btn btn-sm"
            :class="selectedLog === 'messages' ? 'btn-primary' : 'btn-secondary'"
            @click="selectedLog = 'messages'"
          >
            /var/log/messages
          </button>
          <button
            class="btn btn-sm"
            :class="selectedLog === 'pf' ? 'btn-primary' : 'btn-secondary'"
            @click="selectedLog = 'pf'"
          >
            Packet Filter (pflog)
          </button>
          <button
            class="btn btn-sm"
            :class="selectedLog === 'fail2ban' ? 'btn-primary' : 'btn-secondary'"
            @click="selectedLog = 'fail2ban'"
          >
            Fail2ban Actions
          </button>
        </div>
      </div>

      <div class="terminal-window">
        <div class="term-header">
          <div class="term-dots">
            <span class="dot dot-red"></span>
            <span class="dot dot-amber"></span>
            <span class="dot dot-green"></span>
          </div>
          <div class="term-title font-mono">
            root@bsd-node01:/var/log/{{ selectedLog }}
          </div>
        </div>

        <pre class="term-body font-mono"><code>{{ currentLogContent }}</code></pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useBsdStore } from '../stores/bsdStore'
import AppIcon from '../components/AppIcon.vue'

const store = useBsdStore()

const selectedLog = ref('messages')

const logData = ref({
  messages: [
    'Oct 11 19:42:01 bsd-node01 kernel: [vtnet0] link state changed to UP',
    'Oct 11 19:42:02 bsd-node01 pf: Rule 14 pass in quick on vtnet0 proto tcp from 198.51.100.23 to any port 80',
    'Oct 11 19:42:05 bsd-node01 bsdpaneld[1204]: Authentication attempt for root from 127.0.0.1: SUCCESS',
    'Oct 11 19:42:10 bsd-node01 php-fpm[3812]: [pool example.com] child 3822 started',
    'Oct 11 19:42:15 bsd-node01 syslogd: restart',
    'Oct 11 19:44:00 bsd-node01 newsyslog[4201]: logfile turned over /var/log/messages'
  ],
  pf: [
    'Oct 11 19:40:11 pf: block drop in on vtnet0: 203.0.113.88.42391 > 10.0.0.1.22: [S] (table <bruteforce>)',
    'Oct 11 19:40:12 pf: block drop in on vtnet0: 198.51.100.23.51882 > 10.0.0.1.22: [S] (table <bruteforce>)',
    'Oct 11 19:41:00 pf: pass in on vtnet0: 172.56.21.9.51234 > 10.0.0.1.443: [S] keep state',
    'Oct 11 19:41:00 pf: pass out on vtnet0: 10.0.0.1.443 > 172.56.21.9.51234: [S.] keep state'
  ],
  fail2ban: [
    '2026-10-11 19:40:02 fail2ban.filter [1450]: INFO [bsd-sshd] Found 198.51.100.23',
    '2026-10-11 19:40:03 fail2ban.actions [1450]: NOTICE [bsd-sshd] Ban 198.51.100.23',
    '2026-10-11 19:41:20 fail2ban.filter [1450]: INFO [nginx-badbots] Found 45.140.19.8 - 404 scanning',
    '2026-10-11 19:41:21 fail2ban.actions [1450]: NOTICE [nginx-badbots] Ban 45.140.19.8'
  ]
})

const currentLogContent = computed(() => {
  return logData.value[selectedLog.value].join('\n')
})

function executeAction(name, action) {
  alert(`FreeBSD command '/usr/sbin/service ${name} ${action}' berhasil dieksekusi!`)
}

function refreshLogs() {
  alert('Log FreeBSD berhasil disegarkan!')
}
</script>

<style scoped>
.services-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 10px;
}

.page-subtitle {
  color: var(--text-secondary);
  font-size: 0.875rem;
  margin-top: 4px;
}

.srv-btn-group {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}

.log-tabs {
  display: flex;
  align-items: center;
  gap: 8px;
}

.terminal-window {
  background: #05070a;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.term-header {
  background: #0f1420;
  padding: 8px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-color);
}

.term-dots {
  display: flex;
  align-items: center;
  gap: 6px;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.dot-red { background: #ef4444; }
.dot-amber { background: #f59e0b; }
.dot-green { background: #10b981; }

.term-title {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.term-body {
  padding: 16px;
  font-size: 0.8rem;
  color: #a5f3fc;
  line-height: 1.6;
  max-height: 320px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
