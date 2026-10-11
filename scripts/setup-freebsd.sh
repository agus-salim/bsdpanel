#!/bin/sh
# ============================================================
#  BSDPanel - FreeBSD Host Setup Script
#  Menyiapkan dependencies Multi-PHP, Web Server, DB & ZFS
# ============================================================

set -e

echo "==> Memperbarui pkg FreeBSD repository..."
pkg update -f

echo "==> Menginstal Web Server (Nginx & Apache 2.4)..."
pkg install -y nginx apache24

echo "==> Menginstal Multi-PHP (PHP 8.1, PHP 8.2, PHP 8.3) & Extensions..."
# PHP 8.1
pkg install -y php81 php81-mysqli php81-pdo_mysql php81-pgsql php81-curl php81-mbstring php81-gd php81-zip php81-zlib php81-opcache
# PHP 8.2
pkg install -y php82 php82-mysqli php82-pdo_mysql php82-pgsql php82-curl php82-mbstring php82-gd php82-zip php82-zlib php82-opcache
# PHP 8.3
pkg install -y php83 php83-mysqli php83-pdo_mysql php83-pgsql php83-curl php83-mbstring php83-gd php83-zip php83-zlib php83-opcache

echo "==> Menginstal Database Server (MariaDB / PostgreSQL)..."
pkg install -y mariadb1011-server postgresql16-server

echo "==> Mengaktifkan services di /etc/rc.conf..."
sysrc nginx_enable="YES"
sysrc php81_fpm_enable="YES"
sysrc php82_fpm_enable="YES"
sysrc php83_fpm_enable="YES"
sysrc mysql_enable="YES"
sysrc postgresql_enable="YES"

echo "==> Menyiapkan direktori BSDPanel..."
mkdir -p /usr/local/etc/bsdpanel
mkdir -p /var/db/bsdpanel
mkdir -p /usr/local/etc/nginx/conf.d
mkdir -p /usr/local/etc/apache24/Includes
mkdir -p /usr/local/etc/php-fpm.d

echo "==> Setup dataset ZFS (opsional jika root menggunakan ZFS)..."
if zfs list zroot >/dev/null 2>&1; then
    zfs create -o mountpoint=/usr/local/www zroot/bsdpanel || true
fi

echo "==> Instalasi dependensi FreeBSD selesai!"
