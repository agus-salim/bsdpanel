<template>
  <div class="firewall-page">
    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">
          <AppIcon name="Shield" :size="24" class="text-red" />
          FreeBSD Packet Filter (PF) & Fail2ban Security
        </h1>
        <p class="page-subtitle">
          Firewall kernel FreeBSD terpadu dengan fail2ban intrusion prevention dan brute-force table blocking
        </p>
      </div>

      <div class="header-actions">
        <button class="btn btn-secondary" @click="showPfConfModal = true">
          <AppIcon name="FileCode" :size="15" />
          Lihat /etc/pf.conf
        </button>
        <button class="btn btn-danger" @click="showManualBanModal = true">
          <AppIcon name="Ban" :size="15" />
          Ban IP Manual
        </button>
        <button class="btn btn-primary" @click="showAddRuleModal = true">
          <AppIcon name="Plus" :size="15" />
          Tambah Rule PF
        </button>
      </div>
    </div>

    <!-- PF Status Banner -->
    <div class="bsd-card pf-status-banner">
      <div class="pf-status-info">
        <div class="shield-icon-wrap" :class="{ active: store.pfEnabled }">
          <AppIcon name="ShieldCheck" :size="28" :class="store.pfEnabled ? 'text-green' : 'text-muted'" />
        </div>
        <div>
          <div class="pf-title-row">
            <h2 class="pf-heading">FreeBSD Packet Filter (pfctl)</h2>
            <span class="badge" :class="store.pfEnabled ? 'badge-green' : 'badge-red'">
              {{ store.pfEnabled ? 'STATUS: ENFORCING (AKTIF)' : 'STATUS: DISABLED' }}
            </span>
          </div>
          <p class="pf-subtext">
            Filtering paket di level kernel FreeBSD. Table &lt;bruteforce&gt; &amp; &lt;whitelist&gt; tersinkronisasi otomatis.
          </p>
        </div>
      </div>

      <div class="pf-controls">
        <button
          class="btn"
          :class="store.pfEnabled ? 'btn-danger' : 'btn-success'"
          @click="store.togglePf"
        >
          {{ store.pfEnabled ? 'Nonaktifkan PF' : 'Aktifkan PF' }}
        </button>
      </div>
    </div>

    <!-- Fail2ban Jails Grid -->
    <div class="jails-grid">
      <div
        v-for="jail in store.fail2banJails"
        :key="jail.name"
        class="bsd-card jail-card"
      >
        <div class="jail-top">
          <div class="jail-title font-mono">
            <AppIcon name="Lock" :size="15" class="text-amber" />
            <span>Jail: {{ jail.name }}</span>
          </div>
          <span class="badge badge-green font-mono">ACTIVE</span>
        </div>

        <div class="jail-stats font-mono">
          <div class="stat-box">
            <span class="s-val">{{ jail.currently_banned }}</span>
            <span class="s-lbl">Sedang Diblokir</span>
          </div>
          <div class="stat-box">
            <span class="s-val">{{ jail.total_banned }}</span>
            <span class="s-lbl">Total Riwayat Ban</span>
          </div>
          <div class="stat-box">
            <span class="s-val">{{ jail.max_retry }}x</span>
            <span class="s-lbl">Max Retry</span>
          </div>
          <div class="stat-box">
            <span class="s-val">{{ jail.ban_time / 60 }}m</span>
            <span class="s-lbl">Ban Duration</span>
          </div>
        </div>

        <div class="jail-log font-mono text-muted">
          Log: {{ jail.log_path }}
        </div>
      </div>
    </div>

    <!-- Banned IPs Section -->
    <div class="bsd-card">
      <div class="bsd-card-header">
        <h2 class="bsd-card-title">
          <AppIcon name="Ban" :size="18" class="text-red" />
          Daftar IP Diblokir (Active Bans &bull; {{ store.bannedIps.length }})
        </h2>
        <span class="text-muted font-mono" style="font-size: 0.8rem;">
          Tersinkron dengan PF Table &lt;bruteforce&gt;
        </span>
      </div>

      <div class="table-responsive">
        <table class="bsd-table">
          <thead>
            <tr>
              <th>IP Address</th>
              <th>Jail Triggered</th>
              <th>Waktu Diblokir</th>
              <th>Sisa Waktu Ban</th>
              <th>Negara / Sumber</th>
              <th style="text-align: right;">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="b in store.bannedIps" :key="b.ip + b.jail">
              <td>
                <span class="font-mono text-bold text-red">{{ b.ip }}</span>
              </td>
              <td>
                <span class="badge badge-amber font-mono">{{ b.jail }}</span>
              </td>
              <td class="font-mono text-muted">
                {{ new Date(b.banned_at).toLocaleTimeString() }}
              </td>
              <td class="font-mono">
                <span class="badge badge-blue">
                  {{ Math.max(0, Math.round((b.expires_at - Date.now()) / 60000)) }} menit lagi
                </span>
              </td>
              <td>
                <span class="badge badge-purple font-mono">{{ b.country || 'UNKNOWN' }}</span>
              </td>
              <td style="text-align: right;">
                <button
                  class="btn btn-sm btn-success"
                  @click="store.unbanIp(b.ip, b.jail)"
                >
                  <AppIcon name="CheckCircle" :size="13" />
                  Unban IP
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Firewall Rules Table -->
    <div class="bsd-card">
      <div class="bsd-card-header">
        <h2 class="bsd-card-title">
          <AppIcon name="List" :size="18" class="text-blue" />
          Aturan Firewall Packet Filter (PF Rules &bull; {{ store.firewallRules.length }})
        </h2>
      </div>

      <div class="table-responsive">
        <table class="bsd-table">
          <thead>
            <tr>
              <th>Label / Deskripsi</th>
              <th>Port</th>
              <th>Protokol</th>
              <th>Tindakan (Action)</th>
              <th>Arah (Direction)</th>
              <th>Sumber (Source)</th>
              <th>Status</th>
              <th style="text-align: right;">Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in store.firewallRules" :key="r.id">
              <td>
                <span class="text-bold">{{ r.label }}</span>
              </td>
              <td>
                <span class="font-mono badge badge-blue">Port {{ r.port }}</span>
              </td>
              <td>
                <span class="font-mono uppercase">{{ r.protocol }}</span>
              </td>
              <td>
                <span
                  class="badge font-mono uppercase"
                  :class="r.action === 'pass' ? 'badge-green' : 'badge-red'"
                >
                  {{ r.action }}
                </span>
              </td>
              <td>
                <span class="font-mono uppercase">{{ r.direction }}</span>
              </td>
              <td class="font-mono text-muted">{{ r.source }}</td>
              <td>
                <span class="badge" :class="r.is_enabled ? 'badge-green' : 'badge-red'">
                  {{ r.is_enabled ? 'Aktif' : 'Nonaktif' }}
                </span>
              </td>
              <td style="text-align: right;">
                <button
                  class="btn btn-sm btn-danger"
                  @click="store.deleteFirewallRule(r.id)"
                >
                  <AppIcon name="Trash2" :size="13" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal Tambah Rule -->
    <div v-if="showAddRuleModal" class="modal-overlay" @click.self="showAddRuleModal = false">
      <div class="modal-content">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="PlusCircle" :size="18" class="text-red" />
            Tambah Aturan Firewall PF
          </h3>
          <button class="btn btn-sm btn-secondary" @click="showAddRuleModal = false">✕</button>
        </div>

        <form @submit.prevent="submitAddRule">
          <div class="form-group">
            <label class="form-label">Label Aturan</label>
            <input
              v-model="newRule.label"
              type="text"
              class="form-control"
              placeholder="contoh: Izinkan Port Custom Game/App"
              required
            />
          </div>

          <div class="form-grid-2">
            <div class="form-group">
              <label class="form-label">Port / Port Range</label>
              <input
                v-model="newRule.port"
                type="text"
                class="form-control font-mono"
                placeholder="contoh: 8080 atau 9000-9100"
                required
              />
            </div>

            <div class="form-group">
              <label class="form-label">Protokol</label>
              <select v-model="newRule.protocol" class="form-control font-mono">
                <option value="tcp">TCP</option>
                <option value="udp">UDP</option>
                <option value="both">TCP &amp; UDP</option>
              </select>
            </div>
          </div>

          <div class="form-grid-2">
            <div class="form-group">
              <label class="form-label">Tindakan (Action)</label>
              <select v-model="newRule.action" class="form-control font-mono">
                <option value="pass">PASS (Izinkan)</option>
                <option value="block">BLOCK (Tolak/Drop)</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">Arah (Direction)</label>
              <select v-model="newRule.direction" class="form-control font-mono">
                <option value="in">IN (Masuk)</option>
                <option value="out">OUT (Keluar)</option>
              </select>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">IP Sumber (Source IP / CIDR)</label>
            <input
              v-model="newRule.source"
              type="text"
              class="form-control font-mono"
              placeholder="any atau 192.168.1.0/24"
            />
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showAddRuleModal = false">
              Batal
            </button>
            <button type="submit" class="btn btn-primary">
              <AppIcon name="Check" :size="16" />
              Terapkan ke PF Kernel
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Modal Manual Ban IP -->
    <div v-if="showManualBanModal" class="modal-overlay" @click.self="showManualBanModal = false">
      <div class="modal-content">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="Ban" :size="18" class="text-red" />
            Blokir IP Address Manual
          </h3>
          <button class="btn btn-sm btn-secondary" @click="showManualBanModal = false">✕</button>
        </div>

        <form @submit.prevent="submitManualBan">
          <div class="form-group">
            <label class="form-label">Alamat IPv4 / IPv6 Target</label>
            <input
              v-model="manualBanIp"
              type="text"
              class="form-control font-mono"
              placeholder="contoh: 198.51.100.99"
              required
            />
          </div>

          <div class="form-group">
            <label class="form-label">Masukkan ke Jail</label>
            <select v-model="manualBanJail" class="form-control font-mono">
              <option v-for="j in store.fail2banJails" :key="j.name" :value="j.name">
                {{ j.name }}
              </option>
            </select>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showManualBanModal = false">
              Batal
            </button>
            <button type="submit" class="btn btn-danger">
              <AppIcon name="Ban" :size="16" />
              Blokir Sekarang
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Modal Lihat /etc/pf.conf -->
    <div v-if="showPfConfModal" class="modal-overlay" @click.self="showPfConfModal = false">
      <div class="modal-content modal-lg">
        <div class="bsd-card-header">
          <h3 class="bsd-card-title">
            <AppIcon name="FileText" :size="18" class="text-green" />
            Konfigurasi FreeBSD /etc/pf.conf
          </h3>
          <button class="btn btn-sm btn-secondary" @click="showPfConfModal = false">✕</button>
        </div>

        <pre class="pf-conf-box font-mono"><code>{{ generatedPfConf }}</code></pre>

        <div class="modal-actions">
          <button class="btn btn-primary" @click="showPfConfModal = false">
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

