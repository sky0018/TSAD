package tsfetcher

import (
	"context"
	"time"
)

var (
	ctxRetry   = struct{}{}
	ctxTimeout = struct{}{}
)

// WithRetry .
func WithRetry(ctx context.Context, retry int) context.Context {
	return context.WithValue(ctx, ctxRetry, retry)
}

// WithTimeout .
func WithTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, ctxTimeout, timeout)
}

// GetRetry .
func GetRetry(ctx context.Context) (int, bool) {
	v := ctx.Value(ctxRetry)
	if v != nil {
		if retry, ok := v.(int); ok {
			return retry, true
		}
	}
	return 0, false
}

// GetTimeout .
func GetTimeout(ctx context.Context) (time.Duration, bool) {
	v := ctx.Value(ctxTimeout)
	if v != nil {
		if timeout, ok := v.(time.Duration); ok {
			return timeout, false
		}
	}
	return 0, false
}
