/*
Example:

if err := stats.DoReport("example"); err != nil{
    fmt.Fprintf(os.Stderr, "DoReport error: %s\n", err)
    os.Exit(-1)
}
*/
package stats

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"code.byted.org/gopkg/metrics"
)

type StatItem struct {
	runtime.MemStats
	Goroutines int64
	NumGC      int64
	GCPauseUs  uint64
}

func TickStat(d time.Duration) <-chan StatItem {
	ret := make(chan StatItem)
	go func() {
		m0 := ReadMemStats()
		if d < time.Second {
			d = time.Second
		}
		for {
			// use time.After for chan blocking issue
			<-time.After(d)
			m1 := ReadMemStats()
			ret <- StatItem{
				MemStats:   m1,
				Goroutines: int64(runtime.NumGoroutine()),
				NumGC:      int64(m1.NumGC - m0.NumGC),
				GCPauseUs:  GCPauseNs(m1, m0) / 1000,
			}
			m0 = m1
		}
	}()
	return ret
}

func ReadMemStats() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// GCPauseNs cals max(set(new.PauseNs) - set(old.PauseNs))
func GCPauseNs(new runtime.MemStats, old runtime.MemStats) uint64 {
	if new.NumGC <= old.NumGC {
		return new.PauseNs[(new.NumGC+255)%256]
	}
	n := new.NumGC - old.NumGC
	if n > 256 {
		n = 256
	}
	// max PauseNs since last GC
	var maxPauseNs uint64
	for i := uint32(0); i < n; i++ {
		if pauseNs := new.PauseNs[(new.NumGC-i+255)%256]; pauseNs > maxPauseNs {
			maxPauseNs = pauseNs
		}
	}
	return maxPauseNs
}

// GetField is util for emit metrics
func (e StatItem) GetField(key string) interface{} {
	switch key {
	case "HeapAlloc":
		return e.HeapAlloc
	case "StackInuse":
		return e.StackInuse
	case "NumGC":
		return e.NumGC
	case "Goroutines":
		return e.Goroutines
	case "TotalAlloc":
		return e.TotalAlloc
	case "Mallocs":
		return e.Mallocs
	case "Frees":
		return e.Frees
	case "HeapObjects":
		return e.HeapObjects
	case "GCCPUFraction":
		return e.GCCPUFraction
	case "GCPauseUs":
		return e.GCPauseUs
	}
	return nil
}

const (
	MetricsPrefix = "go"
)

type Reporter struct {
	name          string
	metricsClient *metrics.MetricsClient
}

func DoReport(name string) error {
	r := &Reporter{name: name}
	r.metricsClient = metrics.NewDefaultMetricsClient(MetricsPrefix, false)
	r.metricsClient.DefineStore(name+".heap.byte", "")
	r.metricsClient.DefineStore(name+".stack.byte", "")
	r.metricsClient.DefineStore(name+".numGcs", "")
	r.metricsClient.DefineStore(name+".numGos", "")
	r.metricsClient.DefineStore(name+".malloc", "")
	r.metricsClient.DefineStore(name+".free", "")
	r.metricsClient.DefineStore(name+".totalAllocated.byte", "")
	r.metricsClient.DefineStore(name+".objects", "")
	r.metricsClient.DefineTimer(name+".gcPause.us", "")
	r.metricsClient.DefineTimer(name+".gcCPU", "")
	go r.reporting()
	return nil
}

func (r *Reporter) reporting() {
	var s StatItem
	AllStoreMetrics := map[string]string{
		r.name + ".heap.byte":           "HeapAlloc",
		r.name + ".stack.byte":          "StackInuse",
		r.name + ".numGcs":              "NumGC",
		r.name + ".numGos":              "Goroutines",
		r.name + ".malloc":              "Mallocs",
		r.name + ".free":                "Frees",
		r.name + ".totalAllocated.byte": "TotalAlloc",
		r.name + ".objects":             "HeapObjects",
	}
	for _, f := range AllStoreMetrics {
		if s.GetField(f) == nil {
			panic(f + "not found")
		}
	}
	AllTimerMetrics := map[string]string{
		r.name + ".gcPause.us": "GCPauseUs",
		r.name + ".gcCPU":      "GCCPUFraction",
	}
	for _, f := range AllTimerMetrics {
		if s.GetField(f) == nil {
			panic(f + "not found")
		}
	}
	for e := range TickStat(time.Second) {
		for k, f := range AllStoreMetrics {
			err := r.metricsClient.EmitStore(k, e.GetField(f), "", nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "emit %s error: %s\n", k, err)
			}
		}
		for k, f := range AllTimerMetrics {
			err := r.metricsClient.EmitTimer(k, e.GetField(f), "", nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "emit %s error: %s\n", k, err)
			}
		}
	}
}
