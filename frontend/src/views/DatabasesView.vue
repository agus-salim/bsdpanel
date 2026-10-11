<template>
  <div class="databases-page">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">
          <AppIcon name="Database" :size="24" class="text-amber" />
          Multi Database Management (MariaDB & PostgreSQL)
        </h1>
        <p class="page-subtitle">
          Kelola database server, user privilege, dan koneksi pada sistem FreeBSD
        </p>
      </div>

      <div class="header-actions">
        <button class="btn btn-secondary" @click="showAddUserModal = true">
          <AppIcon name="UserPlus" :size="16" />
          Tambah DB User
        </button>
        <button class="btn btn-primary" @click="showAddDbModal = true">
          <AppIcon name="Plus" :size="16" />
          Buat Database Baru
        </button>
      </div>
    </div>

    <!-- Multi Database Server Status Cards -->
    <div class="db-servers-grid">
      <div
        v-for="srv in store.dbServers"
        :key="srv.engine"
        class="bsd-card db-srv-card"
        :class="srv.engine"
      >
        <div class="srv-top">
          <div class="srv-title-wrap">
            <AppIcon
              :name="srv.engine === 'mariadb' ? 'Database' : 'Layers'"
              :size="20"
              :class="srv.engine === 'mariadb' ? 'text-blue' : 'text-purple'"
            />
            <div>
              <h3 class="srv-name">{{ srv.display_name }}</h3>
              <span class="badge badge-green font-mono">
                Port {{ srv.port }} &bull; RUNNING
              </span>
            </div>
          </div>

          <div class="srv-engine-tag font-mono">
            {{ srv.engine.toUpperCase() }}
          </div>
        </div>

        <div class="srv-metrics font-mono">
          <div class="m-item">
            <span class="m-label">Versi Engine:</span>
            <span class="m-val">{{ srv.version }}</span>
          </div>
          <div class="m-item">
            <span class="m-label">RAM Usage:</span>
            <span class="m-val">{{ srv.memory_mb }} MB</span>
          </div>
          <div class="m-item">
            <span class="m-label">Jumlah DB:</span>
            <span class="m-val">{{ store.databases.filter(d => d.engine === srv.engine).length }} databases</span>
          </div>
          <div class="m-item">
            <span class="m-label">rc.d Service:</span>
            <span class="m-val">{{ srv.service_name }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Databases Table -->
    <div class="bsd-card">
      <div class="bsd-card-header">
        <h2 class="bsd-card-title">
          <AppIcon name="HardDrive" :size="18" class="text-amber" />
          Daftar Database Aktif ({{ store.databases.length }})
        </h2>
        <span class="text-muted font-mono" style="font-size: 0.8rem;">
          Total Terpakai: {{ store.databases.reduce((acc, d) => acc + d.size_mb, 0).toFixed(1) }} MB
        </span>
      </div>

      <div class="table-responsive">
        <table class="bsd-table">
          <thead>
            <tr>
              <th>Nama Database</th>
              <th>Engine</th>
              <th>Pemilik</th>
              <th>Ukuran</th>
              <th>Jumlah Tabel</th>
              <th>Collation</th>
              <th style="text-align: right;">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="db in store.databases" :key="db.id">
              <td>
                <div class="db-name-cell">
                  <AppIcon name="Database" :size="15" class="text-amber" />
                  <span class="font-mono text-bold">{{ db.name }}</span>
                </div>
              </td>
              <td>
                <span
                  class="badge font-mono"
                  :class="db.engine === 'mariadb' ? 'badge-blue' : 'badge-purple'"
                >
                  {{ db.engine.toUpperCase() }}
                </span>
              </td>
              <td class="font-mono">{{ db.owner_username }}</td>
              <td class="font-mono">{{ db.size_mb }} MB</td>
              <td class="font-mono">{{ db.tables_count }} tabel</td>
              <td class="font-mono text-muted" style="font-size: 0.775rem;">{{ db.collation }}</td>
              <td style="text-align: right;">
                <button
                  class="btn btn-sm btn-danger"
                  title="Hapus Database"
                  @click="store.deleteDatabase(db.id)"
                >
                  <AppIcon name="Trash2" :size="13" />
                  Drop DB
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Database Users Table -->
    <div class="bsd-card">
      <div class="bsd-card-header">
        <h2 class="bsd-card-title">
          <AppIcon name="Users" :size="18" class="text-green" />
          Daftar Pengguna Database ({{ store.dbUsers.length }})
        </h2>
      </div>

      <div class="table-responsive">
        <table class="bsd-table">
          <thead>
            <tr>
              <th>Username DB</th>
              <th>Engine</th>
              <th>Allowed Host</th>
              <th>Database Yang Diberi Hak Akses</th>
              <th style="text-align: right;">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in store.dbUsers" :key="u.id">
              <td>
                <div class="db-name-cell">
                  <AppIcon name="Key" :size="14" class="text-green" />
                  <span class="font-mono text-bold">{{ u.username }}</span>
                </div>
              </td>
              <td>
                <span
                  class="badge font-mono"
                  :class="u.engine === 'mariadb' ? 'badge-blue' : 'badge-purple'"
                >
                  {{ u.engine.toUpperCase() }}
                </span>
              </td>
              <td class="font-mono text-muted">{{ u.allowed_host }}</td>
              <td>
                <span
                  v-for="d in u.assigned_databases"
                  :key="d"
                  class="badge badge-amber font-mono"
                  style="margin-right: 4px;"
                >
                  {{ d }}
                </span>
              </td>
              <td style="text-align: right;">
                <button
                  class="btn btn-sm btn-danger"
                  @click="deleteDbUser(u.id)"
                >
                  <AppIcon name="Trash2" :size="13" />
                  Hapus User
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal Buat Database -->
    <div v-if="showAddDbModal" class="modal-overlay" @click.self="showAddDbModal = false">
      <div class="modal-content">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="PlusCircle" :size="18" class="text-amber" />
            Buat Database Baru
          </h3>
          <button class="btn btn-sm btn-secondary" @click="showAddDbModal = false">✕</button>
        </div>

        <form @submit.prevent="submitAddDb">
          <div class="form-group">
            <label class="form-label">Nama Database</label>
            <input
              v-model="newDb.name"
              type="text"
              class="form-control font-mono"
              placeholder="contoh: aplikasi_db"
              required
            />
          </div>

          <div class="form-group">
            <label class="form-label">Pilih Database Engine</label>
            <select v-model="newDb.engine" class="form-control font-mono">
              <option value="mariadb">MariaDB 10.11 (MySQL Compatible)</option>
              <option value="postgresql">PostgreSQL 16 (Advanced Object-Relational)</option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">Character Set / Collation</label>
            <input
              v-model="newDb.collation"
              type="text"
              class="form-control font-mono"
              :placeholder="newDb.engine === 'mariadb' ? 'utf8mb4_unicode_ci' : 'en_US.UTF-8'"
            />
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showAddDbModal = false">
              Batal
            </button>
            <button type="submit" class="btn btn-primary">
              <AppIcon name="Check" :size="16" />
              Buat Database
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Modal Tambah DB User -->
    <div v-if="showAddUserModal" class="modal-overlay" @click.self="showAddUserModal = false">
      <div class="modal-content">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="UserPlus" :size="18" class="text-green" />
            Tambah Pengguna Database
          </h3>
          <button class="btn btn-sm btn-secondary" @click="showAddUserModal = false">✕</button>
        </div>

        <form @submit.prevent="submitAddUser">
          <div class="form-group">
            <label class="form-label">Username Database</label>
            <input
              v-model="newUser.username"
              type="text"
              class="form-control font-mono"
              placeholder="contoh: usr_aplikasi"
              required
            />
          </div>

          <div class="form-group">
            <label class="form-label">Password</label>
            <input
              v-model="newUser.password"
              type="password"
              class="form-control font-mono"
              placeholder="Password yang kuat"
              required
            />
          </div>

          <div class="form-group">
            <label class="form-label">Database Engine</label>
            <select v-model="newUser.engine" class="form-control font-mono">
              <option value="mariadb">MariaDB</option>
              <option value="postgresql">PostgreSQL</option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">Allowed Host</label>
            <input
              v-model="newUser.allowed_host"
              type="text"
              class="form-control font-mono"
              placeholder="localhost atau 127.0.0.1 atau %"
            />
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showAddUserModal = false">
              Batal
            </button>
            <button type="submit" class="btn btn-primary">
              <AppIcon name="Check" :size="16" />
              Simpan User
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useBsdStore } from '../stores/bsdStore'
import AppIcon from '../components/AppIcon.vue'

