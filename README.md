# BSD Panel — Control Panel Ringan & Powerfull untuk FreeBSD

Control panel hosting modern, cepat, dan aman yang dirancang khusus untuk FreeBSD.

## ✨ Fitur Utama
- **Backend Rust (`bsdpaneld`)**: Performa ultra cepat, aman memori, menggunakan Axum & Tokio, mematuhi standar keamanan ketat FreeBSD (rc.d, sysrc, pfctl, fail2ban).
- **Frontend Vue 3**: Tampilan simpel, responsif, minimalis, dan elegan menggunakan font **Fira Sans** serta integrasi pustaka ikon ganda (**Lucide Icons** + **@iconify/vue**) untuk menjamin 100% ketersediaan ikon.
- **Multi Web Server**: Dukungan switchboard 4 web server sekaligus (NGINX, Apache HTTPD 2.4, OpenLiteSpeed, dan Caddy) lengkap dengan generator konfigurasi virtual host otomatis dan live preview.
- **Multi PHP Version (8.2 - 8.6)**: Socket pool PHP-FPM terisolasi per versi, penyesuaian parameter worker FPM, serta editor directive `php.ini` dan saklar on/off ekstensi modular.
- **Multi Database**: Manajemen **PostgreSQL 16** dan **MariaDB 10.11 LTS** dengan pembuatan database, pengaturan hak akses pengguna basis data, dan port status.
- **Firewall & Fail2ban**: Integrasi penuh **FreeBSD Packet Filter (PF)** dengan tabel dinamis `<bruteforce>` dan integrasi jail Fail2ban untuk pemblokiran/pembukaan blokir IP secara instan.
- **User & Reseller Tier**: Struktur berjenjang (SuperAdmin/Root, Reseller, dan End User) dengan alokasi kuota disk ZFS/UFS, bandwidth bulanan, dan batas domain/database.

## 🚀 Panduan Cepat
Lihat panduan lengkap di [FREEBSD_GUIDE.md](file:///FREEBSD_GUIDE.md).

```bash
# Jalankan installer otomatis di server FreeBSD Anda:
chmod +x scripts/install-freebsd.sh
./scripts/install-freebsd.sh
```
