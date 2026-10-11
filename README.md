# BSDPanel — Native FreeBSD Web Hosting Control Panel

**BSDPanel** adalah Web Hosting Control Panel modern, ringan, dan berperforma tinggi yang dirancang khusus untuk berjalan secara natif di atas sistem operasi **FreeBSD**.

---

## 🚀 Fitur Utama & Integrasi FreeBSD

1. **Backend Mandiri (Single Binary Go):**
   - Dikompilasi menjadi satu binary executable mandiri (`/usr/local/sbin/bsdpanel`).
   - Sangat hemat memori (~20-40 MB RAM) dan tanpa runtime berat seperti Python atau Ruby.
   - Manajemen database panel internal berbasis **SQLite (WAL Mode)**.

2. **Web Terminal Terintegrasi (xterm.js & WebSocket PTY):**
   - Menggunakan pseudo-terminal (PTY) interaktif FreeBSD di `/bin/sh`.
   - Mendukung eksekusi langsung ke dalam container FreeBSD Jail via `/usr/sbin/jexec <jail_name> /bin/sh`.
   - Deteksi otomatis *window resize* (`cols`, `rows`) untuk kenyamanan penggunaan `htop`, `vi`, `top`, dll.

3. **Nginx Reverse Proxy GUI:**
   - GUI untuk mengatur target upstream (contoh: `http://127.0.0.1:3000`, `http://127.0.0.1:8080`).
   - Opsi switch WebSocket upgrade (`Upgrade $http_upgrade; Connection "upgrade"`).
   - Pengaturan Client Max Body Size dan Proxy Buffering.
   - **Validasi Syntax Otomatis:** Memanggil `/usr/local/sbin/nginx -t` sebelum reload, mencegah website down jika ada kesalahan konfigurasi.

4. **Sertifikat SSL Otomatis Let's Encrypt (ACME):**
   - Penerbitan otomatis via protokol ACME (didukung `certbot` atau `acme.sh` di FreeBSD).
   - Otomatis memperbarui file konfigurasi Virtual Host Nginx dengan parameter HTTPS, HTTP/2, TLSv1.2 & TLSv1.3 modern.

5. **Multi-PHP Management (PHP 8.1, 8.2, 8.3):**
   - Menjalankan berbagai versi PHP secara berdampingan memanfaatkan port FreeBSD `lang/php81`, `lang/php82`, `lang/php83`.
   - Tiap versi berkomunikasi melalui UNIX Domain Socket (`/var/run/php{version}-fpm.sock`) untuk latensi terendah.

6. **Isolasi Keamanan & Storage FreeBSD:**
   - **FreeBSD Jails (`jail` / `jexec`):** Mengisolasi akun pengguna ke dalam container level kernel yang sangat aman.
   - **ZFS Dataset & Quota:** Setiap website dan user memiliki dataset ZFS tersendiri (`zroot/bsdpanel/{user}/{domain}`) dengan snapshot berkala instan.
   - **PF (Packet Filter):** Integrasi firewall bawaan FreeBSD untuk proteksi brute force dan rate-limiting.

7. **Frontend Modern (Vue 3 + Tailwind CSS):**
   - Tipografi elegan menggunakan **Fira Sans** & **Fira Code**.
   - Desain modern bernuansa *dark mode*.

---

## 📂 Struktur Direktori

```text
bsdpanel/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go               # Entry point, HTTP server & graceful shutdown
│   ├── internal/
│   │   ├── api/
│   │   │   └── handlers.go           # REST API & WebSocket Terminal Route
│   │   ├── config/
│   │   │   └── config.go             # Parser konfigurasi panel.yaml
│   │   ├── database/
│   │   │   └── db.go                 # SQLite WAL mode & Schema DDL
│   │   ├── ssl/
│   │   │   └── acme.go               # ACME Let's Encrypt Certificate Manager
│   │   ├── system/
│   │   │   └── freebsd.go            # FreeBSD Controller (service, sysctl, zfs, jail)
│   │   ├── terminal/
│   │   │   ├── terminal.go           # WebSocket upgrader & session multiplexer
│   │   │   ├── pty_unix.go           # PTY driver FreeBSD/Unix (creack/pty & jexec)
│   │   │   └── pty_windows.go        # Windows dev mode fallback
│   │   └── vhost/
│   │       └── generator.go          # VHost Generator (PHP, Reverse Proxy, nginx -t)
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── views/
│   │   │   ├── Dashboard.vue         # Status FreeBSD ZFS ARC, Load, Jails & Services
│   │   │   ├── Websites.vue          # Manajemen Web, Nginx Proxy GUI & Let's Encrypt
│   │   │   └── Terminal.vue          # Web Terminal interaktif (xterm.js + PTY)
│   │   ├── App.vue                   # Layout utama, sidebar & top navbar
│   │   ├── main.js                   # Inisialisasi Vue 3, Pinia & Vue Router
│   │   └── style.css                 # Setup Tailwind & Font Fira Sans / Fira Code
│   ├── index.html                    # Loader Google Font Fira Sans & Fira Code
│   ├── tailwind.config.js
│   └── vite.config.js
├── config/
│   └── panel.yaml                    # Konfigurasi sistem BSDPanel
└── scripts/
    ├── rc.d/
    │   └── bsdpanel                  # FreeBSD rc.d daemon script (/usr/local/etc/rc.d)
    └── setup-freebsd.sh              # Script instalasi otomatis ports/pkg FreeBSD
```

---

## 🛠 Panduan Build & Deployment di FreeBSD

### 1. Build Backend Go
```bash
cd backend
# Kompilasi langsung di mesin FreeBSD:
go build -ldflags="-s -w" -o /usr/local/sbin/bsdpanel ./cmd/server

# Cross-compile dari Windows/Linux untuk FreeBSD:
GOOS=freebsd GOARCH=amd64 go build -ldflags="-s -w" -o bsdpanel ./cmd/server
```

### 2. Build Frontend Vue 3
```bash
cd frontend
npm install
npm run build
```

### 3. Setup FreeBSD Host & Menjalankan Daemon
```bash
sh scripts/setup-freebsd.sh
cp scripts/rc.d/bsdpanel /usr/local/etc/rc.d/bsdpanel
chmod +x /usr/local/etc/rc.d/bsdpanel
sysrc bsdpanel_enable="YES"
service bsdpanel start
```
Akses control panel di: `http://IP_SERVER:8083`
