<template>
  <div class="webservers-page">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">
          <AppIcon name="Globe" :size="24" class="text-red" />
          Multi Web Server & Domain Management
        </h1>
        <p class="page-subtitle">
          Kelola 4 webserver utama (NGINX, Apache HTTPD, OpenLiteSpeed, Caddy) & Virtual Host FreeBSD
        </p>
      </div>

      <button class="btn btn-primary" @click="showAddDomainModal = true">
        <AppIcon name="Plus" :size="16" />
        Tambah Domain Baru
      </button>
    </div>

    <!-- Web Server Switchboard Grid (4 Webservers) -->
    <div class="servers-grid">
      <div
        v-for="ws in store.webservers"
        :key="ws.server_type"
        class="bsd-card server-status-card"
        :class="{ 'is-running': ws.is_running }"
      >
        <div class="card-top">
          <div class="server-badge-wrap">
            <span class="server-type-pill font-mono">{{ ws.server_type.toUpperCase() }}</span>
            <span class="badge" :class="ws.is_running ? 'badge-green' : 'badge-red'">
              {{ ws.is_running ? 'AKTIF (RUNNING)' : 'NONAKTIF' }}
            </span>
          </div>

          <div class="toggle-switch" @click="store.toggleWebserver(ws.server_type)">
            <input type="checkbox" :checked="ws.is_running" readonly />
            <span class="slider"></span>
          </div>
        </div>

        <h3 class="server-title">{{ ws.display_name }}</h3>
        <p class="server-config-path font-mono text-muted">{{ ws.config_path }}</p>

        <div class="server-specs font-mono">
          <div class="spec-item">
            <span class="spec-label">Port HTTP:</span>
            <span class="spec-val">{{ ws.port_http }}</span>
          </div>
          <div class="spec-item">
            <span class="spec-label">Port HTTPS:</span>
            <span class="spec-val">{{ ws.port_https }}</span>
          </div>
          <div class="spec-item">
            <span class="spec-label">RAM Usage:</span>
            <span class="spec-val">{{ ws.memory_mb }} MB</span>
          </div>
          <div class="spec-item">
            <span class="spec-label">rc.d Service:</span>
            <span class="spec-val">{{ ws.service_name }}</span>
          </div>
        </div>

        <div class="server-card-actions">
          <button
            class="btn btn-sm btn-secondary"
            @click="reloadService(ws.service_name)"
            :disabled="!ws.is_running"
          >
            <AppIcon name="RotateCw" :size="13" />
            Reload rc.d
          </button>
          <button
            class="btn btn-sm"
            :class="ws.is_running ? 'btn-danger' : 'btn-success'"
            @click="store.toggleWebserver(ws.server_type)"
          >
            {{ ws.is_running ? 'Stop Daemon' : 'Start Daemon' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Domains / Virtual Hosts Table -->
    <div class="bsd-card">
      <div class="bsd-card-header">
        <h2 class="bsd-card-title">
          <AppIcon name="Server" :size="18" class="text-blue" />
          Daftar Domain & Virtual Host Aktif ({{ store.domains.length }})
        </h2>
        <span class="text-muted font-mono" style="font-size: 0.8rem;">
          Path Standar: /usr/local/www/[domain]/public
        </span>
      </div>

      <div class="table-responsive">
        <table class="bsd-table">
          <thead>
            <tr>
              <th>Domain Name</th>
              <th>Web Server</th>
              <th>PHP FPM Pool</th>
              <th>Document Root</th>
              <th>SSL Security</th>
              <th>Status</th>
              <th style="text-align: right;">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="d in store.domains" :key="d.id">
              <td>
                <div class="domain-name-cell">
                  <AppIcon name="Globe" :size="15" class="text-blue" />
                  <span class="font-mono text-bold">{{ d.domain_name }}</span>
                </div>
              </td>
              <td>
                <span class="badge badge-purple font-mono uppercase">{{ d.webserver_type }}</span>
              </td>
              <td>
                <span class="badge badge-blue font-mono">PHP {{ d.php_version }}</span>
              </td>
              <td class="font-mono text-muted" style="font-size: 0.8rem;">
                {{ d.document_root }}
              </td>
              <td>
                <span v-if="d.ssl_enabled" class="badge badge-green font-mono">
                  <AppIcon name="Lock" :size="11" /> Let's Encrypt
                </span>
                <span v-else class="badge badge-amber font-mono">HTTP Only</span>
              </td>
              <td>
                <span class="badge" :class="d.is_active ? 'badge-green' : 'badge-red'">
                  {{ d.is_active ? 'Aktif' : 'Suspended' }}
                </span>
              </td>
              <td style="text-align: right;">
                <div class="action-buttons-cell">
                  <button
                    class="btn btn-sm btn-secondary"
                    title="Lihat Konfigurasi VHost"
                    @click="previewVhost(d)"
                  >
                    <AppIcon name="Code" :size="13" />
                    VHost Conf
                  </button>
                  <button
                    class="btn btn-sm btn-danger"
                    title="Hapus Domain"
                    @click="store.deleteDomain(d.id)"
                  >
                    <AppIcon name="Trash2" :size="13" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal Tambah Domain -->
    <div v-if="showAddDomainModal" class="modal-overlay" @click.self="showAddDomainModal = false">
      <div class="modal-content">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="PlusCircle" :size="18" class="text-red" />
            Tambah Virtual Host / Domain Baru
          </h3>
          <button class="btn btn-sm btn-secondary" @click="showAddDomainModal = false">✕</button>
        </div>

        <form @submit.prevent="submitAddDomain">
          <div class="form-group">
            <label class="form-label">Nama Domain (FQDN)</label>
            <input
              v-model="newDomain.domain_name"
              type="text"
              class="form-control font-mono"
              placeholder="contoh: aplikasi.sch.id atau web.domain.com"
              required
            />
          </div>

          <div class="form-grid-2">
            <div class="form-group">
              <label class="form-label">Pilihan Web Server Engine</label>
              <select v-model="newDomain.webserver_type" class="form-control font-mono">
                <option value="nginx">NGINX (High Performance)</option>
                <option value="apache">Apache HTTPD (httpd24)</option>
                <option value="litespeed">OpenLiteSpeed (LSWS)</option>
                <option value="caddy">Caddy (Automatic HTTPS)</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">Versi PHP FastCGI (FPM)</label>
              <select v-model="newDomain.php_version" class="form-control font-mono">
                <option v-for="php in store.phpVersions" :key="php.version" :value="php.version">
                  PHP {{ php.version }} {{ php.is_running ? '(Active Pool)' : '' }}
                </option>
              </select>
            </div>
          </div>

          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="newDomain.ssl_auto_cert" />
              <span>Generate Free SSL (Let's Encrypt / ACME FreeBSD)</span>
            </label>
          </div>

          <div class="form-group checkbox-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="newDomain.force_https" />
              <span>Paksa Redirect HTTPS (Port 80 -> 443)</span>
            </label>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showAddDomainModal = false">
              Batal
            </button>
            <button type="submit" class="btn btn-primary">
              <AppIcon name="Check" :size="16" />
              Simpan & Generate VHost FreeBSD
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Modal Preview VHost Configuration -->
    <div v-if="showVhostModal" class="modal-overlay" @click.self="showVhostModal = false">
      <div class="modal-content modal-lg">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="FileCode" :size="18" class="text-blue" />
            VHost Config Preview: <span class="font-mono text-red">{{ activeVhostDomain?.domain_name }}</span>
          </h3>
          <button class="btn btn-sm btn-secondary" @click="showVhostModal = false">✕</button>
        </div>

        <div class="vhost-meta">
          <span class="badge badge-purple font-mono">{{ activeVhostDomain?.webserver_type.toUpperCase() }}</span>
          <span class="badge badge-blue font-mono">PHP-FPM {{ activeVhostDomain?.php_version }}</span>
          <span class="text-muted font-mono" style="font-size: 0.775rem;">
            FreeBSD Location: /usr/local/etc/{{ activeVhostDomain?.webserver_type }}/conf.d/{{ activeVhostDomain?.domain_name }}.conf
          </span>
        </div>

        <pre class="vhost-code-block font-mono"><code>{{ generatedVhostCode }}</code></pre>

        <div class="modal-actions">
          <button class="btn btn-secondary" @click="copyCode">
            <AppIcon name="Copy" :size="14" />
            Salin Kode
          </button>
          <button class="btn btn-primary" @click="showVhostModal = false">
            Tutup
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useBsdStore } from '../stores/bsdStore'
import AppIcon from '../components/AppIcon.vue'

const store = useBsdStore()

const showAddDomainModal = ref(false)
const showVhostModal = ref(false)
const activeVhostDomain = ref(null)

const newDomain = ref({
  domain_name: '',
  webserver_type: 'nginx',
  php_version: '8.3',
  ssl_auto_cert: true,
  force_https: true
})

function submitAddDomain() {
  if (!newDomain.value.domain_name) return
  store.addDomain(newDomain.value)
  showAddDomainModal.value = false
  newDomain.value.domain_name = ''
}

function reloadService(serviceName) {
  alert(`Service ${serviceName} berhasil direload melalui FreeBSD /usr/sbin/service reload!`)
}

function previewVhost(domain) {
  activeVhostDomain.value = domain
  showVhostModal.value = true
}

const generatedVhostCode = computed(() => {
  if (!activeVhostDomain.value) return ''
  const d = activeVhostDomain.value
  const cleanVer = d.php_version.replace('.', '')
  const sock = `/var/run/php-fpm-${cleanVer}.sock`

  if (d.webserver_type === 'nginx') {
    return `# ==============================================================
# FreeBSD NGINX Virtual Host: ${d.domain_name}
# Managed automatically by BSD Panel
# ==============================================================
server {
    listen 80;
    listen [::]:80;
    server_name ${d.domain_name} www.${d.domain_name};

    root ${d.document_root};
    index index.php index.html index.htm;

    access_log /var/log/nginx/${d.domain_name}_access.log;
    error_log /var/log/nginx/${d.domain_name}_error.log;

    ${d.force_https ? 'return 301 https://$host$request_uri;' : '# HTTP allowed'}

    location / {
        try_files $uri $uri/ /index.php?$args;
    }

    location ~ \\.php$ {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\\.php)(/.+)$;
        fastcgi_pass unix:${sock};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
        fastcgi_buffers 16 16k;
        fastcgi_buffer_size 32k;
    }

    location ~ /\\.ht {
        deny all;
    }
}`
  } else if (d.webserver_type === 'caddy') {
    return `# FreeBSD Caddyfile VHost: ${d.domain_name}
${d.domain_name}, www.${d.domain_name} {
    root * ${d.document_root}
    php_fastcgi unix/${sock}
    file_server
    encode gzip zstd
    log {
        output file /var/log/caddy/${d.domain_name}.log
    }
}`
  } else if (d.webserver_type === 'apache') {
    return `# FreeBSD Apache 2.4 VirtualHost: ${d.domain_name}
<VirtualHost *:80>
    ServerName ${d.domain_name}
    ServerAlias www.${d.domain_name}
    DocumentRoot "${d.document_root}"

    <Directory "${d.document_root}">
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>

    <FilesMatch \\.php$>
        SetHandler "proxy:unix:${sock}|fcgi://localhost/"
    </FilesMatch>

    ErrorLog "/var/log/httpd/${d.domain_name}_error.log"
    CustomLog "/var/log/httpd/${d.domain_name}_access.log" combined
</VirtualHost>`
  } else {
    return `# FreeBSD OpenLiteSpeed VHost: ${d.domain_name}
docRoot                   ${d.document_root}
enableGzip                1

errorlog /usr/local/lsws/logs/${d.domain_name}.error_log {
  useServer               0
  logLevel                DEBUG
}

scripthandler {
  add                     lsapi:${sock} php
}`
  }
})

function copyCode() {
  navigator.clipboard.writeText(generatedVhostCode.value)
  alert('Konfigurasi VHost berhasil disalin ke clipboard!')
}
</script>

<style scoped>
.webservers-page {
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

.servers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
}

.server-status-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.server-status-card.is-running {
  border-color: rgba(16, 185, 129, 0.4);
}

.card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.server-badge-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
}

.server-type-pill {
  font-weight: 700;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.server-title {
  font-size: 1.05rem;
  font-weight: 600;
}

.server-config-path {
  font-size: 0.725rem;
  background: var(--bg-tertiary);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  word-break: break-all;
}

.server-specs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  font-size: 0.775rem;
  background: var(--bg-input);
  padding: 10px;
  border-radius: var(--radius-sm);
}

.spec-item {
  display: flex;
  flex-direction: column;
}

.spec-label {
  color: var(--text-muted);
  font-size: 0.7rem;
}

.spec-val {
  font-weight: 600;
  color: var(--text-primary);
}

.server-card-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
  gap: 8px;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 44px;
  height: 24px;
  cursor: pointer;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  inset: 0;
  background-color: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: 24px;
  transition: 0.3s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 16px;
  width: 16px;
  left: 3px;
  bottom: 3px;
  background-color: var(--text-muted);
  border-radius: 50%;
  transition: 0.3s;
}

.toggle-switch input:checked + .slider {
  background-color: var(--accent-green);
  border-color: var(--accent-green);
}

.toggle-switch input:checked + .slider:before {
  transform: translateX(20px);
  background-color: #fff;
}

.domain-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-buttons-cell {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
}

.form-grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.checkbox-group {
  margin-top: 8px;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.85rem;
  cursor: pointer;
  color: var(--text-primary);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 24px;
}

.modal-lg {
  max-width: 800px;
}

.vhost-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.vhost-code-block {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 16px;
  max-height: 400px;
  overflow-y: auto;
  font-size: 0.8rem;
  color: #38bdf8;
  line-height: 1.4;
}
</style>
