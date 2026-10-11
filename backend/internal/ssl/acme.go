package ssl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// IssueResult menyimpan informasi sertifikat yang berhasil diterbitkan
type IssueResult struct {
	Domain   string `json:"domain"`
	CertPath string `json:"cert_path"`
	KeyPath  string `json:"key_path"`
	Issuer   string `json:"issuer"`
	Message  string `json:"message"`
}

type ACMEManager struct{}

func NewACMEManager() *ACMEManager {
	return &ACMEManager{}
}

// IssueCertificate meminta sertifikat SSL Let's Encrypt menggunakan webroot challenge
func (m *ACMEManager) IssueCertificate(ctx context.Context, domain string, webroot string, email string) (*IssueResult, error) {
	if email == "" {
		email = "admin@" + domain
	}

	// Lokasi standar sertifikat Let's Encrypt di FreeBSD (/usr/local/etc/letsencrypt/live/)
	baseCertDir := "/usr/local/etc/letsencrypt/live/" + domain
	certPath := filepath.Join(baseCertDir, "fullchain.pem")
	keyPath := filepath.Join(baseCertDir, "privkey.pem")

	if runtime.GOOS != "freebsd" {
		// Mock sertifikat untuk pengujian di platform development
		_ = os.MkdirAll(filepath.Dir("temp_ssl/"+domain), 0755)
		return &IssueResult{
			Domain:   domain,
			CertPath: "/usr/local/etc/letsencrypt/live/" + domain + "/fullchain.pem",
			KeyPath:  "/usr/local/etc/letsencrypt/live/" + domain + "/privkey.pem",
			Issuer:   "Let's Encrypt (Dev Simulation)",
			Message:  "Sertifikat SSL berhasil diterbitkan (mode simulasi pengembangan)",
		}, nil
	}

	// Pastikan folder .well-known/acme-challenge dapat diakses oleh web server
	challengeDir := filepath.Join(webroot, ".well-known", "acme-challenge")
	if err := os.MkdirAll(challengeDir, 0755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori acme-challenge: %w", err)
	}

	// Cek apakah certbot tersedia di FreeBSD (biasanya di /usr/local/bin/certbot)
	certbotPath, err := exec.LookPath("certbot")
	if err != nil {
		// Cek acme.sh sebagai alternatif populer di FreeBSD ports
		if acmePath, acmeErr := exec.LookPath("acme.sh"); acmeErr == nil {
			return m.issueWithAcmeSh(ctx, acmePath, domain, webroot)
		}
		return nil, fmt.Errorf("utilitas certbot atau acme.sh belum terinstal di FreeBSD. Jalankan: pkg install py-certbot")
	}

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	// Eksekusi certbot webroot mode
	args := []string{
		"certonly",
		"--webroot",
		"-w", webroot,
		"-d", domain,
		"--agree-tos",
		"-m", email,
		"--non-interactive",
		"--keep-until-expiring",
	}

	cmd := exec.CommandContext(ctxTimeout, certbotPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("proses certbot gagal: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}

	return &IssueResult{
		Domain:   domain,
		CertPath: certPath,
		KeyPath:  keyPath,
		Issuer:   "Let's Encrypt Authority",
		Message:  "Sertifikat SSL Let's Encrypt berhasil diterbitkan dan dipasang",
	}, nil
}

func (m *ACMEManager) issueWithAcmeSh(ctx context.Context, acmePath, domain, webroot string) (*IssueResult, error) {
	args := []string{
		"--issue",
		"-d", domain,
		"-w", webroot,
		"--server", "letsencrypt",
	}
	cmd := exec.CommandContext(ctx, acmePath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("proses acme.sh gagal: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}

	return &IssueResult{
		Domain:   domain,
		CertPath: "/root/.acme.sh/" + domain + "/fullchain.cer",
		KeyPath:  "/root/.acme.sh/" + domain + "/" + domain + ".key",
		Issuer:   "Let's Encrypt (via acme.sh)",
		Message:  "Sertifikat SSL berhasil diterbitkan menggunakan acme.sh",
	}, nil
}
