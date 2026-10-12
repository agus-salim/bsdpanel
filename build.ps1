param(
    [string]$Target = "all"
)

$ErrorActionPreference = "Stop"

function Build-FreeBSD {
    Write-Host "==> [1/2] Cross-compiling untuk FreeBSD amd64..." -ForegroundColor Cyan
    $env:GOOS = "freebsd"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    if (!(Test-Path "bin/freebsd_amd64")) {
        New-Item -ItemType Directory -Path "bin/freebsd_amd64" -Force | Out-Null
    }
    go build -ldflags="-s -w" -o bin/freebsd_amd64/bsdpanel ./cmd/bsdpanel
    $file = Get-Item "bin/freebsd_amd64/bsdpanel"
    Write-Host "==> Sukses: bin/freebsd_amd64/bsdpanel ($([math]::round($file.Length/1MB, 2)) MB)" -ForegroundColor Green
}

function Build-Windows {
    Write-Host "==> [2/2] Mengompilasi untuk Windows host..." -ForegroundColor Cyan
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    if (!(Test-Path "bin")) {
        New-Item -ItemType Directory -Path "bin" -Force | Out-Null
    }
    go build -ldflags="-s -w" -o bin/bsdpanel.exe ./cmd/bsdpanel
    $file = Get-Item "bin/bsdpanel.exe"
    Write-Host "==> Sukses: bin/bsdpanel.exe ($([math]::round($file.Length/1MB, 2)) MB)" -ForegroundColor Green
}

function Run-Dev {
    Write-Host "==> Menjalankan BSDPanel di Windows (Mode Demo/Dry-Run)..." -ForegroundColor Cyan
    $env:GOOS = "windows"
    go run ./cmd/bsdpanel -dry-run -listen :8080
}

switch ($Target.ToLower()) {
    "freebsd" { Build-FreeBSD }
    "windows" { Build-Windows }
    "dev"     { Run-Dev }
    "all"     { 
        Build-FreeBSD
        Build-Windows 
        Write-Host "`nSemua target berhasil dikompilasi!" -ForegroundColor Green
        Write-Host "Target FreeBSD : bin/freebsd_amd64/bsdpanel"
        Write-Host "Target Windows : bin/bsdpanel.exe"
    }
    default   { 
        Write-Host "Target tidak dikenal. Pilihan: freebsd, windows, dev, all" -ForegroundColor Yellow 
    }
}
