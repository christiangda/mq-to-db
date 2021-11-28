package storage

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	_ "github.com/lib/pq" // this is the way to load pgsql driver to be used by golang database/sql
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -package=mocks -destination=../../mocks/storage/pgsql_mocks.go -source=pgsql.go SQLService
// Store interface is used to consume methods from sql.db
type SQLService interface {
	Conn(ctx context.Context) (*sql.Conn, error)
	PingContext(ctx context.Context) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Stats() sql.DBStats
	Close() error
}

// PGSQL is a implementation go storage.Store interface
type PGSQL struct {
	db              SQLService
	maxPingTimeOut  time.Duration
	maxQueryTimeOut time.Duration

	DBMetrics               *DBMetricsCollector
	StoragePingTotal        prometheus.Counter
	StoragePingTimeOutTotal prometheus.Counter
	StorageExecTotal        prometheus.Counter
	StorageExecTimeOutTotal prometheus.Counter
}

// New return
// func NewPGSQL(c *Config, db SQLService) (*PGSQL, error) {
func NewPGSQL(c *Config, db SQLService) (*PGSQL, error) {
	out := &PGSQL{
		db:              db,
		maxPingTimeOut:  c.MaxPingTimeOut,
		maxQueryTimeOut: c.MaxQueryTimeOut,

		DBMetrics: NewDBMetricsCollector("mq_to_db", "db", db),
		StoragePingTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Subsystem: "storage",
			Name:      "ping_total",
			Help:      "Number of ping executed by storage.",
		},
		),
		StoragePingTimeOutTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Subsystem: "storage",
			Name:      "ping_timeout_total",
			Help:      "Number of ping with timeouts executed by storage.",
		},
		),
		StorageExecTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Subsystem: "storage",
			Name:      "exec_total",
			Help:      "Number of exec executed by storage.",
		},
		),
		StorageExecTimeOutTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Subsystem: "storage",
			Name:      "exec_timeout_total",
			Help:      "Number of exec with timeouts executed by storage.",
		},
		),
	}

	prometheus.MustRegister(out.DBMetrics)
	prometheus.MustRegister(out.StoragePingTotal)
	prometheus.MustRegister(out.StoragePingTimeOutTotal)
	prometheus.MustRegister(out.StorageExecTotal)
	prometheus.MustRegister(out.StorageExecTimeOutTotal)

	return out, nil
}

// ExecContext executes a query without returning any rows.
func (c *PGSQL) ExecContext(ctx context.Context, q string) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, c.maxQueryTimeOut)
	defer cancel()

	conn, err := c.getConn(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if e := conn.Close(); e != nil {
			log.Errorf("Close failed, error: %v", e)
		}
	}()

	c.StorageExecTotal.Inc()
	res, err := conn.ExecContext(ctx, q)

	if ctx.Err() == context.DeadlineExceeded {
		c.StorageExecTimeOutTotal.Inc()
		log.Warnf("Query time out (%v) for: %s", c.maxQueryTimeOut, q)
		return nil, ctx.Err()
	}

	return res, err
}

// Stats return the database metrics statistics
func (c *PGSQL) Stats() sql.DBStats {
	return c.db.Stats()
}

// getConn returns a single connection by either opening a new connection
// reference: https://stackoverflow.com/questions/67176979/bad-connection-response-to-long-running-mssql-transaction-in-golang
func (c *PGSQL) getConn(ctx context.Context) (*sql.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, c.maxPingTimeOut)
	defer cancel()

	c.StoragePingTotal.Inc()
	err := c.db.PingContext(ctx)
	if ctx.Err() == context.DeadlineExceeded {
		c.StoragePingTimeOutTotal.Inc()
		log.Warnf("Ping time out (%v) ", c.maxQueryTimeOut)
		return nil, ctx.Err()
	} else if err != nil {
		return nil, err
	}

	repeater := backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10), ctx)

	var conn *sql.Conn
	if err := backoff.Retry(func() error {
		// Attempt to get the connection to the database
		var err error
		if conn, err = c.db.Conn(ctx); err != nil {

			// We failed to get the connection; if we have a login error, an EOF or handshake
			// failure then we'll attempt the connection again later so just return it and let
			// the backoff code handle it
			log.Warnf("Database Connection failed, error: %v", err)
			if strings.Contains(err.Error(), "EOF") {
				return err
			} else if strings.Contains(err.Error(), "TLS Handshake failed") {
				return err
			}

			// Otherwise, we can't recover from the error so return it
			return backoff.Permanent(err)
		}

		return nil
	}, repeater); err != nil {
		return nil, err
	}

	return conn, nil
}
