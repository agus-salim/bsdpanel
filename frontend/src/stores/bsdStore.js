import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const API_BASE = 'http://localhost:8888/api'

export const useBsdStore = defineStore('bsd', () => {
  // Loading & error states
  const isLoading = ref(false)
  const isBackendConnected = ref(false)

  // Current logged in user
  const currentUser = ref({
    username: 'root',
    role: 'admin',
    email: 'admin@bsdpanel.lan'
  })

  // 1. Dashboard & Telemetry
  const stats = ref({
    os_name: 'FreeBSD 14.2-RELEASE-p1 (GENERIC)',
    hostname: 'bsd-node01.datacenter.lan',
    uptime_formatted: '4 days, 4 hours, 23 mins',
    cpu_model: 'Intel(R) Xeon(R) Gold 6338 CPU @ 2.00GHz',
    cpu_cores: 8,
    cpu_usage_percent: 14.8,
    load_average: [0.38, 0.42, 0.35],
    memory_total_mb: 16384,
    memory_used_mb: 4890,
    memory_free_mb: 11494,
    memory_usage_percent: 29.8,
    swap_total_mb: 4096,
    swap_used_mb: 120,
    disk_total_gb: 480.0,
    disk_used_gb: 92.4,
    disk_free_gb: 387.6,
    disk_usage_percent: 19.25,
    zfs_pool_name: 'zroot',
    zfs_health: 'ONLINE',
    pf_states_count: 142,
    active_services_count: 7
  })

  // 2. Webservers (Nginx, Apache, LiteSpeed, Caddy)
  const webservers = ref([
    {
      server_type: 'nginx',
      display_name: 'NGINX High Performance HTTP Server',
      is_installed: true,
      is_running: true,
      is_primary: true,
      version: '1.26.2',
      port_http: 80,
      port_https: 443,
      service_name: 'nginx',
      config_path: '/usr/local/etc/nginx/nginx.conf',
      memory_mb: 64.5,
      cpu_percent: 1.2
    },
    {
      server_type: 'apache',
      display_name: 'Apache HTTP Server (httpd24)',
      is_installed: true,
      is_running: false,
      is_primary: false,
      version: '2.4.62',
      port_http: 8080,
      port_https: 8443,
      service_name: 'apache24',
      config_path: '/usr/local/etc/apache24/httpd.conf',
      memory_mb: 0.0,
      cpu_percent: 0.0
    },
    {
      server_type: 'litespeed',
      display_name: 'OpenLiteSpeed Web Server',
      is_installed: true,
      is_running: false,
      is_primary: false,
      version: '1.8.2',
      port_http: 8088,
      port_https: 7080,
      service_name: 'lsws',
      config_path: '/usr/local/lsws/conf/httpd_config.conf',
      memory_mb: 0.0,
      cpu_percent: 0.0
    },
    {
      server_type: 'caddy',
      display_name: 'Caddy Server (Automated TLS)',
      is_installed: true,
      is_running: false,
      is_primary: false,
      version: '2.8.4',
      port_http: 2015,
      port_https: 2016,
      service_name: 'caddy',
      config_path: '/usr/local/etc/caddy/Caddyfile',
      memory_mb: 0.0,
      cpu_percent: 0.0
    }
  ])

  // Domains
  const domains = ref([
    {
      id: 'dom-1',
      domain_name: 'portal-mtsn1skd.sch.id',
      owner_username: 'client_company',
      webserver_type: 'nginx',
      php_version: '8.3',
      document_root: '/usr/local/www/portal-mtsn1skd.sch.id/public',
      ssl_enabled: true,
      force_https: true,
      is_active: true,
      reverse_proxy_target: null,
      created_at: Date.now() - 86400000 * 4
    },
    {
      id: 'dom-2',
      domain_name: 'cbt-exam.school.net',
      owner_username: 'client_company',
      webserver_type: 'caddy',
      php_version: '8.4',
      document_root: '/usr/local/www/cbt-exam.school.net/public',
      ssl_enabled: true,
      force_https: true,
      is_active: true,
      reverse_proxy_target: null,
      created_at: Date.now() - 86400000 * 2
    },
    {
      id: 'dom-3',
      domain_name: 'legacy-simak.org',
      owner_username: 'client_company',
      webserver_type: 'apache',
      php_version: '8.2',
      document_root: '/usr/local/www/legacy-simak.org/public',
      ssl_enabled: false,
      force_https: false,
      is_active: true,
      reverse_proxy_target: null,
      created_at: Date.now() - 86400000 * 12
    }
  ])

  // 3. PHP Versions (8.2 - 8.6)
  const phpVersions = ref([
    {
      version: '8.2',
      is_installed: true,
      is_running: true,
      service_name: 'php-fpm-82',
      socket_path: '/var/run/php-fpm-82.sock',
      port: 9002,
      memory_mb: 38.4,
      active_processes: 4,
      ini_path: '/usr/local/etc/php82/php.ini'
    },
    {
      version: '8.3',
      is_installed: true,
      is_running: true,
      service_name: 'php-fpm-83',
      socket_path: '/var/run/php-fpm-83.sock',
      port: 9003,
      memory_mb: 42.1,
      active_processes: 5,
      ini_path: '/usr/local/etc/php83/php.ini'
    },
    {
      version: '8.4',
      is_installed: true,
      is_running: true,
      service_name: 'php-fpm-84',
      socket_path: '/var/run/php-fpm-84.sock',
      port: 9004,
      memory_mb: 32.8,
      active_processes: 3,
      ini_path: '/usr/local/etc/php84/php.ini'
    },
    {
      version: '8.5',
      is_installed: true,
      is_running: false,
      service_name: 'php-fpm-85',
      socket_path: '/var/run/php-fpm-85.sock',
      port: 9005,
      memory_mb: 0.0,
      active_processes: 0,
      ini_path: '/usr/local/etc/php85/php.ini'
    },
    {
      version: '8.6',
      is_installed: false,
      is_running: false,
      service_name: 'php-fpm-86',
      socket_path: '/var/run/php-fpm-86.sock',
      port: 9006,
      memory_mb: 0.0,
      active_processes: 0,
      ini_path: '/usr/local/etc/php86/php.ini'
    }
  ])

  // PHP Extensions
  const phpExtensions = ref([
    { name: 'opcache', is_enabled: true, category: 'Performance', description: 'Zend OPcache bytecode caching' },
    { name: 'mbstring', is_enabled: true, category: 'String', description: 'Multibyte string support' },
    { name: 'curl', is_enabled: true, category: 'Network', description: 'libcurl URL library' },
    { name: 'pdo_mysql', is_enabled: true, category: 'Database', description: 'MariaDB / MySQL PDO driver' },
    { name: 'pdo_pgsql', is_enabled: true, category: 'Database', description: 'PostgreSQL PDO driver' },
    { name: 'gd', is_enabled: true, category: 'Media', description: 'GD Graphics Image Library' },
    { name: 'zip', is_enabled: true, category: 'Compression', description: 'ZIP archive management' },
    { name: 'intl', is_enabled: true, category: 'Localization', description: 'ICU Internationalization' },
    { name: 'redis', is_enabled: true, category: 'Cache', description: 'Redis memory cache' },
    { name: 'imagick', is_enabled: false, category: 'Media', description: 'ImageMagick raster editor' },
    { name: 'sodium', is_enabled: true, category: 'Security', description: 'Modern cryptographic algorithms' }
  ])

  // PHP INI Settings
  const phpIni = ref({
    memory_limit: '256M',
    upload_max_filesize: '64M',
    post_max_size: '128M',
    max_execution_time: 300,
    max_input_vars: 3000,
    display_errors: false,
    opcache_enable: true,
    opcache_memory_consumption: 128,
    disable_functions: 'exec,passthru,shell_exec,system,proc_open,popen',
    date_timezone: 'Asia/Jakarta'
  })

  // 4. Databases (PostgreSQL, MariaDB)
  const dbServers = ref([
    {
      engine: 'mariadb',
      display_name: 'MariaDB SQL Server (10.11 LTS)',
      is_installed: true,
      is_running: true,
      version: '10.11.8-MariaDB',
      service_name: 'mysql-server',
      port: 3306,
      memory_mb: 184.2,
      databases_count: 5,
      connections_count: 12
    },
    {
      engine: 'postgresql',
      display_name: 'PostgreSQL Database Engine (16)',
      is_installed: true,
      is_running: true,
      version: 'PostgreSQL 16.4',
      service_name: 'postgresql',
      port: 5432,
      memory_mb: 112.5,
      databases_count: 3,
      connections_count: 7
    }
  ])

  const databases = ref([
    {
      id: 'db-1',
      name: 'madrasah_portal_db',
      engine: 'mariadb',
      owner_username: 'client_company',
      size_mb: 52.4,
      tables_count: 42,
      collation: 'utf8mb4_unicode_ci',
      created_at: Date.now() - 86400000 * 5
    },
    {
      id: 'db-2',
      name: 'cbt_exam_scoring',
      engine: 'mariadb',
      owner_username: 'client_company',
      size_mb: 28.1,
      tables_count: 19,
      collation: 'utf8mb4_unicode_ci',
      created_at: Date.now() - 86400000 * 3
    },
    {
      id: 'db-3',
      name: 'alumni_analytics_pg',
      engine: 'postgresql',
      owner_username: 'client_company',
      size_mb: 118.6,
      tables_count: 16,
      collation: 'en_US.UTF-8',
      created_at: Date.now() - 86400000 * 6
    }
  ])

  const dbUsers = ref([
    {
      id: 'dbu-1',
      username: 'usr_portal_db',
      engine: 'mariadb',
      owner_username: 'client_company',
      allowed_host: 'localhost',
      assigned_databases: ['madrasah_portal_db'],
      created_at: Date.now() - 86400000 * 5
    },
    {
      id: 'dbu-2',
      username: 'pg_alumni_usr',
      engine: 'postgresql',
      owner_username: 'client_company',
      allowed_host: '127.0.0.1',
      assigned_databases: ['alumni_analytics_pg'],
      created_at: Date.now() - 86400000 * 6
    }
  ])

  // 5. Firewall (PF) & Fail2ban
  const pfEnabled = ref(true)
  const firewallRules = ref([
    {
      id: 'rule-1',
      label: 'SSH Remote Console',
      port: '22',
      protocol: 'tcp',
      action: 'pass',
      direction: 'in',
      source: 'any',
      is_enabled: true
    },
    {
      id: 'rule-2',
      label: 'HTTP Web Traffic',
      port: '80',
      protocol: 'tcp',
      action: 'pass',
      direction: 'in',
      source: 'any',
      is_enabled: true
    },
    {
      id: 'rule-3',
      label: 'HTTPS Encrypted Web',
      port: '443',
      protocol: 'tcp',
      action: 'pass',
      direction: 'in',
      source: 'any',
      is_enabled: true
    },
    {
      id: 'rule-4',
      label: 'BSD Panel Dashboard Port',
      port: '8888',
      protocol: 'tcp',
      action: 'pass',
      direction: 'in',
      source: 'any',
      is_enabled: true
    },
    {
      id: 'rule-5',
      label: 'MariaDB Internal Only',
      port: '3306',
      protocol: 'tcp',
      action: 'block',
      direction: 'in',
      source: 'any',
      is_enabled: true
    },
    {
      id: 'rule-6',
      label: 'PostgreSQL Internal Only',
      port: '5432',
      protocol: 'tcp',
      action: 'block',
      direction: 'in',
      source: 'any',
      is_enabled: true
    }
  ])

  const fail2banJails = ref([
    {
      name: 'bsd-sshd',
      is_active: true,
      filter: 'sshd',
      log_path: '/var/log/auth.log',
      max_retry: 4,
      find_time: 600,
      ban_time: 3600,
      currently_banned: 3,
      total_banned: 52
    },
    {
      name: 'bsd-panel-auth',
      is_active: true,
      filter: 'bsd-panel',
      log_path: '/var/log/bsdpanel/access.log',
      max_retry: 5,
      find_time: 300,
      ban_time: 7200,
      currently_banned: 1,
      total_banned: 14
    },
    {
      name: 'nginx-badbots',
      is_active: true,
      filter: 'nginx-badbots',
      log_path: '/var/log/nginx/error.log',
      max_retry: 2,
      find_time: 600,
      ban_time: 86400,
      currently_banned: 2,
      total_banned: 108
    }
  ])

  const bannedIps = ref([
    {
      ip: '198.51.100.23',
      jail: 'bsd-sshd',
      banned_at: Date.now() - 1800000,
      expires_at: Date.now() + 1800000,
      country: 'RU'
    },
    {
      ip: '203.0.113.88',
      jail: 'bsd-sshd',
      banned_at: Date.now() - 3200000,
      expires_at: Date.now() + 400000,
      country: 'CN'
    },
    {
      ip: '185.220.101.5',
      jail: 'bsd-sshd',
      banned_at: Date.now() - 900000,
      expires_at: Date.now() + 2700000,
      country: 'DE'
    },
    {
      ip: '192.0.2.14',
      jail: 'bsd-panel-auth',
      banned_at: Date.now() - 1200000,
      expires_at: Date.now() + 6000000,
      country: 'US'
    },
    {
      ip: '45.140.19.8',
      jail: 'nginx-badbots',
      banned_at: Date.now() - 14400000,
      expires_at: Date.now() + 72000000,
      country: 'NL'
    }
  ])

  // 6. Users & Resellers
  const users = ref([
    {
      id: 'usr-1',
      username: 'root',
      role: 'admin',
      email: 'admin@bsdpanel.lan',
      disk_limit_mb: 0,
      disk_used_mb: 2450,
      bandwidth_limit_mb: 0,
      bandwidth_used_mb: 18200,
      domains_limit: 0,
      databases_limit: 0,
      status: 'active',
      created_at: Date.now() - 86400000 * 30
    },
    {
      id: 'usr-2',
      username: 'reseller_sumatra',
      role: 'reseller',
      email: 'sumatra@provider.id',
      disk_limit_mb: 102400,
      disk_used_mb: 14200,
      bandwidth_limit_mb: 1048576,
      bandwidth_used_mb: 85400,
      domains_limit: 50,
      databases_limit: 100,
      status: 'active',
      created_at: Date.now() - 86400000 * 20
    },
    {
      id: 'usr-3',
      username: 'client_company',
      role: 'user',
      email: 'operator@mtsn1skd.sch.id',
      disk_limit_mb: 10240,
      disk_used_mb: 3120,
      bandwidth_limit_mb: 102400,
      bandwidth_used_mb: 12400,
      domains_limit: 5,
      databases_limit: 10,
      status: 'active',
      created_at: Date.now() - 86400000 * 10
    }
  ])

  // 7. System Services
  const systemServices = ref([
    { name: 'nginx', description: 'High Performance HTTP & Reverse Proxy Server', is_running: true, is_enabled_boot: true },
    { name: 'php-fpm-83', description: 'PHP 8.3 FastCGI Process Manager', is_running: true, is_enabled_boot: true },
    { name: 'mysql-server', description: 'MariaDB SQL Database Daemon', is_running: true, is_enabled_boot: true },
    { name: 'postgresql', description: 'PostgreSQL Database Engine', is_running: true, is_enabled_boot: true },
    { name: 'pf', description: 'FreeBSD Packet Filter Firewall Engine', is_running: true, is_enabled_boot: true },
    { name: 'fail2ban', description: 'Automated Intrusion Prevention & Ban Service', is_running: true, is_enabled_boot: true },
    { name: 'sshd', description: 'OpenSSH Secure Shell Daemon', is_running: true, is_enabled_boot: true }
  ])

  // Actions
  function toggleWebserver(serverType) {
    const srv = webservers.value.find(w => w.server_type === serverType)
    if (srv) {
      srv.is_running = !srv.is_running
      srv.memory_mb = srv.is_running ? 54.0 : 0.0
    }
  }

  function togglePhpVersion(version) {
    const php = phpVersions.value.find(p => p.version === version)
    if (php) {
      php.is_running = !php.is_running
      php.active_processes = php.is_running ? 4 : 0
      php.memory_mb = php.is_running ? 36.5 : 0.0
    }
  }

  function toggleExtension(extName) {
    const ext = phpExtensions.value.find(e => e.name === extName)
    if (ext) {
      ext.is_enabled = !ext.is_enabled
    }
  }

  function addDomain(domainPayload) {
    domains.value.push({
      id: 'dom-' + Date.now(),
      domain_name: domainPayload.domain_name,
      owner_username: 'client_company',
      webserver_type: domainPayload.webserver_type,
      php_version: domainPayload.php_version,
      document_root: `/usr/local/www/${domainPayload.domain_name}/public`,
      ssl_enabled: domainPayload.ssl_auto_cert,
      force_https: domainPayload.force_https,
      is_active: true,
      reverse_proxy_target: domainPayload.reverse_proxy_target || null,
      created_at: Date.now()
    })
  }

  function deleteDomain(id) {
    domains.value = domains.value.filter(d => d.id !== id)
  }

  function addDatabase(dbPayload) {
    databases.value.push({
      id: 'db-' + Date.now(),
      name: dbPayload.name,
      engine: dbPayload.engine,
      owner_username: 'client_company',
      size_mb: 0.1,
      tables_count: 0,
      collation: dbPayload.collation || (dbPayload.engine === 'mariadb' ? 'utf8mb4_unicode_ci' : 'en_US.UTF-8'),
      created_at: Date.now()
    })
  }

  function deleteDatabase(id) {
    databases.value = databases.value.filter(d => d.id !== id)
  }

  function togglePf() {
    pfEnabled.value = !pfEnabled.value
  }

  function addFirewallRule(rule) {
    firewallRules.value.push({
      id: 'rule-' + Date.now(),
      ...rule,
      is_enabled: true
    })
  }

  function deleteFirewallRule(id) {
    firewallRules.value = firewallRules.value.filter(r => r.id !== id)
  }

  function unbanIp(ip, jail) {
    bannedIps.value = bannedIps.value.filter(b => !(b.ip === ip && b.jail === jail))
    const j = fail2banJails.value.find(j => j.name === jail)
    if (j && j.currently_banned > 0) {
      j.currently_banned--
    }
  }

  function banIp(ip, jail) {
    bannedIps.value.push({
      ip,
      jail,
      banned_at: Date.now(),
      expires_at: Date.now() + 3600000,
      country: 'MANUAL'
    })
    const j = fail2banJails.value.find(j => j.name === jail)
    if (j) {
      j.currently_banned++
      j.total_banned++
    }
  }

  function addUser(userPayload) {
    users.value.push({
      id: 'usr-' + Date.now(),
      username: userPayload.username,
      role: userPayload.role,
      email: userPayload.email,
      disk_limit_mb: Number(userPayload.disk_limit_mb) || 10240,
      disk_used_mb: 0,
      bandwidth_limit_mb: Number(userPayload.bandwidth_limit_mb) || 102400,
      bandwidth_used_mb: 0,
      domains_limit: Number(userPayload.domains_limit) || 5,
      databases_limit: Number(userPayload.databases_limit) || 10,
      status: 'active',
      created_at: Date.now()
    })
  }

  function toggleUserStatus(id) {
    const u = users.value.find(user => user.id === id)
    if (u && u.role !== 'admin') {
      u.status = u.status === 'active' ? 'suspended' : 'active'
    }
  }

  function deleteUser(id) {
    users.value = users.value.filter(u => u.id !== id)
  }

  function toggleService(serviceName) {
    const srv = systemServices.value.find(s => s.name === serviceName)
    if (srv) {
      srv.is_running = !srv.is_running
    }
  }

  return {
    isLoading,
    isBackendConnected,
    currentUser,
    stats,
    webservers,
    domains,
    phpVersions,
    phpExtensions,
    phpIni,
    dbServers,
    databases,
    dbUsers,
    pfEnabled,
    firewallRules,
    fail2banJails,
    bannedIps,
    users,
    systemServices,
    toggleWebserver,
    togglePhpVersion,
    toggleExtension,
    addDomain,
    deleteDomain,
    addDatabase,
    deleteDatabase,
    togglePf,
    addFirewallRule,
    deleteFirewallRule,
    unbanIp,
    banIp,
    addUser,
    toggleUserStatus,
    deleteUser,
    toggleService
  }
})
