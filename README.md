# BSD Panel — Lightweight FreeBSD VPS Control Panel

Panel kontrol VPS berkinerja tinggi, hemat sumber daya (lightweight), dan aman yang dirancang khusus untuk sistem operasi **FreeBSD (13.x/14.x)** menggunakan **Go (Golang)** murni dan antarmuka **Tailwind CSS** lokal.

---

## ⚡ Fitur Utama

- **Zero Bloat & Hemat RAM**: Daemon Go hanya membutuhkan memory < 30 MB.
- **Model Isolasi Non-Root**: Setiap website berjalan di bawah user FreeBSD terisolasi (`u_<domain>`), PHP-FPM pool terisolasi, dan home folder berizin `0750`.
- **Multi-Web Server**: Mendukung Nginx, Apache 2.4, Caddy, dan OpenLiteSpeed.
- **Multi-Versi PHP**: PHP 8.2, 8.3, 8.4, hingga 8.5 dengan ekstensi lengkap via `pkg`.
- **Manajemen Database**: MariaDB dan PostgreSQL lengkap dengan otomatisasi user & hak akses.
- **Keamanan FreeBSD Native**: FreeBSD PF (Packet Filter) & Fail2ban.
- **Frontend 100% Offline**: Menggunakan Tailwind CSS lokal, tipografi Fira Sans lokal, dan Lucide Icons SVG lokal tanpa ketergantungan CDN eksternal.

---

## 🚀 Panduan Ringkas Menjalankan di FreeBSD

### 1. Unduh Source Code

**Opsi A — Menggunakan Git:**
```sh
pkg install -y git go
git clone https://github.com/agus-salim/bsdpanel.git
cd bsdpanel
```

> **Catatan Troubleshooting `libpcre2`:**
> Jika saat menjalankan `git` muncul error:
> `ld-elf.so.1: /usr/local/lib/libpcre2-8.so.0: version PCRE2_10.47 required by /usr/local/bin/git not defined`
> Perbaiki library FreeBSD Anda dengan perintah:
> ```sh
> pkg install -fy pcre2 git
> # atau update seluruh paket sistem:
> pkg upgrade -y
> ```

**Opsi B — Menggunakan `fetch` bawaan FreeBSD (Tanpa Git):**
```sh
fetch https://github.com/agus-salim/bsdpanel/archive/refs/heads/main.tar.gz
tar -zxvf main.tar.gz
cd bsdpanel-main
```

### 2. Build Binary
```sh
go build -o /usr/local/bin/bsdpanel ./cmd/bsdpanel
```

### 3. Konfigurasi Service FreeBSD
```sh
# Salin service rc.d
cp deploy/rc.d/bsdpanel /usr/local/etc/rc.d/bsdpanel
chmod +x /usr/local/etc/rc.d/bsdpanel

# Tambah user daemon
pw useradd bsdpanel -m -d /var/db/bsdpanel -s /usr/sbin/nologin

# Konfigurasi doas (least privilege)
cp deploy/doas.conf /usr/local/etc/doas.conf
chmod 0600 /usr/local/etc/doas.conf

# Aktifkan di rc.conf
sysrc bsdpanel_enable="YES"
service bsdpanel start
```

### 4. Nginx Reverse Proxy
```sh
cp deploy/nginx/bsdpanel.conf /usr/local/etc/nginx/conf.d/bsdpanel.conf
nginx -t && service nginx reload
```

---

Dokumentasi rancangan teknis lengkap tersedia di file [ARCHITECTURE.md](file:///c:/Users/Operator%20MTsN%201%20SKD/BSD%20Panel/ARCHITECTURE.md).
