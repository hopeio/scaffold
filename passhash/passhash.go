package passhash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonPrefix = "$argon2id$"
	argonTime   = 2
	argonMemory = 19 * 1024 // 19 MiB，OWASP 交互式推荐
	argonThreads = 1
	argonKeyLen = 32
	argonSaltLen = 16
)

// Hash 新密码用 Argon2id；pepper 来自配置 PassSalt。
func Hash(password, pepper string) string {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		// 退化：仍返回不可猜测串，避免注册因 CSPRNG 失败而 panic。
		copy(salt, []byte(pepper))
		for len(fmt.Sprintf("%x", salt)) < argonSaltLen {
			salt = append(salt, pepper...)
		}
		salt = salt[:argonSaltLen]
	}
	key := argon2.IDKey([]byte(password+pepper), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("%s$v=19$m=%d,t=%d,p=%d$%s$%s",
		strings.TrimSuffix(argonPrefix, "$"),
		argonMemory, argonTime, argonThreads,
		b64.EncodeToString(salt),
		b64.EncodeToString(key),
	)
}

// Verify 校验明文，仅支持 Argon2id（无存量用户，不保留旧格式兼容）。
func Verify(password, stored, pepper string) bool {
	if password == "" || stored == "" {
		return false
	}
	if strings.HasPrefix(stored, argonPrefix) {
		return verifyArgon(password, stored, pepper)
	}
	return false
}

func verifyArgon(password, stored, pepper string) bool {
	salt, want, time, memory, threads, ok := parseArgon(stored)
	if !ok {
		return false
	}
	got := argon2.IDKey([]byte(password+pepper), salt, time, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func parseArgon(stored string) (salt, key []byte, time, memory uint32, threads uint8, ok bool) {
	// $argon2id$v=19$m=19456,t=2,p=1$salt$key
	parts := strings.Split(stored, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, 0, 0, 0, false
	}
	var t, m uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return nil, nil, 0, 0, 0, false
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return nil, nil, 0, 0, 0, false
	}
	key, err = b64.DecodeString(parts[5])
	if err != nil {
		return nil, nil, 0, 0, 0, false
	}
	return salt, key, t, m, p, true
}

