# Arsitektur & Panduan Teknis: BSD Panel (Lightweight FreeBSD VPS Control Panel)

Dokumen ini memuat rancangan arsitektur lengkap, tata letak direktori, mekanisme isolasi keamanan non-root, contoh kode Go, antarmuka Tailwind CSS lokal, serta panduan deployment pada sistem operasi **FreeBSD (13.x / 14.x)**.

---

## 1. Arsitektur & Model Keamanan Non-Root (Privilege Isolation)

### 1.1 Prinsip Desain
- **Lightweight & Zero-Bloat**: Ditulis menggunakan Go (Golang) murni dengan runtime statis tunggal (~15-25 MB). Memanfaatkan utilitas bawaan FreeBSD (`pw`, `sysrc`, `service`, `pfctl`, `pkg`).
- **Reverse Proxy Architecture**: Daemon Go mendengarkan port internal `127.0.0.1:8880`. Akses publik diarahkan melalui Nginx yang menangani terminasi SSL, rate limiting, dan WebSocket.
- **Isolasi Tingkat Pengguna (Non-Root)**:
  - Proses daemon panel dijalankan dengan user unprivileged `bsdpanel`.
  - Setiap situs web baru otomatis dibuatkan user sistem tersendiri (misal: `u_domain1`, `u_domain2`) dengan shell `/usr/sbin/nologin` dan home directory `/usr/home/<user>`.
  - File web ditempatkan di `/usr/home/<user>/public_html` dengan hak akses `0750`.
  - PHP-FPM berjalan dengan pool terpisah per user (`listen = /var/run/php-fpm-<user>.sock`, `user = <user>`, `group = <user>`). Dengan cara ini, celah keamanan pada satu situs tidak dapat menembus situs lain maupun sistem utama.
- **Eksekusi Perintah Administratif**: Menggunakan utilitas `doas` (`security/doas`) dengan konfigurasi `doas.conf` berprinsip *least privilege* (hanya perintah tertentu yang diizinkan tanpa kata sandi).

```
[ Klien / Browser ]
        │
        ▼ (Port 80/443 HTTPS)
┌────────────────────────────────────────────────────────┐
│  FreeBSD Host                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │ Nginx Reverse Proxy (Domain: panel.domain.com)   │  │
│  └───────────────────────┬──────────────────────────┘  │
│                          │ (127.0.0.1:8880)            │
│                          ▼                             │
│  ┌──────────────────────────────────────────────────┐  │
│  │ BSD Panel Daemon (Go - User: bsdpanel)           │  │
│  └───────┬──────────────────────────┬───────────────┘  │
│          │ (doas / syscall.Cred)    │                  │
│          ▼                          ▼                  │
│  ┌────────────────────┐     ┌───────────────────────┐  │
│  │ Tenant 1 (u_site1) │     │ Tenant 2 (u_site2)    │  │
│  │ ├─ public_html/    │     │ ├─ public_html/       │  │
│  │ └─ PHP-FPM Pool 1  │     │ └─ PHP-FPM Pool 2     │  │
│  └────────────────────┘     └───────────────────────┘  │
│                          │                             │
│                          ▼                             │
│  [ Native FreeBSD: PF Firewall | MariaDB | Postgresql ]│
└────────────────────────────────────────────────────────┘
```

---

## 2. Struktur Direktori Proyek

