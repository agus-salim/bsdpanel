Bertindaklah sebagai Senior Systems Architect dan Full-Stack Developer yang ahli dalam bahasa Go (Golang), manajemen sistem FreeBSD, dan arsitektur Control Panel hosting. 

Tolong buatkan rancangan arsitektur, struktur folder, dan kode awal untuk aplikasi **Lightweight VPS Control Panel** yang berjalan di sistem operasi **FreeBSD** dengan kriteria teknis sebagai berikut:

### A. Arsitektur & Teknologi Utama
1. **Backend:** Menggunakan **Go (Golang)** murni dengan arsitektur modular, ringan, aman, dan efisien dalam penggunaan resource.
2. **Frontend & UI/UX:** 
   - Tampilan compact, elegan, dan responsive menggunakan **Tailwind CSS** (dimuat secara lokal).
   - Tipografi menggunakan font **Fira Sans** (dimuat secara lokal).
   - Ikon menggunakan **Lucide Icons** (dimuat secara lokal/SVG offline).
3. **Model Eksekusi & Keamanan (Non-Root):**
   - Aplikasi utama panel berjalan di balik **Nginx sebagai reverse proxy** agar dapat diakses melalui domain khusus.
   - **Fitur Utama Keamanan Arsitektur:** Setiap layanan di dalam panel (seperti web server, database, firewall, dll.) serta setiap pembuatan situs/database baru akan dijalankan dan diisolasi pada **level user sistem (bukan root)**. Hal ini mempermudah instalasi, pengaturan, dan penghapusan tanpa memerlukan hak akses root penuh secara langsung oleh proses daemon.

### B. Fitur Utama yang Harus Dibangun
1. **Manajemen Situs & Domain:**
   - Pembuatan situs baru (otomatis membuat system user baru untuk setiap situs).
   - Integrasi SSL otomatis menggunakan Let's Encrypt.
   - Konfigurasi Reverse Proxy per situs.
   - Pemilihan Web Server (Nginx, Apache, LiteSpeed, atau Caddy) dan pemilihan versi PHP secara spesifik per situs.
   - Fitur pendukung: File Manager berbasis web, Web Terminal, dan Log Viewer.
2. **Installer & Manajemen Layanan (Service Manager):**
   - Fitur instalasi/uninstalasi aplikasi web server (Nginx, Apache, LiteSpeed, Caddy).
   - Fitur instalasi aplikasi database (PostgreSQL, MariaDB, phpMyAdmin, phpPgAdmin).
   - Installer multi-versi PHP (PHP 8.2, 8.3, 8.4, hingga 8.5) beserta ekstensi lengkapnya di FreeBSD.
3. **Manajemen Database & User:**
   - Pengaturan database (buat, hapus, import, export SQL dump langsung dari panel atau melalui phpMyAdmin/phpPgAdmin).
   - Manajemen User dan Reseller panel dengan hak akses bertingkat.
4. **Keamanan & Panel Domain:**
   - Pengaturan Firewall (native FreeBSD `pf` / `ipfw`) dan integrasi Fail2ban untuk pencegahan brute-force.
   - Pengaturan domain akses panel (custom domain untuk panel control).

### C. Output yang Diharapkan
1. **Struktur Direktori Proyek:** Penjelasan direktori backend Go dan frontend aset lokal.
2. **Contoh Kode Backend (Golang):** Implementasi dasar server HTTP Go dan fungsi utilitas untuk mengeksekusi perintah shell FreeBSD secara aman dengan user terisolasi.
3. **Contoh Tampilan (HTML/Tailwind):** Snippet kode halaman dashboard utama yang compact, responsif, menggunakan Tailwind CSS, font Fira Sans, dan Lucide Icons secara lokal.
4. **Panduan Konfigurasi:** Langkah singkat konfigurasi Nginx Reverse Proxy di FreeBSD untuk mengarahkan domain panel ke aplikasi Go.