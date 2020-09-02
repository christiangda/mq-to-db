package metrics

import (
	"fmt"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	// Global
	Up   prometheus.Gauge
	Info prometheus.Gauge

	// DB
	MaxOpenConnections prometheus.Gauge
	OpenConnections    prometheus.Gauge

	// Consumers
	RunningConsumers  *prometheus.GaugeVec
	ConsumersMessages *prometheus.CounterVec

	// Storage Workers
	RunningStorageWorkers  *prometheus.GaugeVec
	StorageWorkersMessages *prometheus.CounterVec
}

// New return all the metrics
func New(c *config.Config) *Metrics {

	mtrs := &Metrics{
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
		MaxOpenConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "max_open_connections",
			Help:      "Maximum number of open connections to the database.",
		}),
		OpenConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "open_connections",
			Help:      "The number of established connections both in use and idle.",
		}),

		// Consumers
		RunningConsumers: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "running_consumers",
			Help:      "Number of consumer running"},
			[]string{
				// Consumer name
				"name",
			}),
		ConsumersMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "consumers_messages",
			Help:      "Number of messages consumed my consumers."},
			[]string{
				// Consumer name
				"name",
			}),

		// Workers
		RunningStorageWorkers: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "running_storage_workers",
			Help:      "Number of Storage Workers running"},
			[]string{
				// Storage Worker name
				"name",
			}),
		StorageWorkersMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: c.Application.MetricsNamespace,
			Name:      "storage_workers_messages",
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
	prometheus.MustRegister(mtrs.MaxOpenConnections)
	prometheus.MustRegister(mtrs.OpenConnections)

	// Consumers
	prometheus.MustRegister(mtrs.RunningConsumers)
	prometheus.MustRegister(mtrs.ConsumersMessages)

	// Storage Workers
	prometheus.MustRegister(mtrs.RunningStorageWorkers)
	prometheus.MustRegister(mtrs.StorageWorkersMessages)

	return mtrs
}
