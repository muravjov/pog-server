package util

import (
	"github.com/prometheus/client_golang/prometheus"
)

// :TRICKY:
// New*Metric() functions should be used before main() is invoked
// if you need to create a metric on the fly just register
// the metric explicitely, e.g. via tryRegisterMetric()

var appMetricList []prometheus.Collector

func TryRegisterAppMetrics(r prometheus.Registerer) {
	for _, c := range appMetricList {
		TryRegisterMetric(r, c)
	}
}

func NewCounterVec(name, help string, labelNames []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labelNames,
	)
}

func NewCounterVecMetric(name, help string, labelNames []string) *prometheus.CounterVec {
	metric := NewCounterVec(name, help, labelNames)
	appMetricList = append(appMetricList, metric)

	return metric
}

func MakeCounterVecFunc(name, help string) func(name string, cnt int) {
	metric := NewCounterVecMetric(
		name,
		help,
		[]string{"name"},
	)
	return func(name string, cnt int) {
		metric.With(prometheus.Labels{"name": name}).Add(float64(cnt))
	}
}

func NewGaugeVec(name, help string, labelNames []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	}, labelNames)
}

func NewGaugeVecMetric(name, help string, labelNames []string) *prometheus.GaugeVec {
	metric := NewGaugeVec(name, help, labelNames)
	appMetricList = append(appMetricList, metric)

	return metric
}

func TryRegisterMetric(r prometheus.Registerer, c prometheus.Collector) bool {
	if err := r.Register(c); err != nil {
		Error(err)
		return false
	}
	return true
}

func MakeDefaultObjectives() map[float64]float64 {
	return map[float64]float64{
		0.5:  0.05,  // ±0.05 error
		0.9:  0.01,  // ±0.01 error
		0.99: 0.001, // ±0.001 error
	}
}

func newSummaryVecWithObjectives(name, help string, labelNames []string, objectives map[float64]float64) *prometheus.SummaryVec {
	if objectives == nil {
		objectives = MakeDefaultObjectives()
	}

	return prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name:       name,
		Help:       help,
		Objectives: objectives,
	}, labelNames)
}

func NewSummaryVecWithObjectivesMetric(name, help string, labelNames []string, objectives map[float64]float64) *prometheus.SummaryVec {
	metric := newSummaryVecWithObjectives(name, help, labelNames, objectives)
	appMetricList = append(appMetricList, metric)

	return metric
}

func SpreadObjectives() map[float64]float64 {
	return map[float64]float64{
		0.01: 0.001, // ±0.001 error
		0.5:  0.05,  // ±0.05 error
		0.99: 0.001, // ±0.001 error
	}
}
