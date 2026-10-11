<template>
  <div class="users-page">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">
          <AppIcon name="Users" :size="24" class="text-purple" />
          Manajemen User & Reseller Multi-Tier
        </h1>
        <p class="page-subtitle">
          Hierarki hak akses berjenjang (SuperAdmin, Reseller, User) dan isolasi kuota FreeBSD
        </p>
      </div>

      <button class="btn btn-primary" @click="showAddModal = true">
        <AppIcon name="UserPlus" :size="16" />
        Tambah Akun Baru
      </button>
    </div>

    <!-- Tier Overview Cards -->
    <div class="tiers-grid">
      <div class="bsd-card tier-card">
        <div class="tier-top">
          <AppIcon name="ShieldAlert" :size="22" class="text-red" />
          <span class="badge badge-red font-mono">ROOT / SUPERADMIN</span>
        </div>
        <div class="tier-desc">
          Akses penuh sistem FreeBSD, service kernel, pf firewall, semua reseller & user.
        </div>
        <div class="tier-count font-mono">
          {{ store.users.filter(u => u.role === 'admin').length }} Akun
        </div>
      </div>

      <div class="bsd-card tier-card">
        <div class="tier-top">
          <AppIcon name="Briefcase" :size="22" class="text-blue" />
          <span class="badge badge-blue font-mono">RESELLER HOSTING</span>
        </div>
        <div class="tier-desc">
          Dapat membuat sub-user, membagi kuota disk & bandwidth dalam alokasi paketnya.
        </div>
        <div class="tier-count font-mono">
          {{ store.users.filter(u => u.role === 'reseller').length }} Akun
        </div>
      </div>

      <div class="bsd-card tier-card">
        <div class="tier-top">
          <AppIcon name="UserCheck" :size="22" class="text-green" />
          <span class="badge badge-green font-mono">END USER / CLIENT</span>
        </div>
        <div class="tier-desc">
          Mengelola domain web, database, dan file website terisolasi dalam direktori home.
        </div>
        <div class="tier-count font-mono">
          {{ store.users.filter(u => u.role === 'user').length }} Akun
        </div>
      </div>
    </div>

    <!-- Users & Resellers Table -->
    <div class="bsd-card">
      <div class="bsd-card-header">
        <h2 class="bsd-card-title">
          <AppIcon name="List" :size="18" class="text-purple" />
          Daftar Pengguna Panel ({{ store.users.length }})
        </h2>
      </div>

      <div class="table-responsive">
        <table class="bsd-table">
          <thead>
            <tr>
              <th>Username</th>
              <th>Peran (Role)</th>
              <th>Email Kontak</th>
              <th>Penggunaan Kuota Disk</th>
              <th>Bandwidth Bulanan</th>
              <th>Limit Domain & DB</th>
              <th>Status</th>
              <th style="text-align: right;">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in store.users" :key="u.id">
              <td>
                <div class="user-cell">
                  <div class="avatar font-mono">{{ u.username.charAt(0).toUpperCase() }}</div>
                  <div>
                    <span class="font-mono text-bold">{{ u.username }}</span>
                    <div class="text-muted" style="font-size: 0.725rem;">
                      Terdaftar: {{ new Date(u.created_at).toLocaleDateString() }}
                    </div>
                  </div>
                </div>
              </td>
              <td>
                <span
                  class="badge font-mono uppercase"
                  :class="{
                    'badge-red': u.role === 'admin',
                    'badge-blue': u.role === 'reseller',
                    'badge-green': u.role === 'user'
                  }"
                >
                  {{ u.role }}
                </span>
              </td>
              <td class="font-mono text-muted">{{ u.email }}</td>
              <td>
                <div class="quota-progress-wrap">
                  <div class="quota-text font-mono">
                    <span>{{ (u.disk_used_mb / 1024).toFixed(1) }} GB</span>
                    <span>/ {{ u.disk_limit_mb === 0 ? 'Unlimited' : (u.disk_limit_mb / 1024).toFixed(0) + ' GB' }}</span>
                  </div>
                  <div class="progress-bar-bg" v-if="u.disk_limit_mb > 0">
                    <div
                      class="progress-bar-fill bg-blue"
                      :style="{ width: `${Math.min(100, (u.disk_used_mb / u.disk_limit_mb) * 100)}%` }"
                    ></div>
                  </div>
                </div>
              </td>
              <td>
                <div class="quota-progress-wrap">
                  <div class="quota-text font-mono">
                    <span>{{ (u.bandwidth_used_mb / 1024).toFixed(1) }} GB</span>
                    <span>/ {{ u.bandwidth_limit_mb === 0 ? 'Unlimited' : (u.bandwidth_limit_mb / 1024).toFixed(0) + ' GB' }}</span>
                  </div>
                </div>
              </td>
              <td>
                <div class="font-mono" style="font-size: 0.8rem;">
                  <div>Domains: {{ u.domains_limit === 0 ? 'Unlimited' : u.domains_limit }}</div>
                  <div>Databases: {{ u.databases_limit === 0 ? 'Unlimited' : u.databases_limit }}</div>
                </div>
              </td>
              <td>
                <span class="badge" :class="u.status === 'active' ? 'badge-green' : 'badge-red'">
                  {{ u.status === 'active' ? 'Aktif' : 'Suspended' }}
                </span>
              </td>
              <td style="text-align: right;">
                <div class="user-actions-cell" v-if="u.role !== 'admin'">
                  <button
                    class="btn btn-sm"
                    :class="u.status === 'active' ? 'btn-danger' : 'btn-success'"
                    :title="u.status === 'active' ? 'Suspend Akun' : 'Aktifkan Akun'"
                    @click="store.toggleUserStatus(u.id)"
                  >
                    {{ u.status === 'active' ? 'Suspend' : 'Aktifkan' }}
                  </button>
                  <button
                    class="btn btn-sm btn-danger"
                    title="Hapus Akun"
                    @click="store.deleteUser(u.id)"
                  >
                    <AppIcon name="Trash2" :size="13" />
                  </button>
                </div>
                <span v-else class="text-muted font-mono" style="font-size: 0.75rem;">Root Protected</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal Tambah User / Reseller -->
    <div v-if="showAddModal" class="modal-overlay" @click.self="showAddModal = false">
      <div class="modal-content">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="UserPlus" :size="18" class="text-purple" />
            Tambah Akun User atau Reseller
          </h3>
          <button class="btn btn-sm btn-secondary" @click="showAddModal = false">✕</button>
        </div>

        <form @submit.prevent="submitAddUser">
          <div class="form-grid-2">
            <div class="form-group">
              <label class="form-label">Username FreeBSD</label>
              <input
                v-model="newUser.username"
                type="text"
                class="form-control font-mono"
                placeholder="contoh: client_alpha"
                required
              />
            </div>

            <div class="form-group">
              <label class="form-label">Tingkatan Akun (Role)</label>
              <select v-model="newUser.role" class="form-control font-mono">
                <option value="user">User Hosting Biasa</option>
                <option value="reseller">Reseller Hosting</option>
              </select>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">Alamat Email Kontak</label>
            <input
              v-model="newUser.email"
              type="email"
              class="form-control"
              placeholder="admin@klien.com"
              required
            />
          </div>

          <div class="form-grid-2">
            <div class="form-group">
              <label class="form-label">Alokasi Disk Quota (MB)</label>
              <input
                v-model="newUser.disk_limit_mb"
                type="number"
                class="form-control font-mono"
                placeholder="contoh: 10240 (10 GB)"
                required
              />
            </div>

            <div class="form-group">
              <label class="form-label">Bandwidth Bulanan (MB)</label>
              <input
                v-model="newUser.bandwidth_limit_mb"
                type="number"
                class="form-control font-mono"
                placeholder="contoh: 102400 (100 GB)"
                required
              />
            </div>
          </div>

          <div class="form-grid-2">
            <div class="form-group">
              <label class="form-label">Maksimal Domain</label>
              <input
                v-model="newUser.domains_limit"
                type="number"
                class="form-control font-mono"
                placeholder="5"
                required
              />
            </div>

            <div class="form-group">
              <label class="form-label">Maksimal Database</label>
              <input
                v-model="newUser.databases_limit"
                type="number"
                class="form-control font-mono"
                placeholder="10"
                required
              />
            </div>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showAddModal = false">
              Batal
            </button>
            <button type="submit" class="btn btn-primary">
              <AppIcon name="Check" :size="16" />
              Buat Akun FreeBSD
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

const showAddModal = ref(false)

const newUser = ref({
  username: '',
  role: 'user',
  email: '',
  disk_limit_mb: 10240,
  bandwidth_limit_mb: 102400,
  domains_limit: 5,
  databases_limit: 10
})

function submitAddUser() {
  if (!newUser.value.username || !newUser.value.email) return
  store.addUser(newUser.value)
  showAddModal.value = false
  newUser.value.username = ''
  newUser.value.email = ''
}
</script>

<style scoped>
.users-page {
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

.tiers-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 16px;
}

.tier-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tier-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tier-desc {
  font-size: 0.8rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

.tier-count {
  font-size: 1.25rem;
  font-weight: 700;
  margin-top: auto;
  color: var(--text-primary);
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  color: var(--accent-red);
  font-size: 0.85rem;
}

.quota-progress-wrap {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 120px;
}

.quota-text {
  font-size: 0.75rem;
  display: flex;
  justify-content: space-between;
}

.user-actions-cell {
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

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 24px;
}
</style>
