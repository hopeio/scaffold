// Package uploadsign 为 /upload/ 链接做 HMAC 签名与校验。
// 独立于 global：那个包 init 会连库，纯逻辑放这里才可测。
package uploadsign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"
	"time"
)

// 查询参数：e=过期时间戳（秒），s=签名。
const (
	ExpiryParam = "e"
	ValueParam  = "s"
)

// Sign 对存储 key 与过期时间签名；secret 为空表示不签名。
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

// Query 生成 `e=...&s=...`；无密钥或命中公开前缀时返回空串。
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

// Verify 校验签名与有效期。
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

// IsPublic 命中前缀的对象始终匿名可读（开屏图等）。
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

// IsPlainMD5 判断 ETag 是否就是对象 MD5；分片上传是 `<hash>-<parts>`，不能直接比对。
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
