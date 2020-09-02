package metrics

import (
	"fmt"
	"net/http"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	handler http.Handler

	// Global
	Up   prometheus.Gauge
	Info prometheus.Gauge

	// DB
	DatabaseMaxOpenConnections prometheus.Gauge
	DatabaseOpenConnections    prometheus.Gauge

	// Consumers
	ConsumerRunning  *prometheus.GaugeVec
	ConsumerMessages *prometheus.CounterVec

	// Storage Workers
	StorageWorkerRunning  *prometheus.GaugeVec
	StorageWorkerMessages *prometheus.CounterVec
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
		handler: h,

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

		// DB Metrics
		DatabaseMaxOpenConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "database_max_open_connections",
			Help:      "Maximum number of open connections to the database.",
		}),
		DatabaseOpenConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "database_open_connections",
			Help:      "The number of established connections both in use and idle.",
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
			Subsystem: "storage_worker",
			Name:      "running",
			Help:      "Number of Storage Workers running"},
			[]string{
				// Storage Worker name
				"name",
			}),
		StorageWorkerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Subsystem: "storage_worker",
			Name:      "messages_total",
			Help:      "Number of messages consumed my storage_workers."},
			[]string{
				// Storage Worker name
				"name",
			}),
	}

	// Register prometheus metrics
	// Globals
	prometheus.MustRegister(mtrs.Up)
	prometheus.MustRegister(mtrs.Info)

	// DB
	prometheus.MustRegister(mtrs.DatabaseMaxOpenConnections)
	prometheus.MustRegister(mtrs.DatabaseOpenConnections)

	// Consumers
	prometheus.MustRegister(mtrs.ConsumerRunning)
	prometheus.MustRegister(mtrs.ConsumerMessages)

	// Storage Workers
	prometheus.MustRegister(mtrs.StorageWorkerRunning)
	prometheus.MustRegister(mtrs.StorageWorkerMessages)

	return mtrs
}

func (m Metrics) GetHandler() http.Handler {
	return m.handler
}