const showAddRuleModal = ref(false)
const showManualBanModal = ref(false)
const showPfConfModal = ref(false)

const manualBanIp = ref('')
const manualBanJail = ref('bsd-sshd')

const newRule = ref({
  label: '',
  port: '',
  protocol: 'tcp',
  action: 'pass',
  direction: 'in',
  source: 'any'
})

function submitAddRule() {
  if (!newRule.value.label || !newRule.value.port) return
  store.addFirewallRule(newRule.value)
  showAddRuleModal.value = false
  newRule.value.label = ''
  newRule.value.port = ''
}

function submitManualBan() {
  if (!manualBanIp.value) return
  store.banIp(manualBanIp.value, manualBanJail.value)
  showManualBanModal.value = false
  manualBanIp.value = ''
}

const generatedPfConf = computed(() => {
  let lines = [
    '# ============================================================== #',
    '# FreeBSD Packet Filter (PF) Config - Generated by BSD Panel     #',
    '# Location: /etc/pf.conf                                         #',
    '# ============================================================== #\n',
    'ext_if = "vtnet0"',
    'table <bruteforce> persist',
    'table <whitelist> persist { 127.0.0.1, ::1 }\n',
    'set block-policy drop',
    'set skip on lo0\n',
    'scrub in all fragment reassemble\n',
    'block in all',
    'pass out all keep state',
    'block quick from <bruteforce>',
    'pass quick from <whitelist>\n',
    '# Rules Defined in BSD Panel:'
  ]

  store.firewallRules.forEach(r => {
    if (!r.is_enabled) return
    const proto = r.protocol === 'both' ? '{ proto tcp, proto udp }' : `proto ${r.protocol}`
    lines.push(`${r.action} ${r.direction} on $ext_if ${proto} from ${r.source} to any port ${r.port} keep state # ${r.label}`)
  })

  return lines.join('\n')
})
</script>

