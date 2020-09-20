package pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	log "github.com/christiangda/mq-to-db/internal/logger"
	"github.com/christiangda/mq-to-db/internal/metrics"
	"github.com/christiangda/mq-to-db/internal/storage"
	_ "github.com/lib/pq" // this is the way to load pgsql driver to be used by golang database/sql
)

// PGSQL is a implementation go storage.Store interface
type PGSQL struct {
	pool *sql.DB
	conn *sql.Conn

	maxPingTimeOut  time.Duration
	maxQueryTimeOut time.Duration

	mtrs *metrics.Metrics
}

// New return
func New(c *storage.Config, mtrs *metrics.Metrics) (*PGSQL, error) {

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
	//defer pool.Close()

	pool.SetConnMaxLifetime(c.ConnMaxLifetime)
	pool.SetMaxIdleConns(c.MaxIdleConns)
	pool.SetMaxOpenConns(c.MaxOpenConns)

	out := &PGSQL{
		pool:            pool,
		maxPingTimeOut:  c.MaxPingTimeOut,
		maxQueryTimeOut: c.MaxQueryTimeOut,

		mtrs: mtrs,
	}

	return out, nil
}

// Connect returns a single connection by either opening a new connection
// or returning an existing connection from the connection pool.
func (c *PGSQL) Connect(ctx context.Context) error {

	conn, err := c.pool.Conn(ctx)
	c.conn = conn
	//defer conn.Close()

	return err
}

// Ping verifies a connection to the database is still alive,
// establishing a connection if necessary.
func (c *PGSQL) Ping(ctx context.Context) error {

	ctx, cancel := context.WithTimeout(ctx, c.maxPingTimeOut)
	defer cancel()

	c.mtrs.StoragePingTotal.Inc()
	err := c.pool.PingContext(ctx)

	if ctx.Err() == context.DeadlineExceeded {
		c.mtrs.StoragePingTimeOutTotal.Inc()
		log.Warnf("Ping time out (%v) ", c.maxQueryTimeOut)
		err = ctx.Err()
	}
	return err
}

// ExecContext executes a query without returning any rows.
func (c *PGSQL) ExecContext(ctx context.Context, q string) (sql.Result, error) {

	ctx, cancel := context.WithTimeout(ctx, c.maxQueryTimeOut)
	defer cancel()

	c.mtrs.StorageExecTotal.Inc()
	res, err := c.pool.ExecContext(ctx, q)

	if ctx.Err() == context.DeadlineExceeded {
		c.mtrs.StorageExecTimeOutTotal.Inc()
		log.Warnf("Query time out (%v) for: %s", c.maxQueryTimeOut, q)
		return nil, ctx.Err()
	}

	return res, err
}

// Close closes the database and prevents new queries from starting.
func (c *PGSQL) Close() error {
	return c.pool.Close()
}

// Close closes the database and prevents new queries from starting.
func (c *PGSQL) Stats() sql.DBStats {
	return c.pool.Stats()
}
