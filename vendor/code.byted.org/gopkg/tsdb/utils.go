package tsdb

import (
	"fmt"
	"time"

	"code.byted.org/gopkg/metrics"
)

var metricsCli = metrics.NewDefaultMetricsClient("tsdb_query", true)

// TimeStamp 将time转换TSDB时间戳格式
func TimeStamp(t time.Time) string {
	pattern := "%02d/%02d/%02d-%02d:%02d:%02d"
	return fmt.Sprintf(pattern, t.Year(), int(t.Month()), t.Day(),
		t.Hour(), t.Minute(), t.Second())
}

// HumanTimeStamp 将time转换TSDB时间戳格式
func HumanTimeStamp(t time.Time) string {
	pattern := "%04d-%02d-%02d_%02d:%02d:%02d"
	return fmt.Sprintf(pattern, t.Year(), int(t.Month()), t.Day(),
		t.Hour(), t.Minute(), t.Second())
}
