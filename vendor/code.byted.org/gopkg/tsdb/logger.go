package tsdb

import (
	"fmt"
	"os"
)

// Logger .
type Logger interface {
	Errorf(format string, args ...interface{})
}

type defaultLogger struct{}

func (dl *defaultLogger) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// DefaultLogger .
var DefaultLogger = new(defaultLogger)
