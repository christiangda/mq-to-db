package metrics

import (
	"fmt"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics ...
type Metrics struct {
	namespace string

	// Global
	Up   prometheus.Gauge
	Info prometheus.Gauge
}

// New return all the metrics
func New(c *config.Config) *Metrics {
	// NOTE: Take care of metrics name
	// https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	mtrs := &Metrics{
		namespace: c.Application.MetricsNamespace,

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
	}

	// Register prometheus metrics
	// Global
	prometheus.MustRegister(mtrs.Up)
	prometheus.MustRegister(mtrs.Info)

	return mtrs
}
