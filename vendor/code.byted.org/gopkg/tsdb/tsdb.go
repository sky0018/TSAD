package tsdb

import "context"

// DefaultClient .
var DefaultClient *Client

func init() {
	DefaultClient, _ = NewClient(&Options{
		MaxConcurrency:     DefaultMaxConcurrency,
		DefaultTimeoutInMs: DefaultTimeoutInMs,
		DefaultRetry:       DefaultRetry,
		Logger:             DefaultLogger,
	})
}

// SetDefaultClient .
func SetDefaultClient(client *Client) {
	if client == nil {
		return
	}
	DefaultClient = client
}

// Do .
func Do(ctx context.Context, url string, model interface{}) error {
	return DefaultClient.Do(ctx, url, model)
}

// Go .
func Go(ctx context.Context, url string, model interface{}) *Future {
	return DefaultClient.Go(ctx, url, model)
}
