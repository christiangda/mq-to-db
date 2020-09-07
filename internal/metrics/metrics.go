package metrics

import (
	"fmt"
	"net/http"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	handler http.Handler

	namespace string

	// Global
	Up   prometheus.Gauge
	Info prometheus.Gauge

	// Consumers
	ConsumerRunning  *prometheus.GaugeVec
	ConsumerMessages *prometheus.CounterVec

	// Storage Workers
	StorageWorkerRunning        *prometheus.GaugeVec
	StorageWorkerMessages       *prometheus.CounterVec
	StorageWorkerProcessingTime *prometheus.HistogramVec
}

// New return all the metrics
func New(c *config.Config) *Metrics {

	h := promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)

	// NOTE: Take care of metrics name
	// https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	mtrs := &Metrics{
		namespace: c.Application.MetricsNamespace,
		handler:   h,

		// Globla metrics
		Up: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "up",
			Help:      c.Application.Name + " is up and running.",
		}),
		Info: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "build_info",
			Help: fmt.Sprintf(
				"A metric with a constant '1' value labeled by version, revision, branch, and goversion from which %s was built.",
				c.Application.Name,
			),
			ConstLabels: prometheus.Labels{
				"version":   c.Application.Version,
				"revision":  c.Application.Revision,
				"branch":    c.Application.Branch,
				"goversion": c.Application.GoVersion,
			},
		}),

		// Consumers
		ConsumerRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "consumer_running",
			Help:      "Number of consumer running"},
			[]string{
				// Consumer name
				"name",
			}),
		ConsumerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "consumer_messages_total",
			Help:      "Number of messages consumed my consumers."},
			[]string{
				// Consumer name
				"name",
			}),

		// Workers
		StorageWorkerRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_worker_running",
			Help:      "Number of Storage Workers running"},
			[]string{
				// Storage Worker name
				"name",
			}),
		StorageWorkerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_worker_messages_total",
			Help:      "Number of messages consumed my storage_workers."},
			[]string{
				// Storage Worker name
				"name",
			}),
		StorageWorkerProcessingTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_worker_process_time_seconds",
			Help:      "Amount of time spent storing messages"},
			[]string{
				// Storage Worker name
				"name",
			}),
	}

	// Register prometheus metrics
	prometheus.MustRegister(mtrs.Up)
	prometheus.MustRegister(mtrs.Info)

	// Consumers
	prometheus.MustRegister(mtrs.ConsumerRunning)
	prometheus.MustRegister(mtrs.ConsumerMessages)

	// Storage Workers
	prometheus.MustRegister(mtrs.StorageWorkerRunning)
	prometheus.MustRegister(mtrs.StorageWorkerMessages)
	prometheus.MustRegister(mtrs.StorageWorkerProcessingTime)

	return mtrs
}

func (m Metrics) GetHandler() http.Handler {
	return m.handler
}

func (m Metrics) EnableDBStats(db storage.Store) {

	// DB Stats from database/sql
	dbstats := NewDBMetricsCollector(m.namespace, "db", db)
	prometheus.MustRegister(dbstats)
}
