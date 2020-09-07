package metrics

import (
	"database/sql"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// StatsGetter is an interface that gets sql.DBStats.
// It's implemented by e.g. *sql.DB or *sqlx.DB.
type DBStatsGetter interface {
	Stats() sql.DBStats
}

type DBMetricsCollector struct {
	dbsg  DBStatsGetter
	mutex sync.RWMutex

	// DB
	maxOpenConn           prometheus.Gauge
	openConn              prometheus.Gauge
	inUseConn             prometheus.Gauge
	idleConn              prometheus.Gauge
	waitCountConn         prometheus.Counter
	waitDurationConn      prometheus.Counter
	maxIdleClosedConn     prometheus.Counter
	maxIdleTimeClosedConn prometheus.Counter
	maxLifetimeClosedConn prometheus.Counter
}

// New return all the metrics
func NewDBMetricsCollector(namespace, subsystem string, db DBStatsGetter) *DBMetricsCollector {

	// NOTE: Take care of metrics name
	// https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	mtrs := &DBMetricsCollector{
		dbsg: db,

		// DB Metrics
		maxOpenConn: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "max_open_conn",
			Help:      "Maximum number of open connections to the database.",
		}),
		openConn: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "open_conn",
			Help:      "The number of established connections both in use and idle.",
		}),
		inUseConn: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "in_use_conn",
			Help:      "The number of connections currently in use.",
		}),
		idleConn: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "idle_conn",
			Help:      "The number of idle connections.",
		}),
		waitCountConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "wait_count_conn",
			Help:      "The total number of connections waited for.",
		}),
		waitDurationConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "wait_duration_conn",
			Help:      "The total time blocked waiting for a new connection.",
		}),
		maxIdleClosedConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "max_idle_closed_conn",
			Help:      "The total number of connections closed due to SetMaxIdleConns.",
		}),
		maxIdleTimeClosedConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "max_idle_time_closed_conn",
			Help:      "The total number of connections closed due to SetConnMaxIdleTime.",
		}),
		maxLifetimeClosedConn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "max_lifetime_closed_conn",
			Help:      "The total number of connections closed due to SetConnMaxLifetime.",
		}),
	}
	return mtrs
}

// Describe ...
func (c *DBMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	// DB
	ch <- c.maxOpenConn.Desc()
	ch <- c.openConn.Desc()
	ch <- c.inUseConn.Desc()
	ch <- c.idleConn.Desc()
	ch <- c.waitCountConn.Desc()
	ch <- c.waitDurationConn.Desc()
	ch <- c.maxIdleClosedConn.Desc()
	ch <- c.maxIdleTimeClosedConn.Desc()
	ch <- c.maxLifetimeClosedConn.Desc()
}

// Collect ...
func (c *DBMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	s := c.dbsg.Stats()

	c.maxOpenConn.Add(float64(s.MaxOpenConnections))
	c.openConn.Add(float64(s.OpenConnections))
	c.inUseConn.Add(float64(s.InUse))
	c.idleConn.Add(float64(s.Idle))
	c.waitCountConn.Add(float64(s.WaitCount))
	c.waitDurationConn.Add(float64(s.WaitDuration))
	c.maxIdleClosedConn.Add(float64(s.MaxIdleClosed))
	c.maxIdleTimeClosedConn.Add(float64(s.MaxIdleTimeClosed))
	c.maxLifetimeClosedConn.Add(float64(s.MaxLifetimeClosed))

	ch <- c.maxOpenConn
	ch <- c.openConn
	ch <- c.inUseConn
	ch <- c.idleConn
	ch <- c.waitCountConn
	ch <- c.waitDurationConn
	ch <- c.maxIdleClosedConn
	ch <- c.maxIdleTimeClosedConn
	ch <- c.maxLifetimeClosedConn
}
