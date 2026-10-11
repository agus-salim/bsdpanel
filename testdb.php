<?php
/**
 * ============================================================================
 * BSD Panel - Database Diagnostic & Connection Test Tool (testdb.php)
 * ============================================================================
 * Tool mandiri untuk mendiagnosis:
 * 1. Ketersediaan driver PHP (pdo_mysql, mysqli, pdo_pgsql, pgsql)
 * 2. Status socket & port layanan database (MariaDB/MySQL & PostgreSQL)
 * 3. Analisis kesalahan koneksi:
 *    - Driver belum terpasang
 *    - Layanan database mati / port tertutup
 *    - Username / password salah (Error 1045 Access Denied)
 *    - Nama database tidak ditemukan (Error 1049 Unknown Database)
 *    - Host / socket mismatch (localhost vs 127.0.0.1)
 * 4. Contoh kode koneksi PHP siap pakai (PDO & MySQLi)
 *
 * PERINGATAN KEAMANAN: Hapus file ini setelah selesai melakukan pengujian!
 * ============================================================================
 */

// Error reporting untuk diagnosa maksimal
error_reporting(E_ALL);
ini_set('display_errors', '1');

$isCli = (php_sapi_name() === 'cli');

// Deteksi info PHP & OS
$phpVersion = PHP_VERSION;
$phpMajorMinor = PHP_MAJOR_VERSION . PHP_MINOR_VERSION;
$phpOs = PHP_OS;
$sapiName = php_sapi_name();
$loadedIni = php_ini_loaded_file() ?: '(None)';
$scannedInis = php_ini_scanned_files() ?: '';

// Cek Driver / Extension
$drivers = [
    'pdo'        => class_exists('PDO'),
    'pdo_mysql'  => extension_loaded('pdo_mysql') && class_exists('PDO') && in_array('mysql', PDO::getAvailableDrivers()),
    'mysqli'     => extension_loaded('mysqli') && function_exists('mysqli_connect'),
    'pdo_pgsql'  => extension_loaded('pdo_pgsql') && class_exists('PDO') && in_array('pgsql', PDO::getAvailableDrivers()),
    'pgsql'      => extension_loaded('pgsql') && function_exists('pg_connect'),
    'pdo_sqlite' => extension_loaded('pdo_sqlite') && class_exists('PDO') && in_array('sqlite', PDO::getAvailableDrivers()),
];

// Cek Lokasi Socket MariaDB/MySQL di FreeBSD & Linux
$socketCandidates = [
    '/var/run/mysql/mysql.sock',
    '/tmp/mysql.sock',
    '/var/lib/mysql/mysql.sock',
    '/run/mysqld/mysqld.sock',
];
$foundMysqlSockets = [];
foreach ($socketCandidates as $sock) {
    if (file_exists($sock)) {
        $foundMysqlSockets[] = $sock;
    }
}

// Cek Port Database TCP
function checkTcpPort($host, $port, $timeout = 1.0) {
    $fp = @fsockopen($host, $port, $errno, $errstr, $timeout);
    if ($fp) {
        fclose($fp);
        return true;
    }
    return false;
}

$mysqlPortOpen = checkTcpPort('127.0.0.1', 3306, 0.5);
$pgsqlPortOpen = checkTcpPort('127.0.0.1', 5432, 0.5);

// Input Form / Parameter Pengujian
$submitted = false;
$engine    = $_REQUEST['engine'] ?? 'mysql'; // mysql atau pgsql
$host      = trim($_REQUEST['host'] ?? '127.0.0.1');
$port      = intval($_REQUEST['port'] ?? ($engine === 'pgsql' ? 5432 : 3306));
$database  = trim($_REQUEST['database'] ?? '');
$username  = trim($_REQUEST['username'] ?? '');
$password  = $_REQUEST['password'] ?? '';
$socket    = trim($_REQUEST['socket'] ?? '');

$diagnostics = [];
$testResults = null;

if ($_SERVER['REQUEST_METHOD'] === 'POST' || (isset($_GET['test']) && $_GET['test'] === '1')) {
    $submitted = true;
    $testResults = runDatabaseTest($engine, $host, $port, $database, $username, $password, $socket, $drivers);
}

