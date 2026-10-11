#!/bin/sh
# ==============================================================================
# BSD Panel - Automated FreeBSD Installation Script
# Supported OS: FreeBSD 13.x, 14.x, 15-CURRENT
# ==============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}======================================================${NC}"
echo -e "${BLUE}   Installing BSD Panel for FreeBSD Server            ${NC}"
echo -e "${BLUE}======================================================${NC}"

if [ "$(id -u)" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root!${NC}"
    exit 1
fi

echo -e "${GREEN}[1/7] Updating FreeBSD pkg repositories...${NC}"
pkg update -f

echo -e "${GREEN}[2/7] Installing Core Dependencies & Compilers...${NC}"
pkg install -y rust cargo node npm git curl sudo fail2ban bash

echo -e "${GREEN}[3/7] Installing Multi Web Servers (Nginx, Apache, Caddy)...${NC}"
pkg install -y nginx apache24 caddy

echo -e "${GREEN}[4/7] Installing Multi PHP Versions (PHP 8.2, 8.3, 8.4)...${NC}"
pkg install -y \
    php82 php82-curl php82-mbstring php82-mysqli php82-pdo_pgsql php82-opcache \
    php83 php83-curl php83-mbstring php83-mysqli php83-pdo_pgsql php83-opcache \
    php84 php84-curl php84-mbstring php84-mysqli php84-pdo_pgsql php84-opcache || true

echo -e "${GREEN}[5/7] Installing Multi Databases (PostgreSQL 16 & MariaDB 10.11)...${NC}"
pkg install -y mariadb1011-server mariadb1011-client postgresql16-server postgresql16-client

echo -e "${GREEN}[6/7] Building BSD Panel Daemon and UI...${NC}"
mkdir -p /usr/local/etc/bsdpanel
mkdir -p /var/log/bsdpanel
mkdir -p /usr/local/www

# Copy files & build
cd "$(dirname "$0")/.."
cargo build --release
cp target/release/bsdpaneld /usr/local/sbin/bsdpaneld
chmod +x /usr/local/sbin/bsdpaneld

# Build UI
cd frontend
npm install
npm run build
mkdir -p /usr/local/etc/bsdpanel/ui
cp -r dist/* /usr/local/etc/bsdpanel/ui/

# Install rc.d script
cp ../scripts/rc.d/bsdpanel /usr/local/etc/rc.d/bsdpanel
chmod +x /usr/local/etc/rc.d/bsdpanel

echo -e "${GREEN}[7/7] Enabling Services in /etc/rc.conf...${NC}"
sysrc bsdpanel_enable="YES"
sysrc pf_enable="YES"
sysrc fail2ban_enable="YES"

echo -e "${BLUE}======================================================${NC}"
echo -e "${GREEN}   BSD Panel installation finished successfully!      ${NC}"
echo -e "${BLUE}   Start the panel with: service bsdpanel start       ${NC}"
echo -e "${BLUE}   Access Web UI: http://YOUR_SERVER_IP:8888          ${NC}"
echo -e "${BLUE}======================================================${NC}"
