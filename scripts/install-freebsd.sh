#!/bin/sh
# ==============================================================================
# BSD Panel - Automated FreeBSD Installation Script
# Supported OS: FreeBSD 13.x, 14.x, 15-CURRENT
# ==============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
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
# In FreeBSD, package 'rust' includes cargo.
pkg install -y rust git curl sudo bash || pkg install -y rust git curl bash

# Install Node & NPM (handles FreeBSD naming variations: node20/node22/node)
echo -e "${YELLOW}Installing Node.js & NPM...${NC}"
pkg install -y node20 npm-node20 || pkg install -y node22 npm-node22 || pkg install -y node npm || true

# Install Fail2ban (in FreeBSD named py311-fail2ban / py310-fail2ban / fail2ban)
echo -e "${YELLOW}Installing Fail2ban...${NC}"
pkg install -y py311-fail2ban || pkg install -y py310-fail2ban || pkg install -y py39-fail2ban || pkg install -y fail2ban || true

echo -e "${GREEN}[3/7] Installing Multi Web Servers (Nginx, Apache, Caddy)...${NC}"
pkg install -y nginx apache24 caddy || true

echo -e "${GREEN}[4/7] Installing Multi PHP Versions (PHP 8.2, 8.3, 8.4)...${NC}"
pkg install -y php82 php82-curl php82-mbstring php82-mysqli php82-pdo_pgsql php82-opcache || true
pkg install -y php83 php83-curl php83-mbstring php83-mysqli php83-pdo_pgsql php83-opcache || true
pkg install -y php84 php84-curl php84-mbstring php84-mysqli php84-pdo_pgsql php84-opcache || true

echo -e "${GREEN}[5/7] Installing Multi Databases (PostgreSQL & MariaDB)...${NC}"
pkg install -y mariadb1011-server mariadb1011-client || pkg install -y mariadb106-server mariadb106-client || true
pkg install -y postgresql16-server postgresql16-client || pkg install -y postgresql15-server postgresql15-client || true

echo -e "${GREEN}[6/7] Building BSD Panel Daemon and UI...${NC}"
mkdir -p /usr/local/etc/bsdpanel
mkdir -p /var/log/bsdpanel
mkdir -p /usr/local/www

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# Build Rust backend (inside backend/ directory)
if command -v cargo >/dev/null 2>&1; then
    echo -e "${YELLOW}Compiling Rust backend (bsdpaneld)...${NC}"
    cd "$BASE_DIR/backend"
    cargo build --release
    if [ -f "target/release/bsdpaneld" ]; then
        cp target/release/bsdpaneld /usr/local/sbin/bsdpaneld
    elif [ -f "$BASE_DIR/target/release/bsdpaneld" ]; then
        cp "$BASE_DIR/target/release/bsdpaneld" /usr/local/sbin/bsdpaneld
    fi
    chmod +x /usr/local/sbin/bsdpaneld
else
    echo -e "${RED}Cargo not found! Make sure package 'rust' is installed.${NC}"
    exit 1
fi

# Build Vue 3 UI
cd "$BASE_DIR/frontend"
if command -v npm >/dev/null 2>&1; then
    echo -e "${YELLOW}Building Vue 3 Frontend...${NC}"
    npm install
    npm run build
    mkdir -p /usr/local/etc/bsdpanel/ui
    cp -r dist/* /usr/local/etc/bsdpanel/ui/
else
    echo -e "${YELLOW}NPM not found. Creating placeholder UI directory...${NC}"
    mkdir -p /usr/local/etc/bsdpanel/ui
fi

# Install rc.d script
cd "$BASE_DIR"
cp scripts/rc.d/bsdpanel /usr/local/etc/rc.d/bsdpanel
chmod +x /usr/local/etc/rc.d/bsdpanel

echo -e "${GREEN}[7/7] Enabling and Starting Services in /etc/rc.conf...${NC}"
sysrc bsdpanel_enable="YES"

# Start or restart bsdpanel daemon
echo -e "${YELLOW}Starting BSD Panel service...${NC}"
service bsdpanel restart || service bsdpanel start

echo -e "${BLUE}======================================================${NC}"
echo -e "${GREEN}   BSD Panel installation finished successfully!      ${NC}"
echo -e "${BLUE}   Status: service bsdpanel status                    ${NC}"
echo -e "${BLUE}   Log:    tail -f /var/log/bsdpanel/daemon.log       ${NC}"
echo -e "${BLUE}   Web UI: http://103.150.92.38:8888                  ${NC}"
echo -e "${BLUE}======================================================${NC}"