function runDatabaseTest($engine, $host, $port, $database, $username, $password, $socket, $drivers) {
    $res = [
        'success'       => false,
        'driver_ok'     => false,
        'network_ok'    => false,
        'auth_ok'       => false,
        'db_ok'         => false,
        'query_ok'      => false,
        'messages'      => [],
        'pdo_error'     => null,
        'native_error'  => null,
        'tables'        => [],
        'server_info'   => '',
        'diagnosis'     => '',
        'recommendation'=> '',
    ];

    if ($engine === 'mysql') {
        // 1. Cek Driver MySQL
        if (!$drivers['pdo_mysql'] && !$drivers['mysqli']) {
            $res['diagnosis'] = 'DRIVER_MISSING';
            $res['messages'][] = '❌ Ekstensi PHP MySQL (pdo_mysql dan mysqli) BELUM TERINSTALL pada sistem PHP ini!';
            $res['recommendation'] = "Jalankan perintah berikut di FreeBSD Terminal:\n" .
                "doas pkg install -y php" . PHP_MAJOR_VERSION . PHP_MINOR_VERSION . "-mysqli php" . PHP_MAJOR_VERSION . PHP_MINOR_VERSION . "-pdo_mysql\n" .
                "doas service php-fpm restart (atau doas service apache24 restart)";
            return $res;
        }
        $res['driver_ok'] = true;
        $res['messages'][] = '✅ Driver PHP MySQL (' . ($drivers['pdo_mysql'] ? 'PDO_MySQL ' : '') . ($drivers['mysqli'] ? 'MySQLi' : '') . ') tersedia.';

        // 2. Cek Network / Port TCP jika host adalah IP / hostname
        if ($host !== 'localhost' && empty($socket)) {
            $tcpOk = checkTcpPort($host, $port, 1.5);
            if ($tcpOk) {
                $res['network_ok'] = true;
                $res['messages'][] = "✅ Port database {$port} di {$host} berhasil dijangkau via TCP.";
            } else {
                $res['messages'][] = "⚠️ Port TCP {$port} di {$host} tidak merespons. Layanan database mungkin mati atau diblokir firewall.";
            }
        } else {
            $res['network_ok'] = true;
            $res['messages'][] = "ℹ️ Koneksi menggunakan UNIX domain socket lokal ('{$host}').";
        }

        // 3. Tes Kredensial via PDO (jika pdo_mysql aktif)
        if ($drivers['pdo_mysql']) {
            // Coba konek tanpa database terlebih dahulu untuk memisahkan error kredensial vs error nama database
            $dsnBase = "mysql:host={$host};port={$port};charset=utf8mb4";
            if (!empty($socket)) {
                $dsnBase = "mysql:unix_socket={$socket};charset=utf8mb4";
            }

            try {
                $pdoCheck = new PDO($dsnBase, $username, $password, [
                    PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
                    PDO::ATTR_TIMEOUT => 3,
                ]);
                $res['auth_ok'] = true;
                $res['server_info'] = $pdoCheck->getAttribute(PDO::ATTR_SERVER_VERSION);
                $res['messages'][] = "✅ Kredensial User ('{$username}') & Password BENAR! Berhasil login ke MariaDB/MySQL (v{$res['server_info']}).";

                // Jika user mengisi nama database, uji pilih database
                if (!empty($database)) {
                    $dsnWithDb = $dsnBase . ";dbname={$database}";
                    try {
                        $pdoDb = new PDO($dsnWithDb, $username, $password, [
                            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
                            PDO::ATTR_TIMEOUT => 3,
                        ]);
                        $res['db_ok'] = true;
                        $res['messages'][] = "✅ Database '{$database}' ditemukan dan berhasil diakses!";

                        // Uji Query & Hak Akses
                        $stmt = $pdoDb->query("SHOW TABLES");
                        $tables = $stmt->fetchAll(PDO::FETCH_COLUMN);
                        $res['tables'] = $tables;
                        $res['query_ok'] = true;
                        $res['messages'][] = "✅ Hak akses query valid (Ditemukan " . count($tables) . " tabel).";
                        $res['success'] = true;
                        $res['diagnosis'] = 'SUCCESS';
                        $res['recommendation'] = "Koneksi sempurna! Database dan user siap digunakan oleh aplikasi PHP Anda.";
                    } catch (PDOException $eDb) {
                        $res['pdo_error'] = $eDb->getMessage();
                        if ($eDb->getCode() == 1049 || strpos($eDb->getMessage(), 'Unknown database') !== false) {
                            $res['diagnosis'] = 'UNKNOWN_DATABASE';
                            $res['messages'][] = "❌ User & Password BENAR, namun Database '{$database}' TIDAK DITEMUKAN (Error 1049 Unknown Database)!";
                            $res['recommendation'] = "Pastikan nama database '{$database}' sudah dibuat di menu Databases BSD Panel, atau periksa kesalahan ketik huruf besar/kecil.";
                        } else {
                            $res['diagnosis'] = 'DB_ERROR';
                            $res['messages'][] = "❌ Gagal mengakses database: " . $eDb->getMessage();
                            $res['recommendation'] = "Periksa hak akses user '{$username}' terhadap database '{$database}'. Di BSD Panel, coba hapus dan buat ulang database atau jalankan GRANT ALL PRIVILEGES.";
                        }
                    }
                } else {
                    $res['success'] = true;
                    $res['diagnosis'] = 'AUTH_ONLY_SUCCESS';
                    $res['messages'][] = "ℹ️ Tes kredensial login sukses, namun Anda belum mengisi nama database untuk diuji.";
                    $res['recommendation'] = "Masukkan nama database di form pengujian untuk memeriksa keberadaan database.";
                }
            } catch (PDOException $e) {
                $res['pdo_error'] = $e->getMessage();
                $errCode = $e->getCode();
                $errMsg = $e->getMessage();

                if ($errCode == 1045 || strpos($errMsg, 'Access denied for user') !== false) {
                    $res['diagnosis'] = 'AUTH_FAILED';
                    $res['messages'][] = "❌ User atau Password SALAH (Error 1045: Access denied for user '{$username}')!";
                    $res['recommendation'] = "1. Pastikan username dan password database sama persis dengan yang dibuat di BSD Panel.\n" .
                        "2. Coba ganti host dari 'localhost' menjadi '127.0.0.1' atau sebaliknya.\n" .
                        "3. Anda dapat mereset password database melalui menu 'Databases' di BSD Panel.";
                } elseif ($errCode == 2002 || strpos($errMsg, 'Connection refused') !== false || strpos($errMsg, 'No such file or directory') !== false) {
                    $res['diagnosis'] = 'SOCKET_OR_SERVICE_DOWN';
                    $res['messages'][] = "❌ Gagal terhubung ke server database (Error 2002: Connection refused / Socket error)!";
                    $res['recommendation'] = "1. Pastikan layanan MariaDB aktif: doas service mysql-server status\n" .
                        "2. Jika service mati, jalankan: doas service mysql-server start\n" .
                        "3. Jika menggunakan 'localhost', PHP mungkin mencari socket di /tmp/mysql.sock padahal FreeBSD meletakkannya di /var/run/mysql/mysql.sock. Coba gunakan Host '127.0.0.1' di konfigurasi Anda.";
                } else {
                    $res['diagnosis'] = 'PDO_ERROR';
                    $res['messages'][] = "❌ Terjadi kesalahan koneksi PDO: " . $errMsg;
                    $res['recommendation'] = "Periksa detail error di atas dan sesuaikan konfigurasi host/port.";
                }
            }
        } elseif ($drivers['mysqli']) {
            // Fallback via MySQLi jika PDO tidak ada
            $conn = @mysqli_connect($host, $username, $password, $database ? $database : null, $port, $socket ? $socket : null);
            if ($conn) {
                $res['auth_ok'] = true;
                $res['db_ok'] = !empty($database);
                $res['success'] = true;
                $res['server_info'] = mysqli_get_server_info($conn);
                $res['messages'][] = "✅ Berhasil terhubung via MySQLi (v{$res['server_info']})!";
                $res['diagnosis'] = 'SUCCESS';
                $res['recommendation'] = "Koneksi MySQLi berhasil!";
                mysqli_close($conn);
            } else {
                $res['native_error'] = mysqli_connect_error();
                $errNo = mysqli_connect_errno();
                if ($errNo == 1045) {
                    $res['diagnosis'] = 'AUTH_FAILED';
                    $res['messages'][] = "❌ User atau Password SALAH (MySQLi Error 1045)!";
                    $res['recommendation'] = "Periksa username & password atau reset via menu Databases BSD Panel.";
                } elseif ($errNo == 1049) {
                    $res['diagnosis'] = 'UNKNOWN_DATABASE';
                    $res['messages'][] = "❌ Database '{$database}' tidak ditemukan (MySQLi Error 1049)!";
                    $res['recommendation'] = "Pastikan nama database sudah dibuat di BSD Panel.";
                } else {
                    $res['diagnosis'] = 'MYSQLI_ERROR';
                    $res['messages'][] = "❌ Kesalahan MySQLi: (" . $errNo . ") " . mysqli_connect_error();
                }
            }
        }
    } elseif ($engine === 'pgsql') {
        // PostgreSQL Test
        if (!$drivers['pdo_pgsql'] && !$drivers['pgsql']) {
            $res['diagnosis'] = 'DRIVER_MISSING';
            $res['messages'][] = '❌ Ekstensi PHP PostgreSQL (pdo_pgsql dan pgsql) BELUM TERINSTALL!';
            $res['recommendation'] = "Jalankan perintah berikut di FreeBSD Terminal:\n" .
                "doas pkg install -y php" . PHP_MAJOR_VERSION . PHP_MINOR_VERSION . "-pgsql php" . PHP_MAJOR_VERSION . PHP_MINOR_VERSION . "-pdo_pgsql\n" .
                "doas service php-fpm restart";
            return $res;
        }
        $res['driver_ok'] = true;

        if ($drivers['pdo_pgsql']) {
            $pgDb = $database ? $database : 'postgres';
            $dsn = "pgsql:host={$host};port={$port};dbname={$pgDb}";
            try {
                $pdoPg = new PDO($dsn, $username, $password, [
                    PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
                    PDO::ATTR_TIMEOUT => 3,
                ]);
                $res['auth_ok'] = true;
                $res['db_ok'] = true;
                $res['success'] = true;
                $res['server_info'] = $pdoPg->getAttribute(PDO::ATTR_SERVER_VERSION);
                $res['messages'][] = "✅ Berhasil terhubung ke PostgreSQL 16 (v{$res['server_info']})!";
                $res['diagnosis'] = 'SUCCESS';
                $res['recommendation'] = "Koneksi PostgreSQL berhasil!";
            } catch (PDOException $ePg) {
                $res['pdo_error'] = $ePg->getMessage();
                $res['messages'][] = "❌ Gagal terhubung ke PostgreSQL: " . $ePg->getMessage();
                $res['recommendation'] = "Pastikan user PostgreSQL dan database sudah dibuat, serta pastikan layanan PostgreSQL berjalan (doas service postgresql status).";
            }
        }
    }

    return $res;
}

