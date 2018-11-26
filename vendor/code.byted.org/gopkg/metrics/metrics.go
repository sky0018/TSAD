// Package metrics provides a goroutine safe metrics client package metrics
// if TCE_HOST_IP is setted, will use this env value as host address
package metrics

import (
	"errors"
	"os"
	"strings"
	"sync"
)

type metricsType int

const (
	metricsTypeCounter metricsType = iota
	metricsTypeTimer
	metricsTypeStore
)

func (t metricsType) String() string {
	switch t {
	case metricsTypeCounter:
		return "counter"
	case metricsTypeStore:
		return "store"
	case metricsTypeTimer:
		return "timer"
	}
	return "unknown"
}

const (
	BlackholeAddr = "blackhole"
)

var (
	DefaultMetricsSock   = "/tmp/metric.sock"
	DefaultMetricsServer = "127.0.0.1:9123"

	ErrDuplicatedMetrics    = errors.New("duplicated metrics name")
	ErrEmitUndefinedMetrics = errors.New("emit undefined metrics")
	ErrEmitBadMetricsType   = errors.New("emit bad metrics type")
	ErrUnKnowValue          = errors.New("Unkown metrics value")
	ErrEmitTimeout          = errors.New("Emit metrics timeout")

	extTags = make([]byte, 0, 4096)
)

func AddGlobalTag(name, value string) {
	if name == "" || value == "" {
		return
	}
	if len(extTags) > 0 {
		extTags = append(extTags, '|')
	}
	extTags = append(extTags, name...)
	extTags = append(extTags, '=')
	extTags = append(extTags, value...)
}

func init() {
	if host := strings.TrimSpace(os.Getenv("TCE_HOST_IP")); host != "" {
		// FOR TCE ENV
		DefaultMetricsServer = host + ":9123"
		AddGlobalTag("env_type", "tce")
		AddGlobalTag("pod_name", os.Getenv("MY_POD_NAME"))
		AddGlobalTag("_psm", os.Getenv("TCE_PSM"))
		AddGlobalTag("service_env", os.Getenv("SERVICE_ENV"))
	}
}

type MetricsClient struct {
	mu sync.RWMutex

	NamespacePrefix string
	AllMetrics      map[string]map[string]metricsType
	Server          string
	IgnoreCheck     bool

	sender *sender

	cache *singleMetricCache
}

func NewMetricsClient(server, namespacePrefix string, ignoreCheck bool) *MetricsClient {
	client := &MetricsClient{
		NamespacePrefix: namespacePrefix,
		AllMetrics:      make(map[string]map[string]metricsType),
		Server:          server,
		IgnoreCheck:     ignoreCheck,
		sender:          newSender(server),
		cache:           newSingleMetricCache(),
	}
	return client
}

func NewDefaultMetricsClient(namespacePrefix string, ignoreCheck bool) *MetricsClient {
	return NewMetricsClient(DefaultMetricsServer, namespacePrefix, ignoreCheck)
}

func (mc *MetricsClient) DefineCounter(name, prefix string) error {
	return mc.defineMetrics(name, prefix, metricsTypeCounter)
}

func (mc *MetricsClient) DefineTimer(name, prefix string) error {
	return mc.defineMetrics(name, prefix, metricsTypeTimer)
}

func (mc *MetricsClient) DefineStore(name, prefix string) error {
	return mc.defineMetrics(name, prefix, metricsTypeStore)
}

func (mc *MetricsClient) defineMetrics(name, prefix string, mt metricsType) error {
	// mc.IgnoreCheck won't be modified, not need lock.
	if mc.IgnoreCheck {
		return nil
	}
	if len(prefix) == 0 {
		prefix = mc.NamespacePrefix
	}
	mc.mu.Lock()
	defer mc.mu.Unlock()
	m := mc.AllMetrics[prefix]
	if m == nil {
		m = make(map[string]metricsType)
		mc.AllMetrics[prefix] = m
	}
	t, ok := m[name]
	if !ok {
		m[name] = mt
		return nil
	}
	if mt != t {
		return ErrDuplicatedMetrics
	}
	return nil
}

func (mc *MetricsClient) EmitCounter(name string, value interface{}, prefix string, tagkv map[string]string) error {
	return mc.emit(metricsTypeCounter, name, value, prefix, tagkv)
}

func (mc *MetricsClient) EmitTimer(name string, value interface{}, prefix string, tagkv map[string]string) error {
	return mc.emit(metricsTypeTimer, name, value, prefix, tagkv)
}

func (mc *MetricsClient) EmitStore(name string, value interface{}, prefix string, tagkv map[string]string) error {
	return mc.emit(metricsTypeStore, name, value, prefix, tagkv)
}

func (m *MetricsClient) emit(mt metricsType, name string, value interface{},
	prefix string, tagkv map[string]string) error {
	if len(prefix) == 0 {
		prefix = m.NamespacePrefix
	}
	if !m.IgnoreCheck {
		m.mu.RLock()
		types, ok1 := m.AllMetrics[prefix]
		t, ok2 := types[name] // read from nil is safe
		m.mu.RUnlock()
		if !ok1 || !ok2 {
			return ErrEmitUndefinedMetrics
		}
		if t != mt {
			return ErrEmitBadMetricsType
		}
	}
	v, err := toFloat64(value)
	if err != nil {
		return err
	}
	if mt == metricsTypeCounter && v == 0 { // meaningless
		return nil
	}
	tags := make([]T, 0, len(tagkv))
	for k, v := range tagkv {
		tags = append(tags, T{Name: k, Value: v})
	}

	sm := m.cache.Get(name, tags)
	sm.mt = mt.String()
	sm.prefix = prefix
	sm.v = v
	switch mt {
	case metricsTypeCounter:
		m.sender.SendCounter(sm)
	case metricsTypeStore:
		m.sender.SendStore(sm)
	case metricsTypeTimer:
		m.sender.SendTimer(sm)
	}
	return nil
}

// If you use the default metricsClient, then the NamespacePrefix is "",
// so you can fill in "prefix" when using DefineCounter, DefineTimer etc.
// and EmitCounter, EmitTimer etc.
// default metrics client won't ignore metrics check.
var metricsClient = NewDefaultMetricsClient("", false)

func DefineCounter(name, prefix string) error {
	return metricsClient.DefineCounter(name, prefix)
}

func DefineTimer(name, prefix string) error {
	return metricsClient.DefineStore(name, prefix)
}

func DefineStore(name, prefix string) error {
	return metricsClient.DefineStore(name, prefix)
}

func EmitCounter(name string, value interface{}, prefix string, tagkv map[string]string) error {
	return metricsClient.EmitCounter(name, value, prefix, tagkv)
}

func EmitTimer(name string, value interface{}, prefix string, tagkv map[string]string) error {
	return metricsClient.EmitTimer(name, value, prefix, tagkv)
}

func EmitStore(name string, value interface{}, prefix string, tagkv map[string]string) error {
	return metricsClient.EmitStore(name, value, prefix, tagkv)
}
