package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type MetricsManager struct {
	HitsSuccess   *prometheus.CounterVec
	HitsErrors    *prometheus.CounterVec
	ExecutionTime *prometheus.HistogramVec
	TotalHits     prometheus.Counter
}

func NewPrometheusMetrics(serviceName string) *MetricsManager {
	metrics := &MetricsManager{
		HitsSuccess: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: serviceName + "_success_hits",
			Help: "Count success responses from service",
		}, []string{"status", "path", "method"}),
		HitsErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: serviceName + "_errors_hits",
			Help: "Count errors response from service",
		}, []string{"status", "path", "method"}),
		ExecutionTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:                            serviceName + "_durations",
			Help:                            "Duration execution of request",
			Buckets:                         prometheus.DefBuckets,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: 100 * time.Millisecond,
			NativeHistogramMaxZeroThreshold: 120,
		}, []string{"status", "path", "method"}),
		TotalHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: serviceName + "_total_hits",
		}),
	}

	return metrics
}
func (mm *MetricsManager) SetupMonitoring() error {
	if err := prometheus.Register(mm.HitsErrors); err != nil {
		return err
	}
	if err := prometheus.Register(mm.HitsSuccess); err != nil {
		return err
	}
	if err := prometheus.Register(mm.ExecutionTime); err != nil {
		return err
	}
	if err := prometheus.Register(mm.TotalHits); err != nil {
		return err
	}
	return nil
}
func (mm *MetricsManager) GetSuccessHits() *prometheus.CounterVec {
	return mm.HitsSuccess
}
func (mm *MetricsManager) GetErrorHits() *prometheus.CounterVec {
	return mm.HitsErrors
}
func (mm *MetricsManager) GetRequestCounter() prometheus.Counter {
	return mm.TotalHits
}
func (mm *MetricsManager) GetExecution() *prometheus.HistogramVec {
	return mm.ExecutionTime
}
