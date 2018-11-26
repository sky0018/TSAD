package tsfetcher

import (
	"context"
	"time"

	"code.byted.org/microservice/tsad/worker/ts"
)

// SourceType .
type SourceType string

const (
	// SourceTSDB .
	SourceTSDB SourceType = "TSDB"
)

// Source .
type Source struct {
	Type  SourceType
	Key   string
	Extra string
}

// TSFetcher .
type TSFetcher interface {
	Fetch(ctx context.Context, source Source, begin, end time.Time) (ts.TS, error)
}
