<template>
  <div class="dashboard-page">
    <!-- Header Banner -->
    <div class="page-header">
      <div>
        <h1 class="page-title">
          <AppIcon name="Cpu" :size="24" class="text-red" />
          FreeBSD System Dashboard
        </h1>
        <p class="page-subtitle">
          Host: <span class="font-mono text-bold">{{ store.stats.hostname }}</span> &bull;
          OS: <span class="badge badge-red font-mono">{{ store.stats.os_name }}</span> &bull;
          Uptime: {{ store.stats.uptime_formatted }}
        </p>
      </div>

      <div class="header-actions">
        <button class="btn btn-secondary btn-sm" @click="refreshStats">
          <AppIcon name="RefreshCw" :size="14" />
          Segarkan Data
        </button>
        <span class="live-pill">
          <span class="live-dot"></span> LIVE TELEMETRY
        </span>
      </div>
    </div>

    <!-- Metrics Grid -->
    <div class="metrics-grid">
      <!-- CPU Metric -->
      <div class="bsd-card metric-card">
        <div class="metric-top">
          <div class="metric-label">
            <AppIcon name="Cpu" :size="16" class="text-red" />
            <span>CPU Load</span>
          </div>
          <span class="metric-value font-mono">{{ store.stats.cpu_usage_percent }}%</span>
        </div>
        <div class="progress-bar-bg">
          <div
            class="progress-bar-fill bg-red"
            :style="{ width: `${store.stats.cpu_usage_percent}%` }"
          ></div>
        </div>
        <div class="metric-footer font-mono">
          <span>Load Avg: {{ store.stats.load_average.join(', ') }}</span>
          <span>{{ store.stats.cpu_cores }} Cores</span>
        </div>
      </div>

      <!-- RAM Metric -->
      <div class="bsd-card metric-card">
        <div class="metric-top">
          <div class="metric-label">
            <AppIcon name="Layers" :size="16" class="text-blue" />
            <span>RAM (Memory)</span>
          </div>
          <span class="metric-value font-mono">{{ store.stats.memory_usage_percent }}%</span>
        </div>
        <div class="progress-bar-bg">
          <div
            class="progress-bar-fill bg-blue"
            :style="{ width: `${store.stats.memory_usage_percent}%` }"
          ></div>
        </div>
        <div class="metric-footer font-mono">
          <span>Terpakai: {{ (store.stats.memory_used_mb / 1024).toFixed(1) }} GB</span>
          <span>Total: {{ (store.stats.memory_total_mb / 1024).toFixed(0) }} GB</span>
        </div>
      </div>

      <!-- Storage ZFS Metric -->
      <div class="bsd-card metric-card">
        <div class="metric-top">
          <div class="metric-label">
            <AppIcon name="HardDrive" :size="16" class="text-green" />
            <span>Storage (ZFS zroot)</span>
          </div>
          <span class="metric-value font-mono">{{ store.stats.disk_usage_percent }}%</span>
        </div>
        <div class="progress-bar-bg">
          <div
            class="progress-bar-fill bg-green"
            :style="{ width: `${store.stats.disk_usage_percent}%` }"
          ></div>
        </div>
        <div class="metric-footer font-mono">
          <span>{{ store.stats.disk_used_gb }} GB / {{ store.stats.disk_total_gb }} GB</span>
          <span class="badge badge-green font-mono">ZFS {{ store.stats.zfs_health }}</span>
        </div>
      </div>

      <!-- PF Firewall States -->
      <div class="bsd-card metric-card">
        <div class="metric-top">
          <div class="metric-label">
            <AppIcon name="Shield" :size="16" class="text-amber" />
            <span>Packet Filter (PF)</span>
          </div>
          <span class="metric-value font-mono">{{ store.stats.pf_states_count }} states</span>
        </div>
        <div class="progress-bar-bg">
          <div class="progress-bar-fill bg-amber" style="width: 45%;"></div>
        </div>
        <div class="metric-footer font-mono">
          <span>Enforcing Rules</span>
          <span class="badge badge-red font-mono">{{ store.bannedIps.length }} Banned IPs</span>
        </div>
      </div>
    </div>

    <!-- Quick Services Switchboard & Activity -->
    <div class="dashboard-columns">
      <!-- Web Server & PHP Overview -->
      <div class="bsd-card">
        <div class="bsd-card-header">
          <h2 class="bsd-card-title">
            <AppIcon name="Server" :size="18" class="text-red" />
            Multi Web Server & PHP Switchboard
          </h2>
          <router-link to="/webservers" class="btn btn-secondary btn-sm">
            Kelola Server
            <AppIcon name="ArrowRight" :size="14" />
          </router-link>
        </div>

        <div class="server-tiles-grid">
          <div
            v-for="ws in store.webservers"
            :key="ws.server_type"
            class="server-mini-tile"
            :class="{ active: ws.is_running }"
          >
            <div class="mini-tile-header">
              <span class="server-tag font-mono">{{ ws.server_type.toUpperCase() }}</span>
              <span class="badge" :class="ws.is_running ? 'badge-green' : 'badge-red'">
                {{ ws.is_running ? 'RUNNING' : 'STOPPED' }}
              </span>
            </div>
            <div class="mini-tile-body">
              <div class="server-name">{{ ws.display_name.split(' ')[0] }} {{ ws.version }}</div>
              <div class="server-port font-mono">Port: {{ ws.port_http }} / {{ ws.port_https }}</div>
            </div>
            <div class="mini-tile-action">
              <button
                class="btn btn-sm"
                :class="ws.is_running ? 'btn-danger' : 'btn-success'"
                @click="store.toggleWebserver(ws.server_type)"
              >
                {{ ws.is_running ? 'Matikan' : 'Aktifkan' }}
              </button>
            </div>
          </div>
        </div>

        <!-- PHP Active Versions Strip -->
        <div class="php-active-strip">
          <div class="strip-title font-mono">
            <AppIcon name="Code" :size="14" />
            PHP Multi-Version Socket Pools (8.2 - 8.6):
          </div>
          <div class="php-pill-list">
            <div
              v-for="php in store.phpVersions"
              :key="php.version"
              class="php-pill"
              :class="{ running: php.is_running }"
              @click="store.togglePhpVersion(php.version)"
              :title="php.is_running ? 'Klik untuk stop pool' : 'Klik untuk start pool'"
            >
              <span class="status-indicator"></span>
              <span class="font-mono">PHP {{ php.version }}</span>
              <span v-if="php.is_running" class="socket-tag font-mono">FPM:{{ php.port }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Quick Server Health & FreeBSD Security -->
      <div class="bsd-card">
        <div class="bsd-card-header">
          <h2 class="bsd-card-title">
            <AppIcon name="ShieldCheck" :size="18" class="text-green" />
            FreeBSD Security & Daemon Health
          </h2>
          <router-link to="/firewall" class="btn btn-secondary btn-sm">
            Firewall & Bans
            <AppIcon name="ArrowRight" :size="14" />
          </router-link>
        </div>

        <div class="health-list">
          <div class="health-item">
            <div class="health-info">
              <div class="health-title font-mono">pf (Packet Filter Kernel Engine)</div>
              <div class="health-desc">Perlindungan jaringan berbasis kernel FreeBSD</div>
            </div>
            <span class="badge badge-green">AKTIF</span>
          </div>

          <div class="health-item">
            <div class="health-info">
              <div class="health-title font-mono">fail2ban (Intrusion Detection)</div>
              <div class="health-desc">3 Jails aktif: bsd-sshd, bsd-panel, nginx-badbots</div>
            </div>
            <span class="badge badge-green">AKTIF ({{ store.bannedIps.length }} Bans)</span>
          </div>

          <div class="health-item">
            <div class="health-info">
              <div class="health-title font-mono">PostgreSQL 16 & MariaDB 10.11</div>
              <div class="health-desc">Multi database listening on localhost / unix domain sockets</div>
            </div>
            <span class="badge badge-green">ONLINE</span>
          </div>

          <div class="health-item">
            <div class="health-info">
              <div class="health-title font-mono">ZFS Pool Integrity (zroot)</div>
              <div class="health-desc">ZFS checksum scrubbing & copy-on-write integrity</div>
            </div>
            <span class="badge badge-green">ONLINE / 0 Errors</span>
          </div>
        </div>

        <div class="panel-quick-summary">
          <div class="sum-box">
            <div class="sum-number font-mono">{{ store.domains.length }}</div>
            <div class="sum-label">Domains</div>
          </div>
          <div class="sum-box">
            <div class="sum-number font-mono">{{ store.databases.length }}</div>
            <div class="sum-label">Databases</div>
          </div>
          <div class="sum-box">
            <div class="sum-number font-mono">{{ store.users.length }}</div>
            <div class="sum-label">Akun User/Reseller</div>
          </div>
          <div class="sum-box">
            <div class="sum-number font-mono">{{ store.firewallRules.length }}</div>
            <div class="sum-label">PF Rules</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useBsdStore } from '../stores/bsdStore'
