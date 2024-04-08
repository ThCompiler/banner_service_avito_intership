package metrics

import "github.com/prometheus/client_golang/prometheus"

type Manager interface {
	SetupMonitoring() error
	GetSuccessHits() *prometheus.CounterVec
	GetErrorHits() *prometheus.CounterVec
	GetRequestCounter() prometheus.Counter
	GetExecution() *prometheus.HistogramVec
}