<style scoped>
.firewall-page {
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

.pf-status-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  background: var(--bg-card);
  border-left: 4px solid var(--accent-red);
}

.pf-status-info {
  display: flex;
  align-items: center;
  gap: 16px;
}

.shield-icon-wrap {
  width: 50px;
  height: 50px;
  border-radius: var(--radius-md);
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.shield-icon-wrap.active {
  background: rgba(16, 185, 129, 0.15);
}

.pf-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.pf-heading {
  font-size: 1.15rem;
  font-weight: 700;
}

.pf-subtext {
  font-size: 0.825rem;
  color: var(--text-secondary);
  margin-top: 2px;
}

.jails-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
}

.jail-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.jail-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.jail-title {
  font-weight: 600;
  font-size: 0.95rem;
  display: flex;
  align-items: center;
  gap: 6px;
}

.jail-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  background: var(--bg-input);
  padding: 10px 8px;
  border-radius: var(--radius-sm);
  text-align: center;
}

.stat-box {
  display: flex;
  flex-direction: column;
}

.s-val {
  font-weight: 700;
  font-size: 1rem;
  color: var(--text-primary);
}

.s-lbl {
  font-size: 0.65rem;
  color: var(--text-muted);
}

.jail-log {
  font-size: 0.725rem;
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

.modal-lg {
  max-width: 800px;
}

.pf-conf-box {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 16px;
  max-height: 420px;
  overflow-y: auto;
  font-size: 0.825rem;
  color: #38bdf8;
  line-height: 1.5;
}
</style>