import AppIcon from '../components/AppIcon.vue'

const store = useBsdStore()

function refreshStats() {
  // Simulate slight fluctuation in CPU / RAM
  store.stats.cpu_usage_percent = +(12 + Math.random() * 8).toFixed(1)
  store.stats.pf_states_count = Math.floor(130 + Math.random() * 30)
}
</script>

<style scoped>
.dashboard-page {
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

.text-bold {
  font-weight: 600;
  color: var(--text-primary);
}

.text-red { color: var(--accent-red); }
.text-blue { color: var(--accent-blue); }
.text-green { color: var(--accent-green); }
.text-amber { color: var(--accent-amber); }

.bg-red { background: var(--accent-red); }
.bg-blue { background: var(--accent-blue); }
.bg-green { background: var(--accent-green); }
.bg-amber { background: var(--accent-amber); }

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.live-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(16, 185, 129, 0.15);
  color: #34d399;
  border: 1px solid rgba(16, 185, 129, 0.3);
  font-size: 0.725rem;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 9999px;
  letter-spacing: 0.05em;
}

.live-dot {
  width: 7px;
  height: 7px;
  background: #34d399;
  border-radius: 50%;
  animation: pulse 1.8s infinite;
}

@keyframes pulse {
  0% { transform: scale(0.9); opacity: 0.7; }
  50% { transform: scale(1.3); opacity: 1; }
  100% { transform: scale(0.9); opacity: 0.7; }
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 16px;
}

