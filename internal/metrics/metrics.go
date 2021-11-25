package metrics

import (
	"fmt"
	"net/http"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics ...
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
	StorageWorkerRunning            *prometheus.GaugeVec
	StorageWorkerMessages           *prometheus.CounterVec
	StorageWorkerProcessingDuration *prometheus.HistogramVec

	// Storer
	StorerMessagesTotal              prometheus.Counter
	StorerMessagesErrorsTotal        prometheus.Counter
	StorerSQLMessagesTotal           prometheus.Counter
	StorerSQLMessagesErrorsTotal     prometheus.Counter
	StorerSQLMessagesToDBTotal       prometheus.Counter
	StorerSQLMessagesToDBErrorsTotal prometheus.Counter
	StorerMessagesAckTotal           prometheus.Counter
	StorerMessagesRejectedTotal      prometheus.Counter

	// Storage
	StoragePingTotal        prometheus.Counter
	StoragePingTimeOutTotal prometheus.Counter
	StorageExecTotal        prometheus.Counter
	StorageExecTimeOutTotal prometheus.Counter
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

		// Global metrics
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
			Help:      "Number of consumer running",
		},
			[]string{
				// Consumer name
				"name",
			}),
		ConsumerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "consumer_messages_total",
			Help:      "Number of messages consumed my consumers.",
		},
			[]string{
				// Consumer name
				"name",
			}),

		// Workers
		StorageWorkerRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_worker_running",
			Help:      "Number of Storage Workers running",
		},
			[]string{
				// Storage Worker name
				"name",
			}),
		StorageWorkerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_worker_messages_total",
			Help:      "Number of messages consumed my storage_workers.",
		},
			[]string{
				// Storage Worker name
				"name",
			}),
		StorageWorkerProcessingDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_worker_process_duration_seconds",
			Help:      "Amount of time spent storing messages",
			Buckets:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15}, // Observation buckets
		},
			[]string{
				// Storage Worker name
				"name",
			}),

		// Storer
		StorerMessagesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storer_messages_total",
			Help:      "Number of messages processed by storer.",
		},
		),
		StorerMessagesErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storer_messages_errors_total",
			Help:      "Number of messages with errors processed by storer.",
		},
		),
		StorerSQLMessagesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storer_sql_messages_total",
			Help:      "Number of sql messages processed by storer.",
		},
		),
		StorerSQLMessagesErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storer_sql_messages_errors_total",
			Help:      "Number of sql messages with errors processed by storer.",
		},
		),
		StorerSQLMessagesToDBTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storer_sql_messages_to_db_total",
			Help:      "Number of sql messages sent to database by storer.",
		},
		),
		StorerSQLMessagesToDBErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storer_sql_messages_to_db_errors_total",
			Help:      "Number of sql messages with errors sent to database by storer.",
		},
		),
		StorerMessagesAckTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storer_messages_ack_total",
			Help:      "Number of messages ack into mq system.",
		},
		),
		StorerMessagesRejectedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storer_messages_rejected_total",
			Help:      "Number of messages rejected into mq system.",
		},
		),

		// Storage
		StoragePingTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_ping_total",
			Help:      "Number of ping executed by storage.",
		},
		),
		StoragePingTimeOutTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_ping_timeout_total",
			Help:      "Number of ping with timeouts executed by storage.",
		},
		),
		StorageExecTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_exec_total",
			Help:      "Number of exec executed by storage.",
		},
		),
		StorageExecTimeOutTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_exec_timeout_total",
			Help:      "Number of exec with timeouts executed by storage.",
		},
		),
	}

	// Register prometheus metrics
	// Global
	prometheus.MustRegister(mtrs.Up)
	prometheus.MustRegister(mtrs.Info)

	// Consumers
	prometheus.MustRegister(mtrs.ConsumerRunning)
	prometheus.MustRegister(mtrs.ConsumerMessages)

	// Storage Workers
	prometheus.MustRegister(mtrs.StorageWorkerRunning)
	prometheus.MustRegister(mtrs.StorageWorkerMessages)
	prometheus.MustRegister(mtrs.StorageWorkerProcessingDuration)

	// Storer
	prometheus.MustRegister(mtrs.StorerMessagesTotal)
	prometheus.MustRegister(mtrs.StorerMessagesErrorsTotal)
	prometheus.MustRegister(mtrs.StorerSQLMessagesTotal)
	prometheus.MustRegister(mtrs.StorerSQLMessagesErrorsTotal)
	prometheus.MustRegister(mtrs.StorerSQLMessagesToDBTotal)
	prometheus.MustRegister(mtrs.StorerSQLMessagesToDBErrorsTotal)
	prometheus.MustRegister(mtrs.StorerMessagesAckTotal)
	prometheus.MustRegister(mtrs.StorerMessagesRejectedTotal)

	// Storage
	prometheus.MustRegister(mtrs.StoragePingTotal)
	prometheus.MustRegister(mtrs.StoragePingTimeOutTotal)
	prometheus.MustRegister(mtrs.StorageExecTotal)
	prometheus.MustRegister(mtrs.StorageExecTimeOutTotal)

	return mtrs
}

// GetHandler return the http handler for metrics endpoints
func (m Metrics) GetHandler() http.Handler {
	return m.handler
}

// EnableDBStats enable the database/sql stats metrics
func (m Metrics) EnableDBStats(db storage.Store) {
	// DB Stats from database/sql
	dbStats := NewDBMetricsCollector(m.namespace, "db", db)
	prometheus.MustRegister(dbStats)
}
