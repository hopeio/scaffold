/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"

	logx "github.com/hopeio/gox/log"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AccessHook logs every Redis command in the same shape as GORM SQL access logs:
// elapsedms / cmd / rows / caller / trace_id / span_id.
type AccessHook struct {
	*zap.Logger
	// SlowThreshold, when > 0, upgrades successful cmds slower than this to Warn
	// (message "SLOW CMD >= …"); normal cmds stay Info. Not a filter — all cmds log.
	SlowThreshold time.Duration
}

// NewAccessHook builds a hook; pass a logger without caller (caller is computed here).
func NewAccessHook(loger *zap.Logger) *AccessHook {
	if loger == nil {
		loger = zap.NewNop()
	}
	return &AccessHook{Logger: loger.With(zap.String("component", "redis"))}
}

// DialHook is a pass-through; connect noise is not useful as access logs.
func (h *AccessHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

// ProcessHook times and logs a single command after it finishes.
func (h *AccessHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		begin := time.Now()
		err := next(ctx, cmd)
		rows := int64(1)
		if errors.Is(err, redis.Nil) {
			rows = 0
		}
		h.trace(ctx, begin, formatCmd(cmd), rows, err)
		return err
	}
}

// ProcessPipelineHook times and logs a pipeline as one access line.
func (h *AccessHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		begin := time.Now()
		err := next(ctx, cmds)
		parts := make([]string, 0, len(cmds))
		for _, c := range cmds {
			parts = append(parts, formatCmd(c))
		}
		h.trace(ctx, begin, strings.Join(parts, "; "), int64(len(cmds)), err)
		return err
	}
}

func (h *AccessHook) trace(ctx context.Context, begin time.Time, cmd string, rows int64, err error) {
	elapsed := time.Since(begin)
	level := zapcore.InfoLevel
	var msg string
	switch {
	case err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)):
		// Client disconnect / deadline: not a Redis fault.
		msg = err.Error()
	case err != nil && errors.Is(err, redis.Nil):
		// Key miss is a normal outcome for GET-like cmds.
	case err != nil:
		level = zapcore.ErrorLevel
		msg = err.Error()
	case h.SlowThreshold > 0 && elapsed > h.SlowThreshold:
		level = zapcore.WarnLevel
		msg = fmt.Sprintf("SLOW CMD >= %v", h.SlowThreshold)
	}
	fields := []zap.Field{
		zap.Float64("elapsedms", float64(elapsed.Nanoseconds())/1e6),
		zap.String("cmd", cmd),
		zap.Int64("rows", rows),
		zap.String("caller", fileWithLineNum()),
		logx.Context(ctx),
	}
	if ce := h.Check(level, msg); ce != nil {
		ce.Write(fields...)
	}
}

func formatCmd(cmd redis.Cmder) string {
	args := cmd.Args()
	if len(args) == 0 {
		return cmd.Name()
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = fmt.Sprint(a)
	}
	return strings.Join(parts, " ")
}

func fileWithLineNum() string {
	for i := 2; i < 24; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if strings.Contains(file, "/redis/go-redis/") ||
			strings.Contains(file, "/hopeio/scaffold/redis/") ||
			strings.Contains(file, "/hopeio/gox/database/redis/") ||
			strings.HasSuffix(file, "/runtime/asm_arm64.s") ||
			strings.HasSuffix(file, "/runtime/asm_amd64.s") {
			continue
		}
		return file + ":" + strconv.FormatInt(int64(line), 10)
	}
	return ""
}
