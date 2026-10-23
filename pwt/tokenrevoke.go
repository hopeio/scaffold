// 本文件计算 JWT 吊销名单的键与存活时间（不是第二个包文档）。
//
// JWT 本身无状态，登出后签名依然有效。这里按签名段登记一条吊销记录，
// 存活到原 token 过期为止——之后 token 自然失效，记录留着也没用。
// 全端踢下线的主路径是删 Redis LoginUser 会话，本包只服务「登出当前票」场景。
//
// 逻辑不依赖 redis 与全局配置，可单独测试。
package pwt

import (
	"strings"
	"time"
)

const KeyPrefix = "user:token:revoked:"

// MaxTTL 兜底上限，防止异常大的 exp 把记录长期钉在 redis 里。
const MaxTTL = 30 * 24 * time.Hour

// Signature 取 JWT 的签名段，用它当吊销名单的键：同一 token 唯一，
// 且不会把 header/payload 里的用户信息写进 redis 键名。
// token 结构不合法时返回空串。
func Signature(token string) string {
	token = strings.TrimSpace(token)
	if strings.Count(token, ".") != 2 {
		return ""
	}
	sig := token[strings.LastIndexByte(token, '.')+1:]
	if sig == "" {
		return ""
	}
	return sig
}

func Key(signature string) string {
	return KeyPrefix + signature
}

// TTL 返回吊销记录应存活多久。token 已过期时返回 0，调用方无需写入。
func TTL(exp, now time.Time) time.Duration {
	if exp.IsZero() {
		return MaxTTL
	}
	d := exp.Sub(now)
	if d <= 0 {
		return 0
	}
	if d > MaxTTL {
		return MaxTTL
	}
	// 多留一秒，避免与 redis 过期精度和机器时钟偏差擦边。
	return d + time.Second
}