// Jika dijalankan via CLI (Terminal)
if ($isCli) {
    echo "========================================================\n";
    echo " BSD Panel - Database Diagnostic Tool (CLI Mode)\n";
    echo "========================================================\n";
    echo "PHP Version  : {$phpVersion} ({$sapiName})\n";
    echo "PHP INI      : {$loadedIni}\n";
    echo "PDO MySQL    : " . ($drivers['pdo_mysql'] ? 'OK (Installed)' : 'MISSING (Belum terinstall)') . "\n";
    echo "MySQLi       : " . ($drivers['mysqli'] ? 'OK (Installed)' : 'MISSING (Belum terinstall)') . "\n";
    echo "PDO PgSQL    : " . ($drivers['pdo_pgsql'] ? 'OK (Installed)' : 'MISSING (Belum terinstall)') . "\n";
    echo "MySQL Port   : " . ($mysqlPortOpen ? '3306 Listening (OPEN)' : '3306 Closed / Service Down') . "\n";
    echo "PgSQL Port   : " . ($pgsqlPortOpen ? '5432 Listening (OPEN)' : '5432 Closed') . "\n";
    if (!empty($foundMysqlSockets)) {
        echo "Found Sockets: " . implode(', ', $foundMysqlSockets) . "\n";
    }
    echo "========================================================\n";
    if ($testResults) {
        echo "TEST RESULT:\n";
        foreach ($testResults['messages'] as $m) echo "{$m}\n";
        if ($testResults['recommendation']) {
            echo "\nSARAN / SOLUSI:\n" . $testResults['recommendation'] . "\n";
        }
    } else {
        echo "Gunakan browser untuk membuka http://domain-anda/testdb.php untuk uji interaktif lengkap.\n";
    }
    exit(0);
}
?>
<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Database Connection & Driver Diagnostic | BSD Panel</title>
  <style>
    :root {
      --bg: #080c14;
      --card-bg: #0d121f;
      --card-border: #1e293b;
      --accent: #6366f1;
      --accent-hover: #4f46e5;
      --text: #f1f5f9;
      --text-muted: #94a3b8;
      --success: #10b981;
      --warning: #f59e0b;
      --danger: #ef4444;
      --info: #0ea5e9;
      --font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: var(--bg);
      color: var(--text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
      line-height: 1.5;
      padding: 24px 16px;
      font-size: 14px;
    }
    .container {
      max-width: 960px;
      margin: 0 auto;
    }
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 20px;
      padding-bottom: 16px;
      border-bottom: 1px solid var(--card-border);
      flex-wrap: wrap;
      gap: 12px;
    }
    .title-area h1 {
      font-size: 20px;
      font-weight: 700;
      color: #fff;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .title-area p {
      color: var(--text-muted);
      font-size: 12px;
      margin-top: 2px;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 3px 8px;
      border-radius: 6px;
      font-size: 11px;
      font-weight: 600;
      font-family: var(--font-mono);
    }
    .badge-success { background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); }
    .badge-danger  { background: rgba(239, 68, 68, 0.15); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.3); }
    .badge-warning { background: rgba(245, 158, 11, 0.15); color: #fbbf24; border: 1px solid rgba(245, 158, 11, 0.3); }
    .badge-info    { background: rgba(14, 165, 233, 0.15); color: #38bdf8; border: 1px solid rgba(14, 165, 233, 0.3); }

    .security-alert {
      background: rgba(245, 158, 11, 0.1);
      border: 1px solid rgba(245, 158, 11, 0.3);
      border-radius: 10px;
      padding: 12px 16px;
      margin-bottom: 20px;
      color: #fde68a;
      font-size: 12px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 12px;
      padding: 20px;
      margin-bottom: 20px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.3);
    }
    .card-title {
      font-size: 15px;
      font-weight: 700;
      margin-bottom: 14px;
      color: #fff;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .grid-2 { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 16px; }
    .grid-3 { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 12px; }

    .info-item {
      background: #080c14;
      border: 1px solid #1e293b;
      border-radius: 8px;
      padding: 10px 14px;
    }
    .info-label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; }
    .info-val { font-size: 13px; font-weight: 600; color: #fff; margin-top: 3px; font-family: var(--font-mono); }

    .form-group { margin-bottom: 14px; }
    .form-label { display: block; font-size: 12px; font-weight: 600; color: #cbd5e1; margin-bottom: 5px; }
    .form-input, .form-select {
      width: 100%;
      background: #080c14;
      border: 1px solid #1e293b;
      color: #fff;
      padding: 9px 12px;
      border-radius: 8px;
      font-size: 13px;
      font-family: var(--font-mono);
      outline: none;
      transition: border-color 0.2s;
    }
    .form-input:focus, .form-select:focus { border-color: var(--accent); }
    .btn {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
      background: var(--accent);
      color: #fff;
      border: none;
      border-radius: 8px;
      padding: 10px 18px;
      font-size: 13px;
      font-weight: 600;
      cursor: pointer;
      transition: background 0.2s;
    }
    .btn:hover { background: var(--accent-hover); }
    .btn-secondary { background: #1e293b; color: #cbd5e1; }
    .btn-secondary:hover { background: #334155; }

    .verdict-box {
      border-radius: 10px;
      padding: 16px;
      margin-bottom: 16px;
    }
    .verdict-success { background: rgba(16, 185, 129, 0.12); border: 1px solid rgba(16, 185, 129, 0.4); color: #a7f3d0; }
    .verdict-error   { background: rgba(239, 68, 68, 0.12); border: 1px solid rgba(239, 68, 68, 0.4); color: #fecaca; }
    .verdict-warning { background: rgba(245, 158, 11, 0.12); border: 1px solid rgba(245, 158, 11, 0.4); color: #fde68a; }

    .code-block {
      background: #05080f;
      border: 1px solid #1e293b;
      border-radius: 8px;
      padding: 12px;
      font-family: var(--font-mono);
      font-size: 12px;
      color: #e2e8f0;
      overflow-x: auto;
      white-space: pre;
      line-height: 1.6;
      margin-top: 8px;
      position: relative;
    }
    .copy-btn {
      position: absolute;
      top: 8px;
      right: 8px;
      background: #1e293b;
      border: 1px solid #334155;
      color: #cbd5e1;
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 10px;
      cursor: pointer;
    }
    .copy-btn:hover { background: #334155; color: #fff; }

    .step-log {
      list-style: none;
      margin-top: 10px;
    }
    .step-log li {
      padding: 6px 0;
      border-bottom: 1px solid rgba(255,255,255,0.05);
      font-family: var(--font-mono);
      font-size: 12px;
    }
    .quick-preset {
      display: inline-block;
      padding: 4px 8px;
      background: #1e293b;
      border: 1px solid #334155;
      color: #38bdf8;
      border-radius: 4px;
      font-size: 11px;
      cursor: pointer;
      margin-right: 6px;
      margin-bottom: 6px;
    }
    .quick-preset:hover { background: #334155; }
  </style>
</head>
<body>

<div class="container">
  <!-- Header -->
  <div class="header">
    <div class="title-area">
      <h1>
        <span>⚡ BSD Panel Database Diagnostic</span>
      </h1>
      <p>Pemeriksa Driver PHP, Socket Server, Kredensial Database, dan Hak Akses</p>
    </div>
    <div style="display: flex; gap: 8px; align-items: center;">
      <span class="badge badge-info">PHP <?php echo htmlspecialchars($phpVersion); ?></span>
      <span class="badge <?php echo ($phpOs === 'FreeBSD' ? 'badge-success' : 'badge-info'); ?>"><?php echo htmlspecialchars($phpOs); ?></span>
      <span class="badge badge-warning"><?php echo htmlspecialchars($sapiName); ?></span>
    </div>
  </div>

  <!-- Security Alert Banner -->
  <div class="security-alert">
    <div>
      <strong>⚠️ Peringatan Keamanan:</strong> File diagnosa ini dapat menjalankan koneksi database. Segera hapus file <code>testdb.php</code> ini dari direktori web setelah Anda selesai melakukan pengujian!
    </div>
    <button onclick="if(confirm('Hapus file testdb.php ini sekarang?')) { window.location.href='?delete=1'; }" class="btn btn-secondary" style="font-size: 11px; padding: 4px 10px;">
      Hapus File Ini
    </button>
  </div>

  <?php
  // Handler untuk menghapus file sendiri
  if (isset($_GET['delete']) && $_GET['delete'] === '1') {
      if (@unlink(__FILE__)) {
          echo "<div class='card' style='text-align: center; color: #34d399;'><h3>✅ File testdb.php berhasil dihapus dari server!</h3><p>Halaman ini tidak lagi dapat diakses.</p></div>";
          exit;
      } else {
          echo "<div class='card' style='color: #f87171;'><h3>❌ Gagal menghapus file otomatis.</h3><p>Silakan hapus file testdb.php secara manual melalui File Manager BSD Panel.</p></div>";
      }
  }
  ?>

  <!-- Card 1: Ringkasan Driver & Layanan Database -->
  <div class="card">
    <div class="card-title">
      <span>1. Status Driver PHP & Service Database</span>
    </div>
    <div class="grid-2">
      <!-- Sisi Driver PHP -->
      <div>
        <h4 style="font-size: 12px; color: var(--text-muted); text-transform: uppercase; margin-bottom: 8px;">Driver PHP Yang Terinstall:</h4>
        <div style="display: flex; flex-direction: column; gap: 8px;">
          <div class="info-item" style="display: flex; justify-content: space-between; align-items: center;">
            <div>
              <div class="info-val">PDO (PHP Data Objects)</div>
              <div class="info-label">Framework database standar PHP</div>
            </div>
            <span class="badge <?php echo $drivers['pdo'] ? 'badge-success' : 'badge-danger'; ?>">
              <?php echo $drivers['pdo'] ? 'Aktif' : 'Tidak Aktif'; ?>
            </span>
          </div>

          <div class="info-item" style="display: flex; justify-content: space-between; align-items: center;">
            <div>
              <div class="info-val">pdo_mysql</div>
              <div class="info-label">Driver PDO untuk MySQL / MariaDB</div>
            </div>
            <span class="badge <?php echo $drivers['pdo_mysql'] ? 'badge-success' : 'badge-danger'; ?>">
              <?php echo $drivers['pdo_mysql'] ? 'TERPASANG' : 'BELUM TERINSTALL'; ?>
            </span>
          </div>

          <div class="info-item" style="display: flex; justify-content: space-between; align-items: center;">
            <div>
              <div class="info-val">mysqli</div>
              <div class="info-label">Driver native MySQLi</div>
            </div>
            <span class="badge <?php echo $drivers['mysqli'] ? 'badge-success' : 'badge-danger'; ?>">
              <?php echo $drivers['mysqli'] ? 'TERPASANG' : 'BELUM TERINSTALL'; ?>
            </span>
          </div>

          <div class="info-item" style="display: flex; justify-content: space-between; align-items: center;">
            <div>
              <div class="info-val">pdo_pgsql & pgsql</div>
              <div class="info-label">Driver PostgreSQL</div>
            </div>
            <span class="badge <?php echo ($drivers['pdo_pgsql'] || $drivers['pgsql']) ? 'badge-success' : 'badge-warning'; ?>">
              <?php echo ($drivers['pdo_pgsql'] || $drivers['pgsql']) ? 'TERPASANG' : 'Belum Terinstall'; ?>
            </span>
          </div>
        </div>
      </div>

      <!-- Sisi Layanan & Socket -->
      <div>
        <h4 style="font-size: 12px; color: var(--text-muted); text-transform: uppercase; margin-bottom: 8px;">Status Layanan & Socket Server:</h4>
        <div style="display: flex; flex-direction: column; gap: 8px;">
          <div class="info-item" style="display: flex; justify-content: space-between; align-items: center;">
            <div>
              <div class="info-val">MariaDB Port 3306 (TCP)</div>
              <div class="info-label">127.0.0.1:3306</div>
            </div>
            <span class="badge <?php echo $mysqlPortOpen ? 'badge-success' : 'badge-danger'; ?>">
              <?php echo $mysqlPortOpen ? 'LISTENING (ON)' : 'TERTUTUP (OFF)'; ?>
            </span>
          </div>

          <div class="info-item" style="display: flex; justify-content: space-between; align-items: center;">
            <div>
              <div class="info-val">PostgreSQL Port 5432 (TCP)</div>
              <div class="info-label">127.0.0.1:5432</div>
            </div>
            <span class="badge <?php echo $pgsqlPortOpen ? 'badge-success' : 'badge-warning'; ?>">
              <?php echo $pgsqlPortOpen ? 'LISTENING (ON)' : 'Tertutup'; ?>
            </span>
          </div>

          <div class="info-item">
            <div class="info-label">Socket MySQL Ditemukan di FreeBSD:</div>
            <div class="info-val" style="font-size: 11px; word-break: break-all; margin-top: 4px;">
              <?php
              if (!empty($foundMysqlSockets)) {
                  echo implode('<br>', $foundMysqlSockets);
              } else {
                  echo "<span style='color: #f87171;'>Tidak ditemukan di path standar (/var/run/mysql/mysql.sock)</span>";
              }
              ?>
            </div>
          </div>

          <div class="info-item">
            <div class="info-label">File Konfigurasi PHP (php.ini):</div>
            <div class="info-val" style="font-size: 11px; word-break: break-all; margin-top: 4px;">
              <?php echo htmlspecialchars($loadedIni); ?>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Peringatan Jika Driver MySQL Belum Terinstall -->
    <?php if (!$drivers['pdo_mysql'] && !$drivers['mysqli']): ?>
    <div style="margin-top: 16px; background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.4); border-radius: 8px; padding: 14px;">
      <h4 style="color: #f87171; font-size: 13px; font-weight: 700; margin-bottom: 4px;">🚨 PENYEBAB UTAMA: Driver PHP MySQL Belum Terpasang!</h4>
      <p style="color: #cbd5e1; font-size: 12px; margin-bottom: 8px;">
        Aplikasi PHP tidak dapat terhubung ke database karena paket <code>php<?php echo $phpMajorMinor; ?>-mysqli</code> dan <code>php<?php echo $phpMajorMinor; ?>-pdo_mysql</code> belum terinstall di server FreeBSD.
      </p>
      <div style="font-size: 11px; color: #94a3b8;">Jalankan perintah ini di Terminal Server FreeBSD:</div>
      <div class="code-block" style="margin-top: 4px;">doas pkg install -y php<?php echo $phpMajorMinor; ?>-mysqli php<?php echo $phpMajorMinor; ?>-pdo_mysql
doas service php-fpm restart
doas service apache24 restart</div>
    </div>
    <?php endif; ?>
  </div>

  <!-- Card 2: Hasil Uji Diagnostik (Jika sudah submit) -->
  <?php if ($submitted && $testResults): ?>
  <div class="card">
    <div class="card-title">
      <span>2. Hasil Diagnostik Koneksi Database</span>
    </div>

    <div class="verdict-box <?php echo $testResults['success'] ? 'verdict-success' : 'verdict-error'; ?>">
      <h3 style="font-size: 15px; font-weight: 700; margin-bottom: 6px;">
        <?php if ($testResults['success']): ?>
          🎉 STATUS: KONEKSI BERHASIL! (Semua Kredensial & Driver Valid)
        <?php elseif ($testResults['diagnosis'] === 'AUTH_FAILED'): ?>
          ❌ STATUS: GAGAL LOGIN (Username atau Password Salah / Error 1045)
        <?php elseif ($testResults['diagnosis'] === 'UNKNOWN_DATABASE'): ?>
          ❌ STATUS: DATABASE TIDAK DITEMUKAN (User Benar, Nama Database Salah / Error 1049)
        <?php elseif ($testResults['diagnosis'] === 'DRIVER_MISSING'): ?>
          ❌ STATUS: DRIVER PHP BELUM TERINSTALL DI SERVER
        <?php elseif ($testResults['diagnosis'] === 'SOCKET_OR_SERVICE_DOWN'): ?>
          ❌ STATUS: SERVER DATABASE TIDAK MERESPONS / PORT 3306 TUTUP
        <?php else: ?>
          ❌ STATUS: GAGAL TERHUBUNG KE DATABASE
        <?php endif; ?>
      </h3>

      <p style="font-size: 13px; margin-bottom: 10px;">
        Target: <strong><?php echo htmlspecialchars($username); ?></strong>@<strong><?php echo htmlspecialchars($host); ?></strong> &rarr; DB: <strong><?php echo htmlspecialchars($database ?: '(tanpa database)'); ?></strong>
      </p>

      <ul class="step-log">
        <?php foreach ($testResults['messages'] as $msg): ?>
          <li><?php echo htmlspecialchars($msg); ?></li>
        <?php endforeach; ?>
      </ul>

      <?php if (!empty($testResults['recommendation'])): ?>
      <div style="margin-top: 14px; padding-top: 10px; border-top: 1px solid rgba(255,255,255,0.15);">
        <strong style="display: block; font-size: 12px; margin-bottom: 4px;">💡 Langkah Perbaikan / Solusi:</strong>
        <div style="white-space: pre-wrap; font-family: var(--font-mono); font-size: 12px; background: rgba(0,0,0,0.3); padding: 10px; border-radius: 6px;">
          <?php echo htmlspecialchars($testResults['recommendation']); ?>
        </div>
      </div>
      <?php endif; ?>

      <?php if ($testResults['success'] && !empty($testResults['tables'])): ?>
      <div style="margin-top: 12px;">
        <strong style="font-size: 12px;">Daftar Tabel di Database (<?php echo count($testResults['tables']); ?> tabel):</strong>
        <div style="max-height: 120px; overflow-y: auto; background: rgba(0,0,0,0.3); padding: 8px 12px; border-radius: 6px; font-family: var(--font-mono); font-size: 11px; margin-top: 6px;">
          <?php echo implode(', ', array_map('htmlspecialchars', $testResults['tables'])); ?>
        </div>
      </div>
      <?php endif; ?>
    </div>
  </div>
  <?php endif; ?>

  <!-- Card 3: Form Uji Koneksi Interaktif -->
  <div class="card">
    <div class="card-title">
      <span>3. Form Pengujian Koneksi Interaktif</span>
    </div>
    <form method="POST" action="">
      <div style="margin-bottom: 12px;">
        <span style="font-size: 11px; color: var(--text-muted); margin-right: 8px;">Preset Host Cepat:</span>
        <button type="button" class="quick-preset" onclick="setHost('127.0.0.1')">127.0.0.1 (TCP Standard)</button>
        <button type="button" class="quick-preset" onclick="setHost('localhost')">localhost (UNIX Socket)</button>
        <?php if (!empty($foundMysqlSockets)): ?>
          <button type="button" class="quick-preset" onclick="setSocket('<?php echo htmlspecialchars($foundMysqlSockets[0]); ?>')">Socket FreeBSD (<?php echo htmlspecialchars($foundMysqlSockets[0]); ?>)</button>
        <?php endif; ?>
      </div>

      <div class="grid-2">
        <div class="form-group">
          <label class="form-label">Database Engine:</label>
          <select name="engine" class="form-select" onchange="toggleEngine(this.value)">
            <option value="mysql" <?php echo $engine === 'mysql' ? 'selected' : ''; ?>>MariaDB / MySQL</option>
            <option value="pgsql" <?php echo $engine === 'pgsql' ? 'selected' : ''; ?>>PostgreSQL 16</option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">Host Database (Rekomendasi: 127.0.0.1):</label>
          <input type="text" id="input_host" name="host" class="form-input" required value="<?php echo htmlspecialchars($host); ?>" placeholder="127.0.0.1 atau localhost">
        </div>
      </div>

      <div class="grid-3">
        <div class="form-group">
          <label class="form-label">Port:</label>
          <input type="number" id="input_port" name="port" class="form-input" required value="<?php echo htmlspecialchars($port); ?>">
        </div>

        <div class="form-group">
          <label class="form-label">Nama Database:</label>
          <input type="text" name="database" class="form-input" value="<?php echo htmlspecialchars($database); ?>" placeholder="contoh: u_go_db">
        </div>

        <div class="form-group">
          <label class="form-label">Username Database:</label>
          <input type="text" name="username" class="form-input" required value="<?php echo htmlspecialchars($username); ?>" placeholder="contoh: u_go">
        </div>
      </div>

      <div class="grid-2">
        <div class="form-group">
          <label class="form-label">Password User Database:</label>
          <input type="password" name="password" class="form-input" value="<?php echo htmlspecialchars($password); ?>" placeholder="Masukkan password database">
        </div>

        <div class="form-group">
          <label class="form-label">Custom UNIX Socket (Opsional):</label>
          <input type="text" id="input_socket" name="socket" class="form-input" value="<?php echo htmlspecialchars($socket); ?>" placeholder="/var/run/mysql/mysql.sock">
        </div>
      </div>

      <div style="display: flex; gap: 10px; align-items: center; margin-top: 10px;">
        <button type="submit" class="btn">
          <span>🚀 Jalankan Tes Koneksi Sekarang</span>
        </button>
        <button type="button" class="btn btn-secondary" onclick="window.location.href='testdb.php'">
          Reset Form
        </button>
      </div>
    </form>
  </div>

  <!-- Card 4: Contoh Kode Koneksi PHP Siap Pakai -->
  <div class="card">
    <div class="card-title">
      <span>4. Kode Koneksi PHP Siap Pakai</span>
    </div>
    <p style="font-size: 12px; color: var(--text-muted); margin-bottom: 12px;">
      Gunakan salah satu cuplikan kode di bawah ini pada file aplikasi web Anda (seperti <code>config.php</code> atau <code>koneksi.php</code>):
    </p>

    <!-- Tab PDO -->
    <div style="margin-bottom: 14px;">
      <div style="font-size: 12px; font-weight: 700; color: #38bdf8;">Metode 1: PDO MySQL (Sangat Direkomendasikan / Aman)</div>
      <div class="code-block" id="codePdo">
&lt;?php
// file: config.php atau koneksi.php
$dbHost = '<?php echo htmlspecialchars($host ?: '127.0.0.1'); ?>';
$dbPort = <?php echo intval($port ?: 3306); ?>;
$dbName = '<?php echo htmlspecialchars($database ?: 'nama_db'); ?>';
$dbUser = '<?php echo htmlspecialchars($username ?: 'user_db'); ?>';
$dbPass = '<?php echo htmlspecialchars($password ?: 'password_db'); ?>';

try {
    $dsn = "mysql:host={$dbHost};port={$dbPort};dbname={$dbName};charset=utf8mb4";
    $pdo = new PDO($dsn, $dbUser, $dbPass, [
        PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
        PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
        PDO::ATTR_EMULATE_PREPARES   => false,
    ]);
    // Koneksi berhasil!
} catch (PDOException $e) {
    die("Koneksi Database Gagal: " . $e->getMessage());
}
        <button type="button" class="copy-btn" onclick="copySnippet('codePdo')">Salin Kode</button>
      </div>
    </div>

    <!-- Tab MySQLi -->
    <div>
      <div style="font-size: 12px; font-weight: 700; color: #a7f3d0;">Metode 2: MySQLi Procedural</div>
      <div class="code-block" id="codeMysqli">
&lt;?php
// file: koneksi.php (MySQLi)
$host = '<?php echo htmlspecialchars($host ?: '127.0.0.1'); ?>';
$user = '<?php echo htmlspecialchars($username ?: 'user_db'); ?>';
$pass = '<?php echo htmlspecialchars($password ?: 'password_db'); ?>';
$db   = '<?php echo htmlspecialchars($database ?: 'nama_db'); ?>';
$port = <?php echo intval($port ?: 3306); ?>;

$conn = mysqli_connect($host, $user, $pass, $db, $port);

if (!$conn) {
    die("Koneksi gagal: " . mysqli_connect_error());
}
// Koneksi berhasil!
        <button type="button" class="copy-btn" onclick="copySnippet('codeMysqli')">Salin Kode</button>
      </div>
    </div>
  </div>
</div>

<script>
function setHost(val) {
  document.getElementById('input_host').value = val;
  document.getElementById('input_socket').value = '';
}
function setSocket(val) {
  document.getElementById('input_host').value = 'localhost';
  document.getElementById('input_socket').value = val;
}
function toggleEngine(val) {
  const portInput = document.getElementById('input_port');
  if (val === 'pgsql') {
    portInput.value = '5432';
  } else {
    portInput.value = '3306';
  }
}
function copySnippet(elementId) {
  const el = document.getElementById(elementId);
  const text = el.innerText.replace('Salin Kode', '').trim();
  navigator.clipboard.writeText(text).then(() => {
    alert('Kode koneksi berhasil disalin ke clipboard!');
  }).catch(() => {
    prompt('Salin manual kode:', text);
  });
}
</script>
</body>
</html>
