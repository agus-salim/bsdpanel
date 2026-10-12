# ==============================================================================
# BSDPanel Makefile
# Target: FreeBSD (amd64) & Local Development
# Repository: https://github.com/agus-salim/bsdpanel.git
# ==============================================================================

APP_NAME := bsdpanel
VERSION := 1.0.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: all build-local build-freebsd package clean run-dev git-push

all: build-freebsd

## build-local: Kompilasi binary untuk sistem host saat ini
build-local:
	@echo "==> Mengompilasi untuk host lokal..."
	go build -ldflags="$(LDFLAGS)" -o bin/$(APP_NAME) cmd/bsdpanel/main.go
	@echo "==> Selesai: bin/$(APP_NAME)"

## build-freebsd: Cross-compile static binary untuk FreeBSD amd64
build-freebsd:
	@echo "==> Cross-compiling untuk FreeBSD amd64 (CGO_ENABLED=0)..."
	@mkdir -p bin/freebsd_amd64
	env CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/freebsd_amd64/$(APP_NAME) cmd/bsdpanel/main.go
	@echo "==> Sukses: bin/freebsd_amd64/$(APP_NAME)"

## package: Buat tarball rilis untuk distribusi GitHub Releases
package: build-freebsd
	@echo "==> Mengemas rilis tarball FreeBSD..."
	@mkdir -p dist
	cp bin/freebsd_amd64/$(APP_NAME) dist/$(APP_NAME)
	cp scripts/bsdpanel.rc dist/bsdpanel.rc
	tar -czf dist/$(APP_NAME)-freebsd-amd64.tar.gz -C dist $(APP_NAME) bsdpanel.rc
	@rm -f dist/$(APP_NAME) dist/bsdpanel.rc
	@cd dist && sha256sum $(APP_NAME)-freebsd-amd64.tar.gz > $(APP_NAME)-freebsd-amd64.tar.gz.sha256 2>/dev/null || shasum -a 256 $(APP_NAME)-freebsd-amd64.tar.gz > $(APP_NAME)-freebsd-amd64.tar.gz.sha256
	@echo "==> Paket rilis siap di dist/$(APP_NAME)-freebsd-amd64.tar.gz"

## run-dev: Jalankan panel dalam mode demo/dry-run lokal
run-dev:
	go run cmd/bsdpanel/main.go -dry-run -listen :8080

## git-push: Commit dan push ke repository GitHub
git-push:
	@echo "==> Mendorong perubahan ke GitHub repo..."
	git add .
	git commit -m "feat: bsdpanel core architecture, freebsd runner, vhost & php management, ui, rc.d script" || true
	git branch -M main
	git push -u origin main

## clean: Hapus binary dan artifact build
clean:
	@rm -rf bin/ dist/
	@echo "==> Direktori build dibersihkan."
