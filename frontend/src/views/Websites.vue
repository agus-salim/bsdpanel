<template>
  <div class="space-y-6 max-w-7xl mx-auto">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-2xl font-bold tracking-tight text-white">Manajemen Website & Reverse Proxy</h2>
        <p class="text-sm text-slate-400 mt-1">Kelola virtual host PHP, konfigurasi Nginx Reverse Proxy secara GUI, dan sertifikat SSL Let's Encrypt.</p>
      </div>
      <button
        @click="showCreateModal = true"
        class="px-4 py-2 bg-rose-600 hover:bg-rose-500 text-white text-sm font-semibold rounded-lg shadow-lg shadow-rose-900/30 transition flex items-center gap-2"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>
        Tambah Domain Baru
      </button>
    </div>

    <!-- Table of Websites -->
    <div class="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-sm">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead class="bg-slate-950/70 text-slate-400 border-b border-slate-800 text-xs uppercase tracking-wider font-semibold">
            <tr>
              <th class="px-6 py-4">Domain & Mode</th>
              <th class="px-6 py-4">Tipe Backend</th>
              <th class="px-6 py-4">SSL (Let's Encrypt)</th>
              <th class="px-6 py-4">Status</th>
              <th class="px-6 py-4 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-800/60 text-slate-300">
            <tr v-for="site in websites" :key="site.id" class="hover:bg-slate-800/40 transition">
              <td class="px-6 py-4">
                <div class="font-semibold text-white text-base flex items-center gap-2">
                  {{ site.domain }}
                  <span
                    v-if="site.is_reverse_proxy"
                    class="px-2 py-0.5 rounded text-[11px] font-mono bg-purple-500/10 text-purple-400 border border-purple-500/20"
                  >
                    Reverse Proxy
                  </span>
                  <span
                    v-else
                    class="px-2 py-0.5 rounded text-[11px] font-mono bg-blue-500/10 text-blue-400 border border-blue-500/20"
                  >
                    PHP FPM
                  </span>
                </div>
                <div class="text-xs text-slate-400 font-mono mt-0.5">
                  {{ site.is_reverse_proxy ? 'Target: ' + (site.proxy_target || 'http://127.0.0.1:3000') : site.document_root }}
                </div>
              </td>
              <td class="px-6 py-4">
                <div v-if="site.is_reverse_proxy" class="space-y-1">
                  <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-mono bg-slate-800 text-purple-300">
                    Upstream: {{ site.proxy_target || 'http://127.0.0.1:3000' }}
                  </span>
                  <div class="text-[11px] text-slate-500">
                    WebSocket: {{ site.proxy_websocket ? 'Aktif' : 'Non-aktif' }}
                  </div>
                </div>
                <div v-else>
                  <span class="px-2.5 py-1 bg-slate-800 border border-slate-700 rounded text-xs font-mono font-medium text-rose-300">
                    PHP {{ site.php_version }}
                  </span>
                </div>
              </td>
              <td class="px-6 py-4">
                <div class="flex items-center gap-2">
                  <span
                    class="inline-flex items-center gap-1.5 text-xs font-medium"
                    :class="site.ssl_enabled ? 'text-emerald-400' : 'text-slate-500'"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/></svg>
                    {{ site.ssl_enabled ? 'Aktif (HTTPS)' : 'Non-Aktif' }}
                  </span>
                  <button
                    v-if="!site.ssl_enabled"
                    @click="openSSLModal(site)"
                    class="px-2 py-0.5 text-[11px] font-medium rounded bg-emerald-600/10 hover:bg-emerald-600/20 text-emerald-400 border border-emerald-500/20 transition"
                  >
                    Pasang SSL
                  </button>
                </div>
              </td>
              <td class="px-6 py-4">
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/10 text-emerald-400">
                  {{ site.status }}
                </span>
              </td>
              <td class="px-6 py-4 text-right space-x-2">
                <button
                  @click="openProxyModal(site)"
                  class="px-2.5 py-1 text-xs font-medium rounded bg-slate-800 hover:bg-slate-700 text-slate-200 transition"
                >
                  Nginx Proxy GUI
                </button>
                <button
                  @click="openSSLModal(site)"
                  class="px-2.5 py-1 text-xs font-medium rounded bg-emerald-600/10 hover:bg-emerald-600/20 text-emerald-400 border border-emerald-500/20 transition"
                >
                  SSL Manager
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal Nginx Reverse Proxy GUI -->
    <div v-if="showProxyModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm">
      <div class="bg-slate-900 border border-slate-800 rounded-xl w-full max-w-lg p-6 shadow-2xl">
        <div class="flex items-center justify-between mb-4">
          <div>
            <h3 class="text-lg font-bold text-white">Nginx Reverse Proxy GUI</h3>
            <p class="text-xs text-slate-400 font-mono">{{ activeSite?.domain }}</p>
          </div>
          <button @click="showProxyModal = false" class="text-slate-500 hover:text-white">&times;</button>
        </div>

        <form @submit.prevent="saveProxySettings" class="space-y-4">
          <div class="flex items-center justify-between p-3 rounded-lg bg-slate-950 border border-slate-800">
            <div>
              <p class="text-sm font-semibold text-white">Aktifkan Reverse Proxy</p>
              <p class="text-xs text-slate-400">Meneruskan traffic ke aplikasi upstream (Node.js, Go, Python, dll)</p>
            </div>
            <input
              type="checkbox"
              v-model="proxyForm.is_reverse_proxy"
              class="w-5 h-5 rounded bg-slate-900 border-slate-700 text-rose-600 focus:ring-rose-500"
            />
          </div>

          <div v-if="proxyForm.is_reverse_proxy" class="space-y-3">
            <div>
              <label class="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">Target Upstream URL</label>
              <input
                v-model="proxyForm.proxy_target"
                type="text"
                placeholder="http://127.0.0.1:3000"
                required
                class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3.5 py-2.5 text-sm text-white focus:outline-none focus:border-rose-500 font-mono"
              />
              <span class="text-[11px] text-slate-500">Bisa berupa IP internal, localhost port, atau socket domain unix.</span>
            </div>

            <div class="flex items-center justify-between p-3 rounded-lg bg-slate-950 border border-slate-800">
              <div>
                <p class="text-sm font-semibold text-white">Support WebSocket Upgrade</p>
                <p class="text-xs text-slate-400">Menambahkan header Upgrade & Connection $http_upgrade</p>
              </div>
              <input
                type="checkbox"
                v-model="proxyForm.proxy_websocket"
                class="w-4 h-4 rounded bg-slate-900 border-slate-700 text-rose-600 focus:ring-rose-500"
              />
            </div>

            <div class="flex items-center justify-between p-3 rounded-lg bg-slate-950 border border-slate-800">
              <div>
                <p class="text-sm font-semibold text-white">Proxy Buffering</p>
                <p class="text-xs text-slate-400">Non-aktifkan jika menggunakan streaming event (SSE / AI chat)</p>
              </div>
              <input
                type="checkbox"
                v-model="proxyForm.proxy_buffering"
                class="w-4 h-4 rounded bg-slate-900 border-slate-700 text-rose-600 focus:ring-rose-500"
              />
            </div>

            <div>
              <label class="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">Client Max Body Size (MB)</label>
              <input
                v-model.number="proxyForm.max_body_size_mb"
                type="number"
                min="1"
                max="1024"
                class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3.5 py-2 text-sm text-white focus:outline-none focus:border-rose-500"
              />
            </div>
          </div>

          <div v-if="nginxTestOutput" class="p-3 rounded-lg bg-slate-950 border border-emerald-500/30 text-xs font-mono text-emerald-300 whitespace-pre-wrap">
            {{ nginxTestOutput }}
          </div>

          <div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-800">
            <button
              type="button"
              @click="showProxyModal = false"
              class="px-4 py-2 text-sm font-medium text-slate-400 hover:text-slate-200"
            >
              Tutup
            </button>
            <button
              type="submit"
              :disabled="savingProxy"
              class="px-5 py-2 bg-rose-600 hover:bg-rose-500 disabled:opacity-50 text-white text-sm font-semibold rounded-lg shadow-md transition flex items-center gap-2"
            >
              <span v-if="savingProxy">Menguji & Menerapkan...</span>
              <span v-else>Uji (nginx -t) & Terapkan</span>
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Modal Let's Encrypt SSL -->
    <div v-if="showSSLModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm">
      <div class="bg-slate-900 border border-slate-800 rounded-xl w-full max-w-md p-6 shadow-2xl">
        <div class="flex items-center justify-between mb-3">
          <h3 class="text-lg font-bold text-white flex items-center gap-2">
            <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/></svg>
            Let's Encrypt ACME SSL
          </h3>
          <button @click="showSSLModal = false" class="text-slate-500 hover:text-white">&times;</button>
        </div>

        <p class="text-xs text-slate-400 mb-4">
          Dapatkan sertifikat HTTPS gratis yang otomatis terbarui untuk domain <strong class="text-white">{{ activeSite?.domain }}</strong>.
        </p>

        <form @submit.prevent="issueSSL" class="space-y-4">
          <div>
            <label class="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">Email Pemberitahuan Kedaluwarsa</label>
            <input
              v-model="sslEmail"
              type="email"
              required
              placeholder="admin@domain.com"
              class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3.5 py-2.5 text-sm text-white focus:outline-none focus:border-rose-500"
            />
          </div>

          <div v-if="sslStatusMessage" class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-300">
            {{ sslStatusMessage }}
          </div>

          <div class="flex items-center justify-end gap-3 pt-3 border-t border-slate-800">
            <button
              type="button"
              @click="showSSLModal = false"
              class="px-4 py-2 text-sm font-medium text-slate-400 hover:text-slate-200"
            >
              Batal
            </button>
            <button
              type="submit"
              :disabled="issuingSSL"
              class="px-5 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm font-semibold rounded-lg shadow-md transition"
            >
              {{ issuingSSL ? 'Memproses Challenge...' : 'Terbitkan & Pasang SSL' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Modal Buat Website Baru -->
    <div v-if="showCreateModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
      <div class="bg-slate-900 border border-slate-800 rounded-xl w-full max-w-lg p-6 shadow-2xl">
        <h3 class="text-xl font-bold text-white mb-1">Tambah Website Baru</h3>
        <p class="text-xs text-slate-400 mb-5">Otomatis membuat virtual host, direktori web, dan ZFS dataset.</p>

        <form @submit.prevent="createWebsite" class="space-y-4">
          <div>
            <label class="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">Nama Domain</label>
            <input
              v-model="newSite.domain"
              type="text"
              placeholder="contoh: aplikasi-web.id"
              required
              class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3.5 py-2.5 text-sm text-white focus:outline-none focus:border-rose-500"
            />
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">Tipe / Mode</label>
              <select
                v-model="newSite.is_reverse_proxy"
                class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3.5 py-2.5 text-sm text-white focus:outline-none focus:border-rose-500"
              >
                <option :value="false">Standard PHP Hosting</option>
                <option :value="true">Nginx Reverse Proxy</option>
              </select>
            </div>

            <div v-if="!newSite.is_reverse_proxy">
              <label class="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">PHP Version</label>
              <select
                v-model="newSite.php_version"
                class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3.5 py-2.5 text-sm text-white focus:outline-none focus:border-rose-500"
              >
                <option value="8.3">PHP 8.3 (Terbaru)</option>
                <option value="8.2">PHP 8.2</option>
                <option value="8.1">PHP 8.1</option>
                <option value="none">Tanpa PHP (HTML/Static)</option>
              </select>
            </div>

            <div v-else>
              <label class="block text-xs font-semibold text-slate-300 uppercase tracking-wider mb-1">Target Upstream</label>
              <input
                v-model="newSite.proxy_target"
                type="text"
                placeholder="http://127.0.0.1:3000"
                class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-rose-500 font-mono"
              />
            </div>
          </div>

          <div class="flex items-center gap-3 pt-2">
            <input
              id="ssl"
              type="checkbox"
              v-model="newSite.enable_ssl"
              class="rounded bg-slate-950 border-slate-800 text-rose-600 focus:ring-rose-500"
            />
            <label for="ssl" class="text-sm text-slate-300">Aktifkan SSL Let's Encrypt Otomatis</label>
          </div>

          <div class="flex items-center justify-end gap-3 pt-4 border-t border-slate-800">
            <button
              type="button"
              @click="showCreateModal = false"
              class="px-4 py-2 text-sm font-medium text-slate-400 hover:text-slate-200"
            >
              Batal
            </button>
            <button
              type="submit"
              class="px-5 py-2 bg-rose-600 hover:bg-rose-500 text-white text-sm font-semibold rounded-lg shadow-md transition"
            >
              Deploy Website
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const showCreateModal = ref(false)
const showProxyModal = ref(false)
const showSSLModal = ref(false)

const activeSite = ref(null)
const sslEmail = ref('admin@host.local')
const issuingSSL = ref(false)
const sslStatusMessage = ref('')

const savingProxy = ref(false)
const nginxTestOutput = ref('')

const proxyForm = ref({
  is_reverse_proxy: true,
  proxy_target: 'http://127.0.0.1:3000',
  proxy_websocket: true,
  proxy_buffering: true,
  max_body_size_mb: 50,
})

const websites = ref([
  {
    id: 1,
    domain: 'portal-madrasah.sch.id',
    document_root: '/usr/local/www/portal-madrasah.sch.id/public_html',
    web_server: 'nginx',
    php_version: '8.3',
    ssl_enabled: true,
    is_reverse_proxy: false,
    proxy_target: '',
    proxy_websocket: true,
    status: 'active',
  },
  {
    id: 2,
    domain: 'api.layanan-internal.id',
    document_root: '/usr/local/www/api.layanan-internal.id/public_html',
    web_server: 'nginx',
    php_version: '8.2',
    ssl_enabled: false,
    is_reverse_proxy: true,
    proxy_target: 'http://127.0.0.1:8080',
    proxy_websocket: true,
    status: 'active',
  }
])

const newSite = ref({
  domain: '',
  web_server: 'nginx',
  php_version: '8.3',
  is_reverse_proxy: false,
  proxy_target: 'http://127.0.0.1:3000',
  proxy_websocket: true,
  enable_ssl: false,
})

const fetchWebsites = async () => {
  try {
    const res = await fetch('/api/websites')
    if (res.ok) {
      const data = await res.json()
      if (data && data.length > 0) {
        websites.value = data
      }
    }
  } catch (err) {
    // fallback sample
  }
}

const openProxyModal = (site) => {
  activeSite.value = site
  proxyForm.value = {
    is_reverse_proxy: site.is_reverse_proxy || false,
    proxy_target: site.proxy_target || 'http://127.0.0.1:3000',
    proxy_websocket: site.proxy_websocket ?? true,
    proxy_buffering: site.proxy_buffering ?? true,
    max_body_size_mb: 50,
  }
  nginxTestOutput.value = ''
  showProxyModal.value = true
}

const saveProxySettings = async () => {
  if (!activeSite.value) return
  savingProxy.value = true
  nginxTestOutput.value = ''

  try {
    const res = await fetch(`/api/websites/${activeSite.value.id}/proxy`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(proxyForm.value),
    })
    const data = await res.json()
    if (res.ok) {
      nginxTestOutput.value = `✔ ${data.message}\n${data.nginx_test || ''}`
      activeSite.value.is_reverse_proxy = proxyForm.value.is_reverse_proxy
      activeSite.value.proxy_target = proxyForm.value.proxy_target
      activeSite.value.proxy_websocket = proxyForm.value.proxy_websocket
      setTimeout(() => {
        showProxyModal.value = false
        fetchWebsites()
      }, 1500)
    } else {
      alert(`Gagal: ${data.error}`)
    }
  } catch (err) {
    alert(`Error: ${err.message}`)
  } finally {
    savingProxy.value = false
  }
}

const openSSLModal = (site) => {
  activeSite.value = site
  sslEmail.value = 'admin@' + site.domain
  sslStatusMessage.value = ''
  showSSLModal.value = true
}

const issueSSL = async () => {
  if (!activeSite.value) return
  issuingSSL.value = true
  sslStatusMessage.value = 'Sedang meminta sertifikat ke Let\'s Encrypt via ACME...'

  try {
    const res = await fetch('/api/ssl/issue', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        website_id: activeSite.value.id,
        email: sslEmail.value,
      }),
    })
    const data = await res.json()
    if (res.ok) {
      sslStatusMessage.value = `✔ ${data.message}`
      activeSite.value.ssl_enabled = true
      setTimeout(() => {
        showSSLModal.value = false
        fetchWebsites()
      }, 1500)
    } else {
      sslStatusMessage.value = `Gagal: ${data.error}`
    }
  } catch (err) {
    sslStatusMessage.value = `Error: ${err.message}`
  } finally {
    issuingSSL.value = false
  }
}

const createWebsite = async () => {
  try {
    const res = await fetch('/api/websites', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newSite.value),
    })
    if (res.ok) {
      showCreateModal.value = false
      fetchWebsites()
    }
  } catch (err) {
    showCreateModal.value = false
  }
}

onMounted(() => {
  fetchWebsites()
})
</script>
