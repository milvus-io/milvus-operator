package controllers

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// same as requestLatency in
	// "sigs.k8s.io/controller-runtime/pkg/metrics"
	// its not exported, yet we want to remove it
	// so we copy it here
	requestLatencyCollector = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: metrics.RestClientSubsystem,
		Name:      metrics.LatencyKey,
		Help:      "Request latency in seconds. Broken down by verb and URL.",
		Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
	}, []string{"verb", "url"})

	// metrics to register
	milvusTotalCollector = prometheus.NewGauge(prometheus.GaugeOpts{
		Subsystem: "milvus",
		Name:      "total_count",
		Help:      "Number of milvus CRDs",
	})

	milvusUnhealthyCollector = prometheus.NewGauge(prometheus.GaugeOpts{
		Subsystem: "milvus",
		Name:      "unhealthy",
		Help:      "Number of unhealthy milvus CRDs",
	})
)

// InitializeMetrics for controllers
func InitializeMetrics() {
	// unregister the defaults which we don't need
	metrics.Registry.Unregister(requestLatencyCollector)
	// register our own
	metrics.Registry.MustRegister(milvusTotalCollector)
	metrics.Registry.MustRegister(milvusUnhealthyCollector)
}
