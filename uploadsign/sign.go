// Package uploadsign signs stored-object URLs with an HMAC'd expiry and
// verifies them. Pure logic, no global dependency, so it stays testable.
package uploadsign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

// Query params: e=expiry timestamp (seconds), s=signature.
const (
	ExpiryParam = "e"
	ValueParam  = "s"
)

// Sign signs a storage key with an expiry; an empty secret means unsigned.
func Sign(secret, key string, expiryUnix int64) string {
	if secret == "" || key == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(key))
	mac.Write([]byte{'\n'})
	mac.Write([]byte(strconv.FormatInt(expiryUnix, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Query builds `e=...&s=...`; returns empty when unsigned or hitting a public prefix.
func Query(secret, key string, ttl time.Duration, publicPrefixes []string) string {
	if secret == "" || key == "" || IsPublic(publicPrefixes, key) {
		return ""
	}
	if ttl <= 0 {
		ttl = 6 * time.Hour
	}
	exp := time.Now().Add(ttl).Unix()
	sig := Sign(secret, key, exp)
	if sig == "" {
		return ""
	}
	return ExpiryParam + "=" + strconv.FormatInt(exp, 10) + "&" + ValueParam + "=" + sig
}

// Verify checks the signature and validity window.
func Verify(secret, key, expiry, sig string) bool {
	if secret == "" || key == "" || expiry == "" || sig == "" {
		return false
	}
	exp, err := strconv.ParseInt(expiry, 10, 64)
	if err != nil || exp <= 0 || time.Now().Unix() > exp {
		return false
	}
	want := Sign(secret, key, exp)
	if want == "" {
		return false
	}
	return hmac.Equal([]byte(want), []byte(sig))
}

// IsPublic reports keys always readable anonymously (splash images, etc.).
func IsPublic(prefixes []string, key string) bool {
	if key == "" {
		return false
	}
	for _, p := range prefixes {
		p = strings.TrimSpace(p)
		if p != "" && strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

// IsPlainMD5 reports whether an ETag is the object MD5; multipart uploads are
// `<hash>-<parts>` and cannot be compared directly.
func IsPlainMD5(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}
