package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	METHOD_LABEL = "method"
)

type MetricManager struct {
	loginCounter *prometheus.CounterVec
}

func NewMetricManager() MetricManager {
	m := MetricManager{}
	m.loginCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			// fully qualified name of the metric, only Name mandatory
			Namespace: "namespace",
			Subsystem: "subsystem",
			Name:      "login_count",

			// Metrics with same FQN have same help
			Help: "Total login requests handled by http server",
		},
		[]string{METHOD_LABEL},
	)

	// register metrics
	prometheus.DefaultRegisterer.MustRegister(m.loginCounter)

	return m
}

func (m *MetricManager) IncGetLoginCount() {
	m.loginCounter.With(prometheus.Labels{METHOD_LABEL: "GET"}).Inc()
}

func (m *MetricManager) IncPostLoginCount() {
	m.loginCounter.With(prometheus.Labels{METHOD_LABEL: "POST"}).Inc()
}
