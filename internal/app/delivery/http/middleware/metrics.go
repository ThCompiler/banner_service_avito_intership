package middleware

import (
	"bannersrv/internal/pkg/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const successCodeThreshold = 300

func RequestMetrics(metricsManager metrics.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Stop timer
		timeStamp := time.Now()
		latency := timeStamp.Sub(start)
		statusCode := c.Writer.Status()

		path := c.FullPath()
		method := c.Request.Method

		// Save metrics
		if metricsManager == nil {
			return
		}

		metricsManager.GetRequestCounter().Inc()

		if statusCode < successCodeThreshold {
			metricsManager.GetSuccessHits().WithLabelValues(
				strconv.Itoa(statusCode),
				path,
				method,
			).Inc()
		} else {
			metricsManager.GetErrorHits().WithLabelValues(
				strconv.Itoa(statusCode),
				path,
				method,
			).Inc()
		}

		metricsManager.GetExecution().
			WithLabelValues(strconv.Itoa(statusCode), path, method).
			Observe(latency.Seconds())
	}
}
