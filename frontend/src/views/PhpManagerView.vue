<template>
  <div class="php-page">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">
          <AppIcon name="Code" :size="24" class="text-blue" />
          PHP Multi-Version (8.2 - 8.6) & Management
        </h1>
        <p class="page-subtitle">
          Kelola FreeBSD PHP-FPM pools terisolasi, php.ini tuning, dan ekstensi modul per versi
        </p>
      </div>

      <div class="header-actions">
        <span class="badge badge-purple font-mono">
          FreeBSD Ports: lang/php82 s/d lang/php86
        </span>
      </div>
    </div>

    <!-- PHP Versions Selector Tabs / Cards (8.2 - 8.6) -->
    <div class="php-cards-grid">
      <div
        v-for="php in store.phpVersions"
        :key="php.version"
        class="bsd-card php-card"
        :class="{ active: activeVersion === php.version, running: php.is_running }"
        @click="activeVersion = php.version"
      >
        <div class="php-card-top">
          <div class="ver-label font-mono">PHP {{ php.version }}</div>
          <span class="badge" :class="php.is_running ? 'badge-green' : 'badge-red'">
            {{ php.is_running ? 'ACTIVE POOL' : 'STOPPED' }}
          </span>
        </div>

        <div class="php-card-details font-mono">
          <div class="detail-line">
            <span class="label">Socket:</span>
            <span class="val text-truncate">{{ php.socket_path }}</span>
          </div>
          <div class="detail-line">
            <span class="label">Port FCGI:</span>
            <span class="val">{{ php.port }}</span>
          </div>
          <div class="detail-line">
            <span class="label">Workers:</span>
            <span class="val">{{ php.active_processes }} child processes</span>
          </div>
        </div>

        <div class="php-card-actions">
          <button
            class="btn btn-sm"
            :class="php.is_running ? 'btn-danger' : 'btn-success'"
            @click.stop="store.togglePhpVersion(php.version)"
          >
            {{ php.is_running ? 'Stop FPM' : 'Start FPM' }}
          </button>
          <span class="manage-text font-mono" v-if="activeVersion === php.version">
            Sedang Dipilih ▾
          </span>
        </div>
      </div>
    </div>

    <!-- Active Version Management Console -->
    <div class="management-console">
      <div class="console-tabs">
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'ini' }"
          @click="activeTab = 'ini'"
        >
          <AppIcon name="Sliders" :size="15" />
          Konfigurasi php.ini (PHP {{ activeVersion }})
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'extensions' }"
          @click="activeTab = 'extensions'"
        >
          <AppIcon name="Package" :size="15" />
          Modul & Ekstensi ({{ store.phpExtensions.filter(e => e.is_enabled).length }} Aktif)
        </button>
        <button
          class="tab-btn"
          :class="{ active: activeTab === 'pool' }"
          @click="activeTab = 'pool'"
        >
          <AppIcon name="Cpu" :size="15" />
          FPM Pool Worker Tuning
        </button>
      </div>

      <!-- Tab 1: php.ini Editor -->
      <div v-if="activeTab === 'ini'" class="bsd-card tab-content">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="FileText" :size="18" class="text-amber" />
            Directive php.ini Editor: PHP {{ activeVersion }}
          </h3>
          <span class="text-muted font-mono" style="font-size: 0.775rem;">
            Path: /usr/local/etc/php{{ activeVersion.replace('.', '') }}/php.ini
          </span>
        </div>

        <div class="ini-grid">
          <div class="form-group">
            <label class="form-label">memory_limit</label>
            <input v-model="store.phpIni.memory_limit" type="text" class="form-control font-mono" />
            <small class="text-muted">Batas maksimum RAM per skrip (cth: 256M, 512M)</small>
          </div>

          <div class="form-group">
            <label class="form-label">upload_max_filesize</label>
            <input v-model="store.phpIni.upload_max_filesize" type="text" class="form-control font-mono" />
            <small class="text-muted">Ukuran unggahan file tunggal maksimal (cth: 64M, 128M)</small>
          </div>

          <div class="form-group">
            <label class="form-label">post_max_size</label>
            <input v-model="store.phpIni.post_max_size" type="text" class="form-control font-mono" />
            <small class="text-muted">Ukuran maksimal payload POST (cth: 128M)</small>
          </div>

          <div class="form-group">
            <label class="form-label">max_execution_time (detik)</label>
            <input v-model="store.phpIni.max_execution_time" type="number" class="form-control font-mono" />
            <small class="text-muted">Batas waktu timeout skrip (cth: 300 detik)</small>
          </div>

          <div class="form-group">
            <label class="form-label">date.timezone</label>
            <input v-model="store.phpIni.date_timezone" type="text" class="form-control font-mono" />
            <small class="text-muted">Zona waktu server default (cth: Asia/Jakarta)</small>
          </div>

          <div class="form-group">
            <label class="form-label">opcache.memory_consumption (MB)</label>
            <input v-model="store.phpIni.opcache_memory_consumption" type="number" class="form-control font-mono" />
            <small class="text-muted">Alokasi memori OPcache (cth: 128 MB)</small>
          </div>
        </div>

        <div class="form-group" style="margin-top: 10px;">
          <label class="form-label">disable_functions (Keamanan FreeBSD)</label>
          <input
            v-model="store.phpIni.disable_functions"
            type="text"
            class="form-control font-mono"
            style="color: #f87171;"
          />
          <small class="text-muted">
            Fungsi berbahaya yang dinonaktifkan demi keamanan FreeBSD panel
          </small>
        </div>

        <div class="ini-actions">
          <button class="btn btn-primary" @click="saveIni">
            <AppIcon name="Save" :size="15" />
            Terapkan & Reload PHP-FPM {{ activeVersion }}
          </button>
        </div>
      </div>

      <!-- Tab 2: PHP Extensions Toggle -->
      <div v-if="activeTab === 'extensions'" class="bsd-card tab-content">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="Package" :size="18" class="text-blue" />
            Ekstensi & Modul Terpasang: PHP {{ activeVersion }}
          </h3>
          <span class="text-muted font-mono" style="font-size: 0.8rem;">
            Otomatis sinkron dengan FreeBSD pkg / ports
          </span>
        </div>

        <div class="ext-grid">
          <div
            v-for="ext in store.phpExtensions"
            :key="ext.name"
            class="ext-card"
            :class="{ enabled: ext.is_enabled }"
          >
            <div class="ext-info">
              <div class="ext-top">
                <span class="ext-name font-mono">{{ ext.name }}</span>
                <span class="badge font-mono" :class="ext.is_enabled ? 'badge-green' : 'badge-red'">
                  {{ ext.is_enabled ? 'ON' : 'OFF' }}
                </span>
              </div>
              <div class="ext-desc">{{ ext.description }}</div>
              <div class="ext-category font-mono text-muted">{{ ext.category }}</div>
            </div>

            <div class="toggle-switch" @click="store.toggleExtension(ext.name)">
              <input type="checkbox" :checked="ext.is_enabled" readonly />
              <span class="slider"></span>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab 3: FPM Pool Worker Tuning -->
      <div v-if="activeTab === 'pool'" class="bsd-card tab-content">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="Cpu" :size="18" class="text-green" />
            Tuning PHP-FPM Pool Workers (PHP {{ activeVersion }})
          </h3>
          <span class="badge badge-green font-mono">Process Manager: dynamic</span>
        </div>

        <div class="ini-grid">
          <div class="form-group">
            <label class="form-label">pm.max_children</label>
            <input type="number" class="form-control font-mono" value="25" />
            <small class="text-muted">Maksimum proses worker yang dapat berjalan bersamaan</small>
          </div>

          <div class="form-group">
            <label class="form-label">pm.start_servers</label>
            <input type="number" class="form-control font-mono" value="5" />
            <small class="text-muted">Jumlah worker saat pertama kali FPM dimulai</small>
          </div>

          <div class="form-group">
            <label class="form-label">pm.min_spare_servers</label>
            <input type="number" class="form-control font-mono" value="2" />
            <small class="text-muted">Batas minimal worker idle yang harus siap</small>
          </div>

          <div class="form-group">
            <label class="form-label">pm.max_spare_servers</label>
            <input type="number" class="form-control font-mono" value="10" />
            <small class="text-muted">Batas maksimal worker idle sebelum di-reap</small>
          </div>

          <div class="form-group">
            <label class="form-label">pm.max_requests</label>
            <input type="number" class="form-control font-mono" value="500" />
            <small class="text-muted">Mencegah memory leak dengan merestart worker setelah N request</small>
          </div>
        </div>

        <div class="ini-actions">
          <button class="btn btn-primary" @click="savePool">
            <AppIcon name="Save" :size="15" />
            Simpan Pool Konfigurasi
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useBsdStore } from '../stores/bsdStore'
import AppIcon from '../components/AppIcon.vue'

