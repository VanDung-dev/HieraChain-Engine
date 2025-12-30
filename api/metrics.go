// Package api provides Prometheus metrics for HieraChain Go Engine.
package api

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the engine.
type Metrics struct {
	// Transaction metrics
	TransactionsTotal     prometheus.Counter
	TransactionsProcessed prometheus.Counter
	TransactionsFailed    prometheus.Counter
	TransactionLatency    prometheus.Histogram

	// Batch metrics
	BatchesTotal prometheus.Counter
	BatchSize    prometheus.Histogram
	BatchLatency prometheus.Histogram

	// System metrics
	MempoolSize       prometheus.Gauge
	WorkerPoolActive  prometheus.Gauge
	WorkerPoolPending prometheus.Gauge

	// gRPC metrics
	GRPCRequestsTotal   *prometheus.CounterVec
	GRPCRequestDuration *prometheus.HistogramVec
}

// DefaultMetrics creates metrics with default settings.
var DefaultMetrics = NewMetrics("hierachain")

// NewMetrics creates a new Metrics instance with the given namespace.
func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		TransactionsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "transactions_total",
			Help:      "Total number of transactions submitted",
		}),
		TransactionsProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "transactions_processed_total",
			Help:      "Total number of transactions successfully processed",
		}),
		TransactionsFailed: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "transactions_failed_total",
			Help:      "Total number of failed transactions",
		}),
		TransactionLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "transaction_latency_seconds",
			Help:      "Transaction processing latency in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}),

		BatchesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "batches_total",
			Help:      "Total number of batches submitted",
		}),
		BatchSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "batch_size",
			Help:      "Number of transactions per batch",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
		BatchLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "batch_latency_seconds",
			Help:      "Batch processing latency in seconds",
			Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}),

		MempoolSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "mempool_size",
			Help:      "Current number of pending transactions in mempool",
		}),
		WorkerPoolActive: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "worker_pool_active",
			Help:      "Number of active workers",
		}),
		WorkerPoolPending: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "worker_pool_pending",
			Help:      "Number of pending tasks in worker pool",
		}),

		GRPCRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "grpc_requests_total",
			Help:      "Total gRPC requests by method and status",
		}, []string{"method", "status"}),
		GRPCRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "grpc_request_duration_seconds",
			Help:      "gRPC request duration by method",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method"}),
	}
}

// RecordTransaction records a transaction processing event.
func (m *Metrics) RecordTransaction(success bool, duration time.Duration) {
	m.TransactionsTotal.Inc()
	m.TransactionLatency.Observe(duration.Seconds())
	if success {
		m.TransactionsProcessed.Inc()
	} else {
		m.TransactionsFailed.Inc()
	}
}

// RecordBatch records a batch processing event.
func (m *Metrics) RecordBatch(size int, duration time.Duration) {
	m.BatchesTotal.Inc()
	m.BatchSize.Observe(float64(size))
	m.BatchLatency.Observe(duration.Seconds())
}

// RecordGRPCRequest records a gRPC request.
func (m *Metrics) RecordGRPCRequest(method, status string, duration time.Duration) {
	m.GRPCRequestsTotal.WithLabelValues(method, status).Inc()
	m.GRPCRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// UpdateMempoolSize updates the mempool gauge.
func (m *Metrics) UpdateMempoolSize(size int) {
	m.MempoolSize.Set(float64(size))
}

// UpdateWorkerPool updates worker pool gauges.
func (m *Metrics) UpdateWorkerPool(active, pending int) {
	m.WorkerPoolActive.Set(float64(active))
	m.WorkerPoolPending.Set(float64(pending))
}

// MetricsServer runs an HTTP server exposing /metrics endpoint.
type MetricsServer struct {
	server *http.Server
}

// NewMetricsServer creates a new metrics server on the given address.
func NewMetricsServer(addr string) *MetricsServer {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return &MetricsServer{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// Start starts the metrics server (blocking).
func (s *MetricsServer) Start() error {
	return s.server.ListenAndServe()
}

// StartAsync starts the metrics server in a goroutine.
func (s *MetricsServer) StartAsync() {
	go func() {
		_ = s.server.ListenAndServe()
	}()
}

// Stop gracefully stops the metrics server.
func (s *MetricsServer) Stop() error {
	return s.server.Close()
}
