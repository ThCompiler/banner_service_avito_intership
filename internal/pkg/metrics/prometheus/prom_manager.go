package prometheus

import (
	"context"
	"time"

	"bannersrv/internal/pkg/metrics"

	"github.com/ThCompiler/sdi"
	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	ServiceName string
}

const (
	defaultHistogramMaxBucketNumber  = 100
	defaultHistogramMinResetDuration = 100 * time.Millisecond
	defaultHistogramMaxZeroThreshold = 120
)

type MetricsManager struct {
	HitsSuccess   *prometheus.CounterVec
	HitsErrors    *prometheus.CounterVec
	ExecutionTime *prometheus.HistogramVec
	TotalHits     prometheus.Counter
}

func NewPrometheusMetrics(serviceName string) *MetricsManager {
	mm := &MetricsManager{
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
			NativeHistogramMaxBucketNumber:  defaultHistogramMaxBucketNumber,
			NativeHistogramMinResetDuration: defaultHistogramMinResetDuration,
			NativeHistogramMaxZeroThreshold: defaultHistogramMaxZeroThreshold,
		}, []string{"status", "path", "method"}),
		TotalHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: serviceName + "_total_hits",
		}),
	}

	return mm
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

	return prometheus.Register(mm.TotalHits)
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

func NewManager(cfg Config) (metrics.Manager, error) {
	manager := NewPrometheusMetrics(cfg.ServiceName)
	if err := manager.SetupMonitoring(); err != nil {
		return nil, err
	}

	return manager, nil
}

func NewProvider() sdi.Provider[metrics.Manager, Config] {
	return sdi.ProviderFuncNoClean(func(_ context.Context, cfg Config) (metrics.Manager, error) {
		return NewManager(cfg)
	})
}