const store = useBsdStore()

const showAddDbModal = ref(false)
const showAddUserModal = ref(false)

const newDb = ref({
  name: '',
  engine: 'mariadb',
  collation: ''
})

const newUser = ref({
  username: '',
  password: '',
  engine: 'mariadb',
  allowed_host: 'localhost'
})

function submitAddDb() {
  if (!newDb.value.name) return
  store.addDatabase(newDb.value)
  showAddDbModal.value = false
  newDb.value.name = ''
}

function submitAddUser() {
  if (!newUser.value.username) return
  store.dbUsers.push({
    id: 'dbu-' + Date.now(),
    username: newUser.value.username,
    engine: newUser.value.engine,
    owner_username: 'client_company',
    allowed_host: newUser.value.allowed_host || 'localhost',
    assigned_databases: [],
    created_at: Date.now()
  })
  showAddUserModal.value = false
  newUser.value.username = ''
  newUser.value.password = ''
}

function deleteDbUser(id) {
  store.dbUsers = store.dbUsers.filter(u => u.id !== id)
}
</script>

<style scoped>
.databases-page {
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

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.db-servers-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

@media (max-width: 800px) {
  .db-servers-grid {
    grid-template-columns: 1fr;
  }
}

.db-srv-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.srv-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
}

.srv-title-wrap {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.srv-name {
  font-size: 1.05rem;
  font-weight: 600;
  margin-bottom: 4px;
}

.srv-engine-tag {
  font-size: 0.75rem;
  font-weight: 700;
  background: var(--bg-tertiary);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
}

.srv-metrics {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  background: var(--bg-input);
  padding: 12px;
  border-radius: var(--radius-sm);
  font-size: 0.775rem;
}

.m-item {
  display: flex;
  flex-direction: column;
}

.m-label {
  color: var(--text-muted);
  font-size: 0.7rem;
}

.m-val {
  font-weight: 600;
  color: var(--text-primary);
}

.db-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 24px;
}
</style>
