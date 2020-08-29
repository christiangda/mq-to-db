package pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/storage"
	_ "github.com/lib/pq" // this is the way to load pgsql driver to be used by golang database/sql
	log "github.com/sirupsen/logrus"
)

type pgsqlConf struct {
	pool *sql.DB
	conn *sql.Conn

	maxPingTimeOut  time.Duration
	maxQueryTimeOut time.Duration
}

// New return
func New(c *config.Config) (storage.Store, error) {

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Address,
		c.Database.Port,
		c.Database.Username,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)

	pool, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	//defer pool.Close()

	pool.SetConnMaxLifetime(c.Database.ConnMaxLifetime)
	pool.SetMaxIdleConns(c.Database.MaxIdleConns)
	pool.SetMaxOpenConns(c.Database.MaxOpenConns)

	return &pgsqlConf{
		pool:            pool,
		maxPingTimeOut:  c.Database.MaxPingTimeOut,
		maxQueryTimeOut: c.Database.MaxQueryTimeOut,
	}, nil

}

// Connect returns a single connection by either opening a new connection
// or returning an existing connection from the connection pool.
func (c *pgsqlConf) Connect(ctx context.Context) error {

	conn, err := c.pool.Conn(ctx)
	c.conn = conn
	//defer conn.Close()

	return err
}

// Ping verifies a connection to the database is still alive,
// establishing a connection if necessary.
func (c *pgsqlConf) Ping(ctx context.Context) error {

	ctx, cancel := context.WithTimeout(ctx, c.maxPingTimeOut)
	defer cancel()

	err := c.pool.PingContext(ctx)

	if ctx.Err() == context.DeadlineExceeded {
		log.Warnf("Ping time out (%v) ", c.maxQueryTimeOut)
		err = ctx.Err()
	}
	return err
}

// ExecContext executes a query without returning any rows.
func (c *pgsqlConf) ExecContext(ctx context.Context, q string) (sql.Result, error) {

	ctx, cancel := context.WithTimeout(ctx, c.maxQueryTimeOut)
	defer cancel()

	res, err := c.pool.ExecContext(ctx, q)

	if ctx.Err() == context.DeadlineExceeded {
		log.Warnf("Query time out (%v) for: %s", c.maxQueryTimeOut, q)
		return nil, ctx.Err()
	}

	return res, err
}

// Close closes the database and prevents new queries from starting.
func (c *pgsqlConf) Close() error {
	return c.pool.Close()
}
