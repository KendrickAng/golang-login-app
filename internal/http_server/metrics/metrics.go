package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type MetricManager struct {
	loginCounter prometheus.Counter
}

func NewMetricManager() MetricManager {
	m := MetricManager{}
	m.loginCounter = prometheus.NewCounter(prometheus.CounterOpts{
		// fully qualified name of the metric, only Name mandatory
		Namespace: "namespace",
		Subsystem: "subsystem",
		Name:      "login_count",

		// Metrics with same FQN have same help
		Help: "Total login requests handled by http server",
	})

	// register metrics
	prometheus.DefaultRegisterer.MustRegister(m.loginCounter)

	return m
}

func (m *MetricManager) IncLoginCount() {
	m.loginCounter.Inc()
}