```
BSD Panel/
├── cmd/
│   └── bsdpanel/
│       └── main.go                  # Entry point daemon panel & embed assets
├── internal/
│   ├── config/
│   │   └── config.go                # Loader konfigurasi JSON/YAML
│   ├── system/
│   │   ├── executor.go              # Eksekutor shell aman & user switching
│   │   ├── user.go                  # Manajemen user FreeBSD (pw useradd/userdel)
│   │   ├── service.go               # FreeBSD rc.d & sysrc service manager
│   │   ├── pkg.go                   # FreeBSD pkg installer (Web server, DB, PHP)
│   │   └── firewall.go              # FreeBSD pf (Packet Filter) & fail2ban
│   ├── sites/
│   │   └── site_manager.go          # VHost generator & PHP-FPM pool per tenant
│   ├── database/
│   │   └── db_manager.go            # MariaDB & PostgreSQL provisioning/dump
│   └── web/
│       ├── server.go                # HTTP Server, router & security middleware
│       └── handlers.go              # REST API & Web Page Handlers
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   ├── tailwind.min.css     # Tailwind CSS lokal (offline)
│   │   │   └── fira-sans.css        # Definisi font Fira Sans lokal
│   │   ├── fonts/                   # File woff2/ttf Fira Sans
│   │   └── js/
│   │       └── lucide-offline.js    # Lucide Icons offline SVG hydrator
│   └── templates/
│       ├── layout.html              # Master layout (Sidebar, Header, Scripts)
│       ├── dashboard.html           # Ringkasan resource (CPU, RAM, UFS/ZFS)
│       ├── sites.html               # Manajemen VHost & isolasi user
│       ├── services.html            # Service Manager & Multi-PHP 8.2-8.5
│       ├── databases.html           # MariaDB & PostgreSQL manager
│       ├── firewall.html            # Kontrol PF Firewall & Fail2ban
│       └── terminal.html            # Web terminal & log viewer
├── deploy/
│   ├── nginx/
│   │   └── bsdpanel.conf            # Konfigurasi reverse proxy Nginx di FreeBSD
│   ├── rc.d/
│   │   └── bsdpanel                 # Script service FreeBSD rc.d
│   └── doas.conf                    # Aturan least privilege doas
├── config.json                      # Konfigurasi runtime default
├── go.mod                           # Modul Go
└── ARCHITECTURE.md                  # Dokumentasi arsitektur ini
```

---

## 3. Implementasi Kode Backend Kunci (Golang)

### 3.1 Eksekusi Perintah Shell dengan Isolasi Pengguna (`internal/system/executor.go`)
Fungsi ini mengeksekusi perintah shell pada konteks user FreeBSD non-root tertentu (`syscall.Credential` atau `doas -u <user>`):

```go
func (e *Executor) ExecuteAsUser(ctx context.Context, targetUser string, command string, args ...string) (*ExecutionResult, error) {
    if targetUser == "" || targetUser == "root" {
        return nil, fmt.Errorf("eksekusi terisolasi harus membidik user non-root, didapat: '%s'", targetUser)
    }

    u, err := user.Lookup(targetUser)
    if err != nil {
        return nil, fmt.Errorf("user '%s' tidak ditemukan di sistem: %w", targetUser, err)
    }

    var cmd *exec.Cmd
    if e.UseDoas {
        // Melalui utilitas doas FreeBSD
        doasArgs := append([]string{"-u", targetUser, command}, args...)
        cmd = exec.CommandContext(ctx, "/usr/local/bin/doas", doasArgs...)
    } else {
        // Melalui pergantian POSIX Credential murni
        uid, _ := strconv.ParseUint(u.Uid, 10, 32)
        gid, _ := strconv.ParseUint(u.Gid, 10, 32)
        cmd = exec.CommandContext(ctx, command, args...)
        cmd.Dir = u.HomeDir
        cmd.SysProcAttr = &syscall.SysProcAttr{
            Credential: &syscall.Credential{
                Uid: uint32(uid),
                Gid: uint32(gid),
            },
        }
    }

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err = cmd.Run()
    return &ExecutionResult{
        Stdout:   strings.TrimSpace(stdout.String()),
        Stderr:   strings.TrimSpace(stderr.String()),
        ExitCode: cmd.ProcessState.ExitCode(),
    }, err
}
```

### 3.2 Pembuatan User FreeBSD & Struktur Direktori (`internal/system/user.go`)
Menggunakan perintah native FreeBSD `/usr/sbin/pw`:

