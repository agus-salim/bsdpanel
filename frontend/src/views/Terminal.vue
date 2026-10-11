<template>
  <div class="space-y-4 max-w-7xl mx-auto h-[calc(100vh-10rem)] flex flex-col">
    <!-- Header & Controls -->
    <div class="flex items-center justify-between bg-slate-900 border border-slate-800 px-5 py-3 rounded-xl shadow-sm">
      <div class="flex items-center gap-3">
        <div class="w-3 h-3 rounded-full" :class="isConnected ? 'bg-emerald-400 shadow-sm shadow-emerald-400/50' : 'bg-rose-500'"></div>
        <div>
          <h2 class="text-sm font-bold text-white flex items-center gap-2">
            FreeBSD Web Terminal
            <span class="text-xs px-2 py-0.5 rounded font-mono font-normal" :class="isConnected ? 'bg-emerald-500/10 text-emerald-400' : 'bg-rose-500/10 text-rose-400'">
              {{ isConnected ? 'Online (PTY Connected)' : 'Disconnected' }}
            </span>
          </h2>
          <p class="text-xs text-slate-400">Akses shell FreeBSD langsung dari browser melalui WebSocket PTY.</p>
        </div>
      </div>

      <!-- Action Controls -->
      <div class="flex items-center gap-3">
        <!-- Target Jail Selector -->
        <div class="flex items-center gap-2 text-xs">
          <label class="text-slate-400 font-medium">Target Lingkungan:</label>
          <select
            v-model="selectedJail"
            @change="reconnect"
            class="bg-slate-950 border border-slate-800 rounded-lg px-2.5 py-1 text-slate-200 text-xs focus:outline-none focus:border-rose-500 font-mono"
          >
            <option value="">Host Kernel (/bin/sh)</option>
            <option value="jail-client1">Jail: client1-jail (jexec)</option>
            <option value="jail-client2">Jail: client2-jail (jexec)</option>
          </select>
        </div>

        <button
          @click="clearTerminal"
          class="px-3 py-1 bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-medium rounded-lg border border-slate-700 transition"
        >
          Clear
        </button>

        <button
          @click="reconnect"
          class="px-3 py-1 bg-rose-600 hover:bg-rose-500 text-white text-xs font-medium rounded-lg transition shadow flex items-center gap-1.5"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>
          Reconnect
        </button>
      </div>
    </div>

    <!-- Terminal Container -->
    <div class="flex-1 bg-[#090d16] border border-slate-800 rounded-xl overflow-hidden p-3 relative shadow-inner">
      <div ref="terminalContainer" class="w-full h-full"></div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

const terminalContainer = ref(null)
const isConnected = ref(false)
const selectedJail = ref('')

let term = null
let fitAddon = null
let ws = null

const initTerminal = () => {
  if (!terminalContainer.value) return

  term = new Terminal({
    cursorBlink: true,
    fontFamily: '"Fira Code", monospace',
    fontSize: 13,
    lineHeight: 1.25,
    theme: {
      background: '#090d16',
      foreground: '#e2e8f0',
      cursor: '#f43f5e',
      cursorAccent: '#090d16',
      selectionBackground: 'rgba(244, 63, 94, 0.3)',
      black: '#1e293b',
      red: '#f43f5e',
      green: '#10b981',
      yellow: '#f59e0b',
      blue: '#3b82f6',
      magenta: '#ec4899',
      cyan: '#06b6d4',
      white: '#f8fafc',
    },
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(terminalContainer.value)
  fitAddon.fit()

  connectWebSocket()

  // Handle user typing
  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  // Handle browser window resize
  window.addEventListener('resize', handleResize)
}

const handleResize = () => {
  if (fitAddon && term) {
    fitAddon.fit()
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: 'resize',
        cols: term.cols,
        rows: term.rows,
      }))
    }
  }
}

const connectWebSocket = () => {
  if (ws) {
    ws.close()
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  let url = `${protocol}//${host}/api/ws/terminal`
  if (selectedJail.value) {
    url += `?jail=${encodeURIComponent(selectedJail.value)}`
  }

  term.write('\r\n\x1b[33mConnecting to FreeBSD terminal...\x1b[0m\r\n')

  try {
    ws = new WebSocket(url)
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      isConnected.value = true
      term.write('\x1b[32m✔ Connected to FreeBSD Kernel PTY.\x1b[0m\r\n\r\n')
      handleResize()
    }

    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        term.write(event.data)
      } else {
        const text = new TextDecoder('utf-8').decode(event.data)
        term.write(text)
      }
    }

    ws.onclose = () => {
      isConnected.value = false
      term.write('\r\n\x1b[31mConnection closed.\x1b[0m\r\n')
    }

    ws.onerror = (err) => {
      isConnected.value = false
      term.write('\r\n\x1b[31mWebSocket error occurred.\x1b[0m\r\n')
    }
  } catch (e) {
    term.write(`\r\n\x1b[31mFailed to create WebSocket: ${e.message}\x1b[0m\r\n`)
  }
}

const clearTerminal = () => {
  if (term) term.clear()
}

const reconnect = () => {
  if (term) term.clear()
  connectWebSocket()
}

onMounted(() => {
  initTerminal()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  if (ws) ws.close()
  if (term) term.dispose()
})
</script>
