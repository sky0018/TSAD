# Metrics Package

Go语言的Metrics包，实现了协程安全地往TSDB输出Metrics信息, 可以直接使用default client发送信息，也可以新建一个client用于发送数据, 建议新建client用于发送数据

FAQ：

1. 出于性能考虑，`metrics Emit`其实只是存放到内存中，当足够多了之后才会发起请求，当前`maxPendingSize`是4096

1. 使用`Emit`前进行`Define`可以有效减少粗心造成的`query key`错误。如果不想Define，在Client初始化的时候将nocheck参数设为true即可。
  **e.g.** cli := metrics.NewDefaultMetricsClientV2("toutiao.gopkg.I.Dont.Wanna.Check", true)

1. 此Metrics库不支持string作为value传入，使用前尽量转化为float64或int。

# Exmaple

```go
package main
import (
	"code.byted.org/gopkg/metrics"
)
func main() {
	// Use param server in NewMetricsClientV2 to determine metrics_server.
	// cli := NewMetricsClientV2(server, prefix string, nocheck bool) *MetricsClientV2
	// Use NewDefaultMetricsClientV2 to take default agent 127.0.0.1:9123
	cli := metrics.NewDefaultMetricsClientV2("toutiao.gopkg.example", false)

	// Counter Example:
	// EmitCounter accepts float32 float64 int int8 int16 int32 int64 uint8 uint16 uint32 uint64
	cli.DefineCounter("test.counter")
	cli.EmitCounter("test.counter", 1)
	cli.EmitCounter("test.counter", 1, metrics.T{"func", "main"})
	cli.EmitCounter("test.counter", 1,
		metrics.T{"product", "toutiao"},
		metrics.T{"app", "news_article"},
	)
	cli.EmitCounter("test.counter", 1,
		[]metrics.T{
			{"product", "toutiao"},
			{"app", "news_article"},
		}...)

	// Timer Example:
	// EmitTimer accepts float32 float64 int int8 int16 int32 int64 uint8 uint16 uint32 uint64 time.Duration
	cli.DefineTimer("test.timer")
	cli.EmitTimer("test.timer1", 1000000.1)
	cli.EmitTimer("test.timer2", 1000000)
	startTime := time.Now()
	time.Sleep(300000)
	cli.EmitTimer("test.timer3", time.Since(startTime))

	// Store Example:
	// EmitStore accepts float32 float64 int int8 int16 int32 int64 uint8 uint16 uint32 uint64
	cli.DefineStore("test.store")
	cli.EmitStore("test.store", 60000)
}
```