const store = useBsdStore()

const activeVersion = ref('8.3')
const activeTab = ref('ini')

function saveIni() {
  alert(`Pengaturan php.ini untuk PHP ${activeVersion.value} berhasil disimpan dan daemon FPM direload!`)
}

function savePool() {
  alert(`Konfigurasi pool worker PHP-FPM ${activeVersion.value} berhasil diperbarui!`)
}
</script>

<style scoped>
.php-page {
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

.php-cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
}

.php-card {
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: all 0.2s;
  padding: 16px;
}

.php-card.active {
  border-color: var(--accent-red);
  background: var(--bg-card-hover);
}

.php-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.ver-label {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--text-primary);
}

.php-card-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 0.725rem;
}

.detail-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.detail-line .label {
  color: var(--text-muted);
}

.detail-line .val {
  color: var(--text-primary);
  font-weight: 500;
}

.text-truncate {
  max-width: 110px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.php-card-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
}

.manage-text {
  font-size: 0.7rem;
  color: var(--accent-red);
  font-weight: 600;
}

.console-tabs {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-family: var(--font-family);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn.active {
  background: var(--accent-red-subtle);
  border-color: var(--accent-red);
  color: #fff;
}

.tab-content {
  padding: 24px;
}

.ini-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
}

.ini-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 24px;
}

.ext-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 14px;
}

.ext-card {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.ext-card.enabled {
  border-color: rgba(16, 185, 129, 0.3);
}

.ext-info {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.ext-top {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ext-name {
  font-weight: 600;
  font-size: 0.9rem;
}

.ext-desc {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.ext-category {
  font-size: 0.7rem;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  cursor: pointer;
  flex-shrink: 0;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  inset: 0;
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 22px;
  transition: 0.3s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 14px;
  width: 14px;
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
  transform: translateX(18px);
  background-color: #fff;
}
</style>
