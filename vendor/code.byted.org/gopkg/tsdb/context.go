package tsdb

import (
	"context"
	"time"
)

const (
	keyTimeout       = "_tsdb_timeout"
	keyRetry         = "_tsdb_retry"
	keyMetrics       = "_tsdb_metrics"
	keyMetricsTags   = "_tsdb_metrics_tags"
	keyRetryInterval = "_tsdb_retry_interval"
)

// WithMetricsTags .
func WithMetricsTags(ctx context.Context, tags map[string]string) context.Context {
	return context.WithValue(ctx, keyMetricsTags, tags)
}

// GetMetricsTags .
func GetMetricsTags(ctx context.Context) (map[string]string, bool) {
	if v := ctx.Value(keyMetricsTags); v != nil {
		if t, ok := v.(map[string]string); ok {
			return t, ok
		}
	}

	return nil, false
}

// WithMetrics .
func WithMetrics(ctx context.Context, metrics string) context.Context {
	return context.WithValue(ctx, keyMetrics, metrics)
}

// GetMetrics .
func GetMetrics(ctx context.Context) (string, bool) {
	if v := ctx.Value(keyMetrics); v != nil {
		if m, ok := v.(string); ok {
			return m, ok
		}
	}

	return "", false
}

// WithTimeout .
func WithTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, keyTimeout, timeout)
}

// GetTimeout .
func GetTimeout(ctx context.Context) (time.Duration, bool) {
	if v := ctx.Value(keyTimeout); v != nil {
		if t, ok := v.(time.Duration); ok {
			return t, ok
		}
	}
	return 0, false
}

// WithRetry .
func WithRetry(ctx context.Context, retry int) context.Context {
	return context.WithValue(ctx, keyRetry, retry)
}

// GetRetry .
func GetRetry(ctx context.Context) (int, bool) {
	if v := ctx.Value(keyRetry); v != nil {
		if t, ok := v.(int); ok {
			return t, ok
		}
	}
	return 0, false
}

// WithRetryInterval .
func WithRetryInterval(ctx context.Context, retryInterval time.Duration) context.Context {
	return context.WithValue(ctx, keyRetryInterval, retryInterval)
}

// GetRetryInterval .
func GetRetryInterval(ctx context.Context) (time.Duration, bool) {
	if v := ctx.Value(keyRetryInterval); v != nil {
		if t, ok := v.(time.Duration); ok {
			return t, ok
		}
	}
	return 0, false
}