.metric-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.metric-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.metric-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.875rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.metric-value {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
}

.metric-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.dashboard-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

@media (max-width: 1024px) {
  .dashboard-columns {
    grid-template-columns: 1fr;
  }
}

.server-tiles-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 20px;
}

@media (max-width: 600px) {
  .server-tiles-grid {
    grid-template-columns: 1fr;
  }
}

.server-mini-tile {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  transition: all 0.2s;
}

.server-mini-tile.active {
  border-color: rgba(16, 185, 129, 0.4);
}

.mini-tile-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.server-tag {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--text-secondary);
}

.server-name {
  font-weight: 600;
  font-size: 0.9rem;
}

.server-port {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.php-active-strip {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 14px;
}

.strip-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  color: var(--text-secondary);
  margin-bottom: 10px;
}

.php-pill-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.php-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  font-size: 0.775rem;
  cursor: pointer;
  transition: all 0.2s;
}

.php-pill:hover {
  border-color: var(--text-muted);
}

.php-pill.running {
  border-color: rgba(16, 185, 129, 0.4);
  background: rgba(16, 185, 129, 0.08);
}

.status-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #64748b;
}

.php-pill.running .status-indicator {
  background: #10b981;
}

.socket-tag {
  font-size: 0.7rem;
  color: var(--text-muted);
  background: var(--bg-primary);
  padding: 2px 4px;
  border-radius: 3px;
}

.health-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;
}

.health-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-color);
}

.health-title {
  font-size: 0.825rem;
  font-weight: 600;
}

.health-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.panel-quick-summary {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  text-align: center;
}

.sum-box {
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 10px 6px;
}

.sum-number {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--text-primary);
}

.sum-label {
  font-size: 0.7rem;
  color: var(--text-muted);
  text-transform: uppercase;
}
</style>
