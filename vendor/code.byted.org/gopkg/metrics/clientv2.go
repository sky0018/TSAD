package metrics

import (
	"sync"
)

type MetricsClientV2 struct {
	mu sync.RWMutex

	server  string
	prefix  string
	nocheck bool

	sender *sender

	metrictypes map[string]metricsType

	cache *singleMetricCache
}

var (
	clients   map[string]*MetricsClientV2
	clientsMu sync.Mutex
)

func init() {
	clients = make(map[string]*MetricsClientV2)
}

func NewMetricsClientV2(server, prefix string, nocheck bool) *MetricsClientV2 {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if clients[prefix] != nil {
		return clients[prefix]
	}
	cli := &MetricsClientV2{server: server, prefix: prefix, nocheck: nocheck}
	cli.metrictypes = make(map[string]metricsType)
	cli.sender = newSender(server)
	cli.cache = newSingleMetricCache()
	clients[prefix] = cli
	return cli
}

func NewDefaultMetricsClientV2(prefix string, nocheck bool) *MetricsClientV2 {
	return NewMetricsClientV2(DefaultMetricsServer, prefix, nocheck)
}

func (m *MetricsClientV2) DefineCounter(name string) error {
	return m.defineMetrics(name, metricsTypeCounter)
}

func (m *MetricsClientV2) DefineTimer(name string) error {
	return m.defineMetrics(name, metricsTypeTimer)
}

func (m *MetricsClientV2) DefineStore(name string) error {
	return m.defineMetrics(name, metricsTypeStore)
}

func (m *MetricsClientV2) defineMetrics(name string, mt metricsType) error {
	if m.nocheck {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.metrictypes[name]
	if !ok {
		m.metrictypes[name] = mt
	}
	if mt != t {
		return ErrDuplicatedMetrics
	}
	return nil
}

func (m *MetricsClientV2) EmitCounter(name string, value interface{}, tags ...T) error {
	return m.emit(metricsTypeCounter, name, value, tags...)
}

func (m *MetricsClientV2) EmitTimer(name string, value interface{}, tags ...T) error {
	return m.emit(metricsTypeTimer, name, value, tags...)
}

func (m *MetricsClientV2) EmitStore(name string, value interface{}, tags ...T) error {
	return m.emit(metricsTypeStore, name, value, tags...)
}

func (m *MetricsClientV2) emit(mt metricsType, name string, value interface{}, tags ...T) error {
	if !m.nocheck {
		m.mu.RLock()
		t, ok := m.metrictypes[name]
		m.mu.RUnlock()
		if !ok {
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
	sm := m.cache.Get(name, tags)
	sm.mt = mt.String()
	sm.prefix = m.prefix
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

type singleMetricCache struct {
	mu sync.RWMutex
	m  map[uint64]singleMetric
}

func newSingleMetricCache() *singleMetricCache {
	return &singleMetricCache{m: make(map[uint64]singleMetric)}
}

func hashkey(name string, tags []T) uint64 {
	// fork: https://golang.org/src/hash/fnv/fnv.go
	h := uint64(14695981039346656037)
	for _, c := range Bytes(name) {
		h *= uint64(1099511628211)
		h ^= uint64(c)
	}
	for _, t := range tags {
		for _, c := range Bytes(t.Name) {
			h *= uint64(1099511628211)
			h ^= uint64(c)
		}
		for _, c := range Bytes(t.Value) {
			h *= uint64(1099511628211)
			h ^= uint64(c)
		}
	}
	return h
}

func (c *singleMetricCache) Get(name string, tags []T) singleMetric {
	SortTags(tags)
	h := hashkey(name, tags)
	c.mu.RLock()
	m, ok := c.m[h]
	c.mu.RUnlock()
	if ok && m.name == name && TagsEqual(m.tags, tags) {
		return m
	}
	m.name = name
	m.tags = make([]T, len(tags))
	copy(m.tags, tags) // fix tags escape to heap
	m.SetTags(tags, extTags)

	c.mu.Lock()
	if len(c.m) > 100000 { // memory leak due to too many tags ?
		c.m = make(map[uint64]singleMetric)
	}
	c.m[h] = m
	c.mu.Unlock()

	return m
}
