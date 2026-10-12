package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// SessionPayload menyimpan klaim data autentikasi stateless.
type SessionPayload struct {
	Username  string `json:"usr"`
	Role      Role   `json:"rol"`
	ExpiresAt int64  `json:"exp"`
}

type AuthManager struct {
	secretKey []byte
}

func NewAuthManager(secretKey string) *AuthManager {
	return &AuthManager{
		secretKey: []byte(secretKey),
	}
}

// GenerateToken membuat token berformat header.payload.signature mirip JWT menggunakan HMAC-SHA256.
func (a *AuthManager) GenerateToken(username string, role Role, duration time.Duration) (string, error) {
	payload := SessionPayload{
		Username:  username,
		Role:      role,
		ExpiresAt: time.Now().Add(duration).Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// Hitung tanda tangan HMAC
	mac := hmac.New(sha256.New, a.secretKey)
	mac.Write([]byte(payloadB64))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s.%s", payloadB64, signature), nil
}

// VerifyToken memvalidasi keaslian tanda tangan token dan batas kedaluwarsa.
func (a *AuthManager) VerifyToken(tokenString string) (*SessionPayload, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 2 {
		return nil, errors.New("format token tidak valid")
	}

	payloadB64 := parts[0]
	signature := parts[1]

	// Verifikasi HMAC
	mac := hmac.New(sha256.New, a.secretKey)
	mac.Write([]byte(payloadB64))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return nil, errors.New("tanda tangan token tidak valid")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, errors.New("gagal decode payload token")
	}

	var payload SessionPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, errors.New("gagal parse payload token")
	}

	if time.Now().Unix() > payload.ExpiresAt {
		return nil, errors.New("token telah kedaluwarsa")
	}

	return &payload, nil
}

// SetAuthCookie menyimpan token ke cookie HTTP-only yang aman.
func (a *AuthManager) SetAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "bsdpanel_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Dapat diset true saat menggunakan HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // 7 hari
	})
}

// ClearAuthCookie menghapus sesi.
func (a *AuthManager) ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "bsdpanel_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}
