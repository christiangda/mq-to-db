package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
	pool *sql.DB
	conn *sql.Conn

	maxPingTimeOut  time.Duration
	maxQueryTimeOut time.Duration

	StoragePingTotal        prometheus.Counter
	StoragePingTimeOutTotal prometheus.Counter
	StorageExecTotal        prometheus.Counter
	StorageExecTimeOutTotal prometheus.Counter
}

// New return
func NewPGSQL(c *Config, db SQLService) (*PGSQL, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Address,
		c.Port,
		c.Username,
		c.Password,
		c.Database,
		c.SSLMode,
	)

	pool, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	// defer pool.Close()

	pool.SetConnMaxLifetime(c.ConnMaxLifetime)
	pool.SetMaxIdleConns(c.MaxIdleConns)
	pool.SetMaxOpenConns(c.MaxOpenConns)

	out := &PGSQL{
		pool:            pool,
		maxPingTimeOut:  c.MaxPingTimeOut,
		maxQueryTimeOut: c.MaxQueryTimeOut,

		StoragePingTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Name:      "storage_ping_total",
			Help:      "Number of ping executed by storage.",
		},
		),
		StoragePingTimeOutTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Name:      "storage_ping_timeout_total",
			Help:      "Number of ping with timeouts executed by storage.",
		},
		),
		StorageExecTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Name:      "storage_exec_total",
			Help:      "Number of exec executed by storage.",
		},
		),
		StorageExecTimeOutTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Name:      "storage_exec_timeout_total",
			Help:      "Number of exec with timeouts executed by storage.",
		},
		),
	}
	return out, nil
}

// Connect returns a single connection by either opening a new connection
// or returning an existing connection from the connection pool.
func (c *PGSQL) Connect(ctx context.Context) error {
	conn, err := c.pool.Conn(ctx)
	c.conn = conn
	// defer conn.Close()

	return err
}

// Ping verifies a connection to the database is still alive,
// establishing a connection if necessary.
func (c *PGSQL) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.maxPingTimeOut)
	defer cancel()

	c.StoragePingTotal.Inc()
	err := c.pool.PingContext(ctx)

	if ctx.Err() == context.DeadlineExceeded {
		c.StoragePingTimeOutTotal.Inc()
		log.Warnf("Ping time out (%v) ", c.maxQueryTimeOut)
		err = ctx.Err()
	}
	return err
}

// ExecContext executes a query without returning any rows.
func (c *PGSQL) ExecContext(ctx context.Context, q string) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, c.maxQueryTimeOut)
	defer cancel()

	c.StorageExecTotal.Inc()
	res, err := c.pool.ExecContext(ctx, q)

	if ctx.Err() == context.DeadlineExceeded {
		c.StorageExecTimeOutTotal.Inc()
		log.Warnf("Query time out (%v) for: %s", c.maxQueryTimeOut, q)
		return nil, ctx.Err()
	}

	return res, err
}

// Close closes the database and prevents new queries from starting.
func (c *PGSQL) Close() error {
	return c.pool.Close()
}

// Stats return the database metrics statistics
func (c *PGSQL) Stats() sql.DBStats {
	return c.pool.Stats()
}