```go
func (u *UserManager) CreateSiteUser(ctx context.Context, username, homedir string) error {
    cmd := "/usr/sbin/pw"
    args := []string{
        "useradd", username,
        "-m",
        "-d", homedir,
        "-s", "/usr/sbin/nologin",
        "-c", fmt.Sprintf("BSDPanel User (%s)", username),
    }
    _, err := u.exec.Execute(ctx, cmd, args...)
    if err != nil {
        return err
    }

    // Kunci hak akses home direktori ke 0750
    _, _ = u.exec.Execute(ctx, "/bin/chmod", "0750", homedir)

    // Buat subdirektori web standar
    for _, sub := range []string{"public_html", "logs", "ssl", "tmp"} {
        p := filepath.Join(homedir, sub)
        _, _ = u.exec.Execute(ctx, "/bin/mkdir", "-p", p)
        _, _ = u.exec.Execute(ctx, "/usr/sbin/chown", "-R", fmt.Sprintf("%s:%s", username, username), p)
    }
    return nil
}
```

---

## 4. Konfigurasi Nginx Reverse Proxy di FreeBSD

Simpan file di `/usr/local/etc/nginx/conf.d/bsdpanel.conf`:

```nginx
upstream bsdpanel_backend {
    server 127.0.0.1:8880;
    keepalive 32;
}

server {
    listen 80;
    server_name panel.domainanda.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name panel.domainanda.com;

    ssl_certificate /usr/local/etc/letsencrypt/live/panel.domainanda.com/fullchain.pem;
    ssl_certificate_key /usr/local/etc/letsencrypt/live/panel.domainanda.com/privkey.pem;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    location / {
        proxy_pass http://bsdpanel_backend;
        proxy_http_version 1.1;

        # Mendukung WebSocket (Terminal & Live Logs)
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Langkah Aktivasi di FreeBSD:
1. Pastikan Nginx terpasang dan aktif:
   ```sh
   pkg install -y nginx doas
   sysrc nginx_enable="YES"
   ```
2. Pastikan direktori `conf.d` disertakan dalam `/usr/local/etc/nginx/nginx.conf`:
   ```nginx
   http {
       ...
       include /usr/local/etc/nginx/conf.d/*.conf;
   }
   ```
3. Uji sintaks dan reload Nginx:
   ```sh
   nginx -t
   service nginx reload
   ```

---

## 5. Menjalankan BSD Panel sebagai Service FreeBSD (`rc.d`)

1. Salin script rc:
   ```sh
   cp deploy/rc.d/bsdpanel /usr/local/etc/rc.d/bsdpanel
   chmod +x /usr/local/etc/rc.d/bsdpanel
   ```
2. Tambahkan user sistem khusus untuk panel:
   ```sh
   pw useradd bsdpanel -m -d /var/db/bsdpanel -s /usr/sbin/nologin -c "BSD Panel Daemon User"
   ```
3. Aktifkan dan jalankan:
   ```sh
   sysrc bsdpanel_enable="YES"
   service bsdpanel start
   service bsdpanel status
   ```

---

## 6. Ringkasan Fitur Unggulan

| Komponen | Implementasi FreeBSD | Keunggulan |
| :--- | :--- | :--- |
| **Backend** | Go (Golang) murni | Penggunaan RAM sangat rendah (< 30 MB), binary mandiri tanpa dependency runtime |
| **User Isolation** | `pw`, `chmod 0750`, POSIX Credential | Isolasi per tenant, non-root daemons, proteksi antar situs |
| **Web Server** | Nginx, Apache 2.4, Caddy, OpenLiteSpeed | Fleksibel per domain virtual host |
| **PHP Engine** | PHP 8.2, 8.3, 8.4, 8.5 via FreeBSD Ports/Pkg | PHP-FPM pool mandiri per tenant (`unix:/var/run/php-fpm-<user>.sock`) |
| **Database** | MariaDB & PostgreSQL | Otomatisasi pembuatan database, user, dump SQL, phpMyAdmin/phpPgAdmin |
| **Firewall** | FreeBSD PF (`/etc/pf.conf`) & Fail2ban | Proteksi brute force, synflood mitigation, perlindungan port panel |
| **Frontend** | Tailwind CSS lokal, Fira Sans lokal, Lucide Icons | 100% offline, compact, responsif, dark mode, zero CDN dependency |
