package utils

import "code.byted.org/gopkg/metrics"

type Metricser interface {
	EmitCounter(name string, count int, tags map[string]string)
	EmitStore(name string, value int, tags map[string]string)
	EmitTimer(name string, cost int, tags map[string]string)
}

type DefaultMetricser struct {
	*metrics.MetricsClient
}

func (m *DefaultMetricser) EmitCounter(name string, count int, tags map[string]string) {
	m.MetricsClient.EmitCounter(name, count, "", tags)
}

func (m *DefaultMetricser) EmitStore(name string, value int, tags map[string]string) {
	m.MetricsClient.EmitStore(name, value, "", tags)

}

func (m *DefaultMetricser) EmitTimer(name string, cost int, tags map[string]string) {
	m.MetricsClient.EmitTimer(name, cost, "", tags)
}

func NewDefaultMetricser() Metricser {
	m := metrics.NewDefaultMetricsClient("toutiao.microservice.tsad", true)
	return &DefaultMetricser{m}
}

func NewMetricser(prefix string) Metricser {
	m := metrics.NewDefaultMetricsClient(prefix, true)
	return &DefaultMetricser{m}
}

var defaultMetricser = NewDefaultMetricser()

func EmitCounter(name string, count int, tags map[string]string) {
	defaultMetricser.EmitStore(name, count, tags)
}

func EmitStore(name string, value int, tags map[string]string) {
	defaultMetricser.EmitStore(name, value, tags)
}

func EmitTimer(name string, cost int, tags map[string]string) {
	defaultMetricser.EmitTimer(name, cost, tags)
}