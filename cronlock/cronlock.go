// Package cronlock makes a periodic job run on a single instance.
//
// The lock is a plain SET NX PX key: if an instance dies mid-run the TTL expires and
// the next tick can take over. Release compares the token, so a run that overruns its
// TTL cannot delete a lock that already belongs to another instance.
package cronlock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

const releaseScript = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`

// Try acquires key for ttl. ok=false means another instance owns it; the returned
// release is a no-op then, so callers can call it unconditionally.
//
// ttl must cover one run plus some slack: a job that runs longer than ttl lets another
// instance start the same work.
func Try(ctx context.Context, rdb redis.Cmdable, key string, ttl time.Duration) (func(), bool) {
	if rdb == nil || key == "" || ttl <= 0 {
		return func() {}, false
	}
	token := newToken()
	acquired, err := rdb.SetNX(ctx, key, token, ttl).Result()
	if err != nil || !acquired {
		return func() {}, false
	}
	return func() {
		// Background context: the caller's ctx is often the job's own, already
		// cancelled by the time the run ends.
		rdb.Eval(context.Background(), releaseScript, []string{key}, token)
	}, true
}

func newToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().Format(time.RFC3339Nano)
	}
	return hex.EncodeToString(b[:])
}
