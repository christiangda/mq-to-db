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
	DBMaxOpenConn           prometheus.Gauge
	DBOpenConn              prometheus.Gauge
	DBInUseConn             prometheus.Gauge
	DBIdleConn              prometheus.Gauge
	DBWaitCountConn         prometheus.Counter
	DBWaitDurationConn      prometheus.Counter
	DBMaxIdleClosedConn     prometheus.Counter
	DBMaxIdleTimeClosedConn prometheus.Counter
	DBMaxLifetimeClosedConn prometheus.Counter

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
		DBMaxOpenConn: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "db_max_open_conn",
			Help:      "Maximum number of open connections to the database.",
		}),
		DBOpenConn: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "db_open_conn",
			Help:      "The number of established connections both in use and idle.",
		}),
		DBInUseConn: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "db_in_use_conn",
			Help:      "The number of connections currently in use.",
		}),
		DBIdleConn: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "db_idle_conn",
			Help:      "The number of idle connections.",
		}),
		DBWaitCountConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "db_wait_count_conn",
			Help:      "The total number of connections waited for.",
		}),
		DBWaitDurationConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "db_wait_duration_conn",
			Help:      "The total time blocked waiting for a new connection.",
		}),
		DBMaxIdleClosedConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "db_wait_duration_conn",
			Help:      "The total number of connections closed due to SetMaxIdleConns.",
		}),
		DBMaxIdleTimeClosedConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "db_wait_duration_conn",
			Help:      "The total number of connections closed due to SetConnMaxIdleTime.",
		}),
		DBMaxLifetimeClosedConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "db_wait_duration_conn",
			Help:      "The total number of connections closed due to SetConnMaxLifetime.",
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
	// Globals
	prometheus.MustRegister(mtrs.Up)
	prometheus.MustRegister(mtrs.Info)

	// DB
	prometheus.MustRegister(mtrs.DBMaxOpenConn)
	prometheus.MustRegister(mtrs.DBOpenConn)
	prometheus.MustRegister(mtrs.DBInUseConn)
	prometheus.MustRegister(mtrs.DBIdleConn)
	prometheus.MustRegister(mtrs.DBWaitCountConn)
	prometheus.MustRegister(mtrs.DBWaitDurationConn)
	prometheus.MustRegister(mtrs.DBMaxIdleClosedConn)
	prometheus.MustRegister(mtrs.DBMaxIdleTimeClosedConn)
	prometheus.MustRegister(mtrs.DBMaxLifetimeClosedConn)

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
