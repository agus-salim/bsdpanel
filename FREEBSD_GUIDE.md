# BSD Panel — Panduan Teknis & Pengoperasian di FreeBSD

**BSD Panel** adalah sistem kontrol panel web hosting modern yang dirancang khusus untuk sistem operasi **FreeBSD (FreeBSD 13.x, 14.x, 15-CURRENT)**. Dibangun dengan backend **Rust (Axum/Tokio)** yang sangat ringan, aman dari memory corruption, dan menggunakan footprint memori sangat minim, dipadukan dengan frontend **Vue 3** berbasis **Fira Sans** dan pustaka ikon lengkap.

---

## 1. Arsitektur & Keamanan FreeBSD

FreeBSD memiliki karakteristik keamanan yang sangat ketat:
- **rc.d & sysrc**: Semua service dikontrol melalui skrip rc standar (`/usr/sbin/service <name> [start|stop|restart|reload|status]`) dan dideklarasikan di `/etc/rc.conf`.
- **Packet Filter (PF)**: Firewall kernel FreeBSD yang dikendalikan melalui `pfctl` dan `/etc/pf.conf`. Menggunakan tabel dinamis `<bruteforce>` dan `<whitelist>`.
- **Fail2ban Integration**: Mendeteksi brute-force SSH, login panel, dan HTTP bot scanning langsung dari log FreeBSD (`/var/log/auth.log`, `/var/log/nginx/error.log`).
- **User Isolation**: Pengguna dibuat menggunakan utility bawaan `/usr/sbin/pw useradd` dengan home directory `/home/<user>` atau direktori web terisolasi `/usr/local/www/<domain>/public`.
- **OpenZFS Quota Ready**: Siap terintegrasi dengan dataset ZFS (`zroot`) untuk pembatasan kuota disk yang presisi.

---

## 2. Fitur Utama

### A. Multi Web Server
Mendukung 4 web server sekaligus dengan switchboard aktif:
1. **NGINX**: Reverse proxy & web server berkecepatan tinggi dengan FastCGI caching.
2. **Apache HTTPD (httpd24)**: Web server fleksibel dengan dukungan `.htaccess`.
3. **OpenLiteSpeed (LSWS)**: Web server event-driven dengan kompatibilitas LSAPI.
4. **Caddy**: Web server modern dengan otomatisasi HTTPS/TLS Let's Encrypt bawaan.
- Generator VHost otomatis untuk masing-masing webserver.
- Modal preview konfigurasi langsung di web panel.

### B. Multi PHP Version (8.2 - 8.6)
- Mendukung instalasi paralel versi PHP dari FreeBSD Ports / Packages (`lang/php82` hingga `lang/php86`).
- Setiap versi berjalan sebagai daemon **PHP-FPM** independen dengan Unix socket terisolasi:
  - `/var/run/php-fpm-82.sock` (Port 9002)
  - `/var/run/php-fpm-83.sock` (Port 9003)
  - `/var/run/php-fpm-84.sock` (Port 9004)
  - `/var/run/php-fpm-85.sock` (Port 9005)
  - `/var/run/php-fpm-86.sock` (Port 9006)
- **PHP Management**:
  - Live editor directive `php.ini` (`memory_limit`, `upload_max_filesize`, `post_max_size`, `max_execution_time`, `date.timezone`, dll).
  - Saklar on/off ekstensi (`opcache`, `mbstring`, `curl`, `pdo_mysql`, `pdo_pgsql`, `gd`, `zip`, `redis`, `imagick`, `sodium`).
  - Penyetelan worker pool (`pm.max_children`, `pm.start_servers`, `pm.min_spare_servers`, `pm.max_spare_servers`, `pm.max_requests`).

### C. Multi Database (PostgreSQL & MariaDB)
- **MariaDB 10.11 LTS** (Port 3306) & **PostgreSQL 16** (Port 5432).
- Pembuatan dan penghapusan database dengan charset/collation pilihan.
- Manajemen pengguna database, penetapan hak akses basis data, dan allowed host.

### D. Firewall (PF) & Fail2ban
- Status aktif Packet Filter dengan pemantauan counter state kernel.
- Saklar toggle On/Off Packet Filter.
- Tabel Aturan Port kustom (TCP/UDP, Pass/Block, In/Out, Source CIDR).
- Sinkronisasi instan ke `/etc/pf.conf`.
- Integrasi Fail2ban Jails (`bsd-sshd`, `bsd-panel-auth`, `nginx-badbots`).
- Tabel live IP yang diblokir dengan tombol unban instan dan tombol manual ban.

