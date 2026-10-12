#!/bin/sh
# ==============================================================================
# BSDPanel One-Line Installer for FreeBSD
# Repository: https://github.com/agus-salim/bsdpanel.git
# ==============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "${CYAN}"
echo "  ____  ____  ____  ____                  _ "
echo " | __ )/ ___||  _ \|  _ \ __ _ _ __   ___| |"
echo " |  _ \\___ \| | | | |_) / _\` | '_ \ / _ \ |"
echo " | |_) |___) | |_| |  __/ (_| | | | |  __/ |"
echo " |____/|____/|____/|_|   \__,_|_| |_|\___|_|"
echo "   FreeBSD Lightweight Web Control Panel"
echo "${NC}"

# 1. Validasi Root
if [ "$(id -u)" -ne 0 ]; then
    echo "${RED}[ERROR] Skrip installer harus dijalankan sebagai root / superuser!${NC}" >&2
    exit 1
fi

# 2. Validasi Sistem Operasi FreeBSD
OS=$(uname -s)
if [ "$OS" != "FreeBSD" ]; then
    echo "${RED}[ERROR] BSDPanel dirancang khusus untuk FreeBSD. Sistem terdeteksi: $OS${NC}" >&2
    exit 1
fi

ARCH=$(uname -m)
if [ "$ARCH" != "amd64" ]; then
    echo "${RED}[ERROR] Saat ini hanya arsitektur amd64 yang didukung. Sistem: $ARCH${NC}" >&2
    exit 1
fi

VERSION=$(uname -r)
echo "${GREEN}[INFO] Sistem terverifikasi: FreeBSD $VERSION ($ARCH)${NC}"

# 3. Instalasi Dependensi Dasar via pkg
printf "%b\n" "${YELLOW}[1/5] Memperbarui repo pkg dan memasang dependensi dasar...${NC}"
pkg install -y ca_root_nss curl nginx mariadb1011-server \
    php83 php83-curl php83-mysqli php83-pdo_mysql php83-mbstring php83-zip php83-gd

# 4. Buat Direktori Konfigurasi
printf "%b\n" "${YELLOW}[2/5] Menyiapkan struktur direktori...${NC}"
mkdir -p /usr/local/etc/bsdpanel
mkdir -p /usr/local/etc/nginx/conf.d
mkdir -p /usr/local/etc/php-fpm.d
mkdir -p /var/log/nginx
mkdir -p /home

chmod 750 /usr/local/etc/bsdpanel

# 5. Pasang Binary & Skrip rc.d
printf "%b\n" "${YELLOW}[3/5] Memasang binary bsdpanel...${NC}"

# Cek binary lokal hasil kompilasi
if [ -f "./bin/freebsd_amd64/bsdpanel" ]; then
    printf "%s\n" "[INFO] Menggunakan binary lokal ./bin/freebsd_amd64/bsdpanel..."
    install -m 755 ./bin/freebsd_amd64/bsdpanel /usr/local/sbin/bsdpanel
elif [ -f "./cmd/bsdpanel/main.go" ]; then
    # Jika source code tersedia, kompilasi langsung di FreeBSD
    printf "%s\n" "[INFO] Mengompilasi bsdpanel langsung dari kode sumber..."
    if ! which go >/dev/null 2>&1; then
        pkg install -y go
    fi
    go build -ldflags="-s -w" -o /usr/local/sbin/bsdpanel ./cmd/bsdpanel
else
    # Coba download release asset
    GITHUB_REPO="agus-salim/bsdpanel"
    RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/bsdpanel-freebsd-amd64.tar.gz"
    printf "%s\n" "[INFO] Mengunduh binary rilis dari GitHub..."
    TMP_DIR=$(mktemp -d)
    curl -sSL "$RELEASE_URL" -o "$TMP_DIR/bsdpanel.tar.gz" || true
    if [ -f "$TMP_DIR/bsdpanel.tar.gz" ]; then
        tar -xzf "$TMP_DIR/bsdpanel.tar.gz" -C "$TMP_DIR"
        install -m 755 "$TMP_DIR/bsdpanel" /usr/local/sbin/bsdpanel
        rm -rf "$TMP_DIR"
    else
        printf "%b\n" "${RED}[ERROR] Binary tidak ditemukan. Silakan jalankan 'pkg install -y go && go build -o /usr/local/sbin/bsdpanel ./cmd/bsdpanel'${NC}"
        exit 1
    fi
fi

# Pasang script rc.d
install -m 555 ./scripts/bsdpanel.rc /usr/local/etc/rc.d/bsdpanel

# 6. Konfigurasi rc.conf FreeBSD
echo "${YELLOW}[4/5] Mengonfigurasi layanan di /etc/rc.conf...${NC}"
sysrc bsdpanel_enable="YES"
sysrc nginx_enable="YES"
sysrc php_fpm_enable="YES"
sysrc mysql_enable="YES"

# 7. Nyalakan Layanan
echo "${YELLOW}[5/5] Menjalankan layanan...${NC}"
service nginx start || service nginx restart
service php_fpm start || service php_fpm restart
service mysql-server start || true
service bsdpanel restart || service bsdpanel start

# Dapatkan IP Server
IP_ADDR=$(ifconfig | awk '/inet / && !/127.0.0.1/ {print $2; exit}')
if [ -z "$IP_ADDR" ]; then
    IP_ADDR="YOUR_SERVER_IP"
fi

echo ""
echo "${GREEN}================================================================${NC}"
echo "${GREEN}  Instalasi BSDPanel Berhasil! 🚀${NC}"
echo "${GREEN}================================================================${NC}"
echo "URL Panel      : ${CYAN}http://${IP_ADDR}:8080${NC}"
echo "Username Default: ${YELLOW}admin${NC}"
echo "Konfigurasi    : ${YELLOW}/usr/local/etc/bsdpanel/config.json${NC}"
echo "Kontrol Layanan: ${YELLOW}service bsdpanel [start|stop|restart|status]${NC}"
echo ""
echo "Dokumentasi & Source: https://github.com/agus-salim/bsdpanel.git"
echo "${GREEN}================================================================${NC}"
