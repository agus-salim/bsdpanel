# BSDPanel 😈

**BSDPanel** adalah Web Hosting Control Panel modern, mandiri (*single static binary*), dan ultra-ringan yang dirancang khusus untuk sistem operasi **FreeBSD**. 

Dibangun dengan **Go (Golang)** dan antarmuka web responsif berbasis **Tailwind CSS** dengan font **Fira Sans**, BSDPanel mengutamakan performa tinggi, jejak memori minimal (< 15MB RSS RAM), keamanan tingkat kernel FreeBSD, serta kemudahan instalasi.

---

## 🌟 Fitur Utama

- **Single Binary Mandiri:** Semua template HTML, CSS, dan aset di-embed langsung ke dalam satu binary Go (`embed.FS`). Tidak memerlukan runtime eksternal (Node.js/Python).
- **FreeBSD Native Process Orchestration:** Terintegrasi langsung dengan perkakas native FreeBSD:
  - `service` dan `rc.d` untuk pengelolaan daemon tanpa downtime (*graceful reload*).
  - `pw` untuk isolasi akun pengguna UNIX FreeBSD.
  - `pkg` untuk manajemen paket dan dependensi.
  - `sysctl` & `sockstat` untuk pemantauan CPU, RAM (*page stats*), dan port terbuka secara real-time.
  - Dukungan filesystem **OpenZFS** dan UFS.
- **Multi-Webserver:** Orkestrasi otomatis untuk **Nginx** dan **Apache 2.4** dengan pengujian konfigurasi otomatis (`nginx -t`) dan rollback jika terjadi kegagalan sintaks.
- **Multi-PHP Isolation:** Isolasi pool PHP-FPM per user melalui UNIX domain socket (`/var/run/php{version}-fpm-{user}.sock`) dengan penyesuaian direktif `php.ini` (`memory_limit`, `upload_max_filesize`, `max_execution_time`).
- **Multi-Database:** Manajemen database dan user untuk **MariaDB/MySQL** dan **PostgreSQL**.
- **Autentikasi Aman:** Token sesi stateless berbasis HMAC-SHA256 dengan Role-Based Access Control (RBAC).

---

## 📁 Struktur Direktori Proyek

```text
bsdpanel/
├── cmd/
│   └── bsdpanel/
│       └── main.go                  # Entry point, flag parsing & graceful shutdown
├── internal/
│   ├── api/
│   │   ├── router.go                # HTTP router standar Go 1.22+ & middleware
│   │   └── handlers.go              # Handler dashboard, vhosts, services, & API
│   ├── auth/
│   │   └── auth.go                  # Stateless HMAC session & RBAC
│   ├── config/
│   │   └── config.go                # Konfigurasi & default FreeBSD paths
│   ├── database/
│   │   └── database.go              # Wrapper MariaDB/MySQL & PostgreSQL
│   ├── php/
│   │   └── php_fpm.go               # Generator & manager pool PHP-FPM FreeBSD
│   ├── system/
│   │   ├── runner.go                # Safe exec.Command wrapper (timeout & mock support)
│   │   ├── service.go               # rc.d & sysrc wrapper (/etc/rc.conf)
│   │   ├── stats.go                 # sysctl, df, sockstat FreeBSD metrics collector
│   │   ├── user.go                  # pw useradd / userdel UNIX isolation
│   │   └── pkg.go                   # pkg install & query wrapper
│   └── webserver/
│       └── vhost.go                 # Nginx & Apache virtual host orchestrator
├── web/
│   ├── templates/                   # Go html/template
│   │   ├── layout.html              # Layout utama Tailwind CSS + Fira Sans
│   │   ├── dashboard.html           # Dashboard metrik & status layanan
│   │   ├── vhosts.html              # Antarmuka manajemen domain
│   │   ├── services.html            # Pengendali daemon FreeBSD
│   │   └── php.html                 # Pengaturan direktif PHP
│   ├── static/                      # Aset statis pendukung
│   └── embed.go                     # Directive //go:embed
├── templates/
│   ├── nginx/
│   │   └── vhost.conf.tmpl          # Template vhost Nginx teroptimasi FreeBSD
│   └── php-fpm/
│       └── pool.conf.tmpl           # Template pool PHP-FPM terisolasi per user
├── scripts/
│   ├── bsdpanel.rc                  # FreeBSD rc.subr init script (/usr/local/etc/rc.d)
│   └── install.sh                   # One-line installer FreeBSD (/bin/sh)
├── Makefile                         # Skrip kompilasi cross-platform & release
├── go.mod                           # Go Module definisi
└── README.md
```

---

## 🚀 Instalasi Cepat di FreeBSD

Jalankan perintah berikut di FreeBSD server Anda sebagai `root`:

```sh
# Menggunakan skrip installer otomatis:
fetch -o - https://raw.githubusercontent.com/agus-salim/bsdpanel/main/scripts/install.sh | sh
```

Atau unduh dan jalankan manual:

```sh
git clone https://github.com/agus-salim/bsdpanel.git
cd bsdpanel
sh scripts/install.sh
```

Installer akan otomatis:
1. Memverifikasi OS FreeBSD (amd64).
2. Memasang dependensi dasar via `pkg install` (`nginx`, `php83`, `mariadb1011-server`, dll).
3. Memasang binary ke `/usr/local/sbin/bsdpanel`.
4. Memasang skrip rc.d ke `/usr/local/etc/rc.d/bsdpanel`.
5. Mengaktifkan daemon di `/etc/rc.conf` via `sysrc bsdpanel_enable="YES"`.
6. Menjalankan layanan pada port `8080`.

---

## 🛠️ Kompilasi & Pembangunan (Cross-Compile)

Anda dapat mengompilasi binary FreeBSD langsung dari komputer lokal Anda (Linux/macOS/Windows) tanpa memerlukan mesin FreeBSD berkat kapabilitas Go cross-compilation:

```sh
# Kompilasi khusus untuk target FreeBSD amd64:
make build-freebsd

# Hasil binary berada di:
# bin/freebsd_amd64/bsdpanel
```

Untuk menguji UI secara lokal di lingkungan pengembangan (Windows/Mac/Linux):
```sh
make run-dev
# Buka http://localhost:8080 di browser
```

---

## ⚙️ Pengendalian Layanan di FreeBSD

BSDPanel terintegrasi dengan sistem `rc.d` FreeBSD:

```sh
# Memeriksa status panel:
service bsdpanel status

# Menyalakan panel:
service bsdpanel start

# Mematikan panel:
service bsdpanel stop

# Memuat ulang / restart:
service bsdpanel restart
```

Konfigurasi disimpan di `/usr/local/etc/bsdpanel/config.json`.

---

## 📄 Lisensi

BSD 2-Clause License. Hak Cipta (c) Agus Salim.