### E. User & Reseller Hierarchy
- **Super Admin (Root)**: Kontrol menyeluruh seluruh konfigurasi sistem FreeBSD.
- **Reseller**: Alokasi kuota paket untuk dijual kembali ke sub-user.
- **User**: Pengguna akhir yang hanya dapat mengakses website, database, dan file miliknya.
- Indikator bar visual untuk pemakaian Disk Quota (MB/GB) dan Bandwidth.

---

## 3. Cara Instalasi di FreeBSD

### Langkah 1: Clone Repositori
```bash
git clone https://github.com/agus-salim/bsdpanel.git /usr/local/src/bsdpanel
cd /usr/local/src/bsdpanel
```

### Langkah 2: Jalankan Skrip Instalasi Otomatis
```bash
chmod +x scripts/install-freebsd.sh
./scripts/install-freebsd.sh
```

Skrip ini akan secara otomatis:
1. Memperbarui repositori package FreeBSD (`pkg update`).
2. Memasang compiler Rust, Cargo, Node.js, NPM, Git, sudo, dan Fail2ban.
3. Memasang Nginx, Apache24, Caddy, PHP 8.2 s/d 8.6, MariaDB, dan PostgreSQL.
4. Melakukan kompilasi daemon backend Rust (`cargo build --release`).
5. Membangun aset frontend Vue 3 (`npm run build`).
6. Memasang rc.d script ke `/usr/local/etc/rc.d/bsdpanel`.
7. Mengaktifkan konfigurasi di `/etc/rc.conf`.

### Langkah 3: Menjalankan Service
```bash
service bsdpanel start
```

Periksa status:
```bash
service bsdpanel status
```

### Langkah 4: Mengakses Web Panel
Buka browser dan akses:
```
http://IP_SERVER_FREEBSD:8888
```
- **Default Username**: `root`
- **Default Password**: Password root yang Anda tentukan saat instalasi.

---

## 4. Struktur Direktori Proyek

```
BSD Panel/
├── backend/                  # Daemon Rust (bsdpaneld)
│   ├── Cargo.toml            # Konfigurasi Cargo & Dependensi Axum/Tokio
│   └── src/
│       ├── main.rs           # Entrypoint daemon, HTTP & Web listener
│       ├── config.rs         # Path FreeBSD & environment settings
│       ├── state.rs          # In-memory thread-safe state
│       ├── auth/             # JWT authentication & password hashing
│       ├── models/           # Struktur data (User, WebServer, PHP, DB, PF)
│       ├── services/         # Logika bisnis & generator VHost/PF
│       ├── freebsd/          # Integrasi native rc.d, sysrc, pfctl, fail2ban, pw
│       └── api/              # REST API controllers
├── frontend/                 # UI Single Page Application (Vue 3)
│   ├── index.html            # Entrypoint dengan Google Font "Fira Sans"
│   ├── package.json          # Dependensi Vue 3, Pinia, Vue Router, Lucide, Iconify
│   ├── src/
│   │   ├── main.js           # Vue bootstrapping
│   │   ├── style.css         # Desain sistem Fira Sans & tema gelap/terang
│   │   ├── App.vue           # Shell navigasi & header bar responsif
│   │   ├── components/
│   │   │   └── AppIcon.vue   # Universal Icon bridging (100% ketersediaan icon)
│   │   ├── stores/
│   │   │   └── bsdStore.js   # State management Pinia
│   │   └── views/
│   │       ├── DashboardView.vue   # Telemetri FreeBSD & ZFS status
│   │       ├── WebserversView.vue  # Nginx, Apache, LiteSpeed, Caddy
│   │       ├── PhpManagerView.vue  # PHP 8.2 - 8.6, php.ini, extensions
│   │       ├── DatabasesView.vue   # PostgreSQL 16 & MariaDB 10.11
│   │       ├── FirewallView.vue    # PF Firewall & Fail2ban
│   │       ├── UsersView.vue       # User & Reseller hierarchy
│   │       └── ServicesView.vue    # rc.d daemon & log viewer
├── scripts/
│   ├── install-freebsd.sh    # Otomatisasi instalasi FreeBSD
│   └── rc.d/bsdpanel         # Skrip FreeBSD rc.d daemon service
└── README.md
```
