#!/bin/sh
# ==============================================================================
# BSDPanel One-Line Installer for FreeBSD
# Repository: https://github.com/agus-salim/bsdpanel.git
# ==============================================================================

set -e

# Format ANSI escape characters secara native untuk FreeBSD /bin/sh
RED=$(printf '\033[31m')
GREEN=$(printf '\033[32m')
CYAN=$(printf '\033[36m')
YELLOW=$(printf '\033[33m')
BOLD=$(printf '\033[1m')
NC=$(printf '\033[0m')

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"

printf "%s\n" "${CYAN}"
cat << 'EOF'
  ____  ____  ____  ____                  _ 
 | __ )/ ___||  _ \|  _ \ __ _ _ __   ___| |
 |  _ \___ \| | | | |_) / _` | '_ \ / _ \ |
 | |_) |___) | |_| |  __/ (_| | | | |  __/ |
 |____/|____/|____/|_|   \__,_|_| |_|\___|_|
   FreeBSD Lightweight Web Control Panel
EOF
printf "%s\n" "${NC}"

# 1. Validasi Root
if [ "$(id -u)" -ne 0 ]; then
    printf "%s\n" "${RED}[ERROR] Skrip installer harus dijalankan sebagai root / superuser!${NC}" >&2
    exit 1
fi

# 2. Validasi Sistem Operasi FreeBSD
OS=$(uname -s)
if [ "$OS" != "FreeBSD" ]; then
    printf "%s\n" "${RED}[ERROR] BSDPanel dirancang khusus untuk FreeBSD. Sistem terdeteksi: $OS${NC}" >&2
    exit 1
fi

ARCH=$(uname -m)
if [ "$ARCH" != "amd64" ]; then
    printf "%s\n" "${RED}[ERROR] Saat ini hanya arsitektur amd64 yang didukung. Sistem: $ARCH${NC}" >&2
    exit 1
fi

VERSION=$(uname -r)
printf "%s\n" "${GREEN}[INFO] Sistem terverifikasi: FreeBSD $VERSION ($ARCH)${NC}"

# 3. Instalasi Dependensi Dasar via pkg
printf "%s\n" "${YELLOW}[1/5] Memperbarui repo pkg dan memasang dependensi dasar...${NC}"
pkg install -y ca_root_nss curl nginx apache24 mariadb1011-server \
    php83 php83-pdo php83-curl php83-mysqli php83-pdo_mysql php83-pdo_pgsql php83-pgsql \
    php83-session php83-filter php83-ctype php83-tokenizer php83-dom php83-xml \
    php83-iconv php83-intl php83-fileinfo php83-mbstring php83-zip php83-gd \
    security/py-certbot security/py-certbot-nginx || \
pkg install -y py312-certbot py312-certbot-nginx || true

# 4. Buat Direktori Konfigurasi
printf "%s\n" "${YELLOW}[2/5] Menyiapkan struktur direktori...${NC}"
mkdir -p /usr/local/etc/bsdpanel
mkdir -p /usr/local/etc/nginx/conf.d
mkdir -p /usr/local/etc/php-fpm.d
mkdir -p /var/log/nginx
mkdir -p /home

chmod 750 /usr/local/etc/bsdpanel

# Pastikan nginx.conf memuat berkas konfigurasi virtual host dari conf.d dan batas upload 1024M
if [ -f "/usr/local/etc/nginx/nginx.conf" ]; then
    if ! grep -q "conf.d/\*\.conf" /usr/local/etc/nginx/nginx.conf; then
        sed -i '' 's|http {|http {\n    include /usr/local/etc/nginx/conf.d/*.conf;\n    client_max_body_size 1024M;|' /usr/local/etc/nginx/nginx.conf
    elif ! grep -q "client_max_body_size" /usr/local/etc/nginx/nginx.conf; then
        sed -i '' 's|http {|http {\n    client_max_body_size 1024M;|' /usr/local/etc/nginx/nginx.conf
    fi
fi

# Konfigurasi Apache 2.4 mendengarkan port 8081 agar tidak bentrok dengan Nginx port 80
if [ -f "/usr/local/etc/apache24/httpd.conf" ]; then
    sed -i '' 's/^Listen 80$/Listen 8081/' /usr/local/etc/apache24/httpd.conf
    if ! grep -q "ServerName" /usr/local/etc/apache24/httpd.conf; then
        echo "ServerName localhost:8081" >> /usr/local/etc/apache24/httpd.conf
    fi
fi

# Siapkan aturan firewall dasar /etc/pf.conf jika belum ada
if [ ! -f "/etc/pf.conf" ]; then
    cat << 'PFOUT' > /etc/pf.conf
# BSDPanel Default PF Configuration
set skip on lo0
pass in all
pass out all
PFOUT
fi

# 5. Pasang Binary & Skrip rc.d
printf "%s\n" "${YELLOW}[3/5] Memasang binary bsdpanel...${NC}"

# Helper download menggunakan curl atau FreeBSD fetch bawaan
download_file() {
    URL="$1"
    DEST="$2"
    if which curl >/dev/null 2>&1; then
        curl -sSL "$URL" -o "$DEST"
    elif which fetch >/dev/null 2>&1; then
        fetch -q -o "$DEST" "$URL"
    fi
}

# Cek binary lokal hasil kompilasi
if [ -f "./bin/freebsd_amd64/bsdpanel" ]; then
    printf "%s\n" "[INFO] Menggunakan binary lokal ./bin/freebsd_amd64/bsdpanel..."
    install -m 755 ./bin/freebsd_amd64/bsdpanel /usr/local/sbin/bsdpanel
elif [ -f "./cmd/bsdpanel/main.go" ]; then
    printf "%s\n" "[INFO] Mengompilasi bsdpanel langsung dari kode sumber lokal..."
    if ! which go >/dev/null 2>&1; then
        pkg install -y go
    fi
    go build -ldflags="-s -w" -o /usr/local/sbin/bsdpanel ./cmd/bsdpanel
else
    GITHUB_REPO="agus-salim/bsdpanel"
    RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/bsdpanel-freebsd-amd64.tar.gz"
    printf "%s\n" "[INFO] Mengunduh binary rilis dari GitHub..."
    TMP_DIR=$(mktemp -d)
    download_file "$RELEASE_URL" "$TMP_DIR/bsdpanel.tar.gz" || true
    if [ -s "$TMP_DIR/bsdpanel.tar.gz" ]; then
        tar -xzf "$TMP_DIR/bsdpanel.tar.gz" -C "$TMP_DIR"
        install -m 755 "$TMP_DIR/bsdpanel" /usr/local/sbin/bsdpanel
        rm -rf "$TMP_DIR"
    else
        printf "%s\n" "[INFO] Mengunduh kode sumber terbaru dari GitHub main branch..."
        SRC_URL="https://github.com/${GITHUB_REPO}/archive/refs/heads/main.tar.gz"
        download_file "$SRC_URL" "$TMP_DIR/src.tar.gz" || true
        if [ -s "$TMP_DIR/src.tar.gz" ]; then
            tar -xzf "$TMP_DIR/src.tar.gz" -C "$TMP_DIR"
            if ! which go >/dev/null 2>&1; then
                pkg install -y go
            fi
            (cd "$TMP_DIR/bsdpanel-main" && go build -ldflags="-s -w" -o /usr/local/sbin/bsdpanel ./cmd/bsdpanel)
            if [ ! -f "./scripts/bsdpanel.rc" ] && [ -f "$TMP_DIR/bsdpanel-main/scripts/bsdpanel.rc" ]; then
                install -m 555 "$TMP_DIR/bsdpanel-main/scripts/bsdpanel.rc" /usr/local/etc/rc.d/bsdpanel
            fi
            rm -rf "$TMP_DIR"
        else
            printf "%s\n" "${RED}[ERROR] Gagal mengunduh bsdpanel dari GitHub.${NC}"
            exit 1
        fi
    fi
fi

# Pasang script rc.d
if [ -f "./scripts/bsdpanel.rc" ]; then
    install -m 555 ./scripts/bsdpanel.rc /usr/local/etc/rc.d/bsdpanel
elif [ ! -f "/usr/local/etc/rc.d/bsdpanel" ]; then
    printf "%s\n" "[INFO] Mengunduh skrip rc.d bsdpanel..."
    download_file "https://raw.githubusercontent.com/agus-salim/bsdpanel/main/scripts/bsdpanel.rc" "/usr/local/etc/rc.d/bsdpanel"
    chmod 555 /usr/local/etc/rc.d/bsdpanel
fi

# 6. Konfigurasi rc.conf FreeBSD
printf "%s\n" "${YELLOW}[4/5] Mengonfigurasi layanan di /etc/rc.conf...${NC}"
sysrc bsdpanel_enable="YES"
sysrc nginx_enable="YES"
sysrc apache24_enable="YES"
sysrc php_fpm_enable="YES"
sysrc mysql_enable="YES"
sysrc postgresql_enable="YES"
sysrc pf_enable="YES"

# Inisialisasi cluster database PostgreSQL jika belum pernah dibuat
service postgresql initdb 2>/dev/null || true

# 7. Nyalakan Layanan
printf "%s\n" "${YELLOW}[5/5] Menjalankan layanan...${NC}"
service nginx start || service nginx restart
service apache24 start || service apache24 restart || true
service php_fpm start || service php_fpm restart
service mysql-server start || true
service postgresql start || true
service pf start || true

# Restart bsdpanel secara bersih
service bsdpanel stop 2>/dev/null || true
pkill -9 bsdpanel 2>/dev/null || true
sleep 1
service bsdpanel start

# Dapatkan IP Server
IP_ADDR=$(ifconfig | awk '/inet / && !/127.0.0.1/ {print $2; exit}')
if [ -z "$IP_ADDR" ]; then
    IP_ADDR="YOUR_SERVER_IP"
fi

printf "\n%s\n" "${GREEN}================================================================${NC}"
printf "%s\n" "${GREEN}  Instalasi BSDPanel Berhasil! 🚀${NC}"
printf "%s\n" "${GREEN}================================================================${NC}"
printf "URL Panel        : %s\n" "${CYAN}http://${IP_ADDR}:8080${NC}"
printf "Username Default : %s\n" "${YELLOW}pengelola${NC}"
printf "Password Default : %s\n" "${YELLOW}Suksesterus123#${NC}"
printf "Konfigurasi      : %s\n" "${YELLOW}/usr/local/etc/bsdpanel/config.json${NC}"
printf "Kontrol Layanan  : %s\n" "${YELLOW}service bsdpanel [start|stop|restart|status]${NC}"
printf "\n"
printf "Dokumentasi & Source: https://github.com/agus-salim/bsdpanel.git\n"
printf "%s\n" "${GREEN}================================================================${NC}"
