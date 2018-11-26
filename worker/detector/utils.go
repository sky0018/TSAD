package detector

import (
	"context"
)

func Done(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func getInt(confMap map[string]interface{}, key string, defaultVal int) int {
	if confMap == nil {
		return defaultVal
	}
	v, ok := confMap[key]
	if !ok {
		return defaultVal
	}

	switch v.(type) {
	case int:
		return v.(int)
	case int64:
		return int(v.(int64))
	case int32:
		return int(v.(int32))
	}
	return defaultVal
}

func getFloat64(confMap map[string]interface{}, key string, defaultVal float64) float64 {
	if confMap == nil {
		return defaultVal
	}
	v, ok := confMap[key]
	if !ok {
		return defaultVal
	}

	switch v.(type) {
	case float64:
		return v.(float64)
	case float32:
		return float64(v.(float32))
	}
	return defaultVal
}