package pgsql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/storage"
)

type pgsqlConf struct {
	pool *sql.DB
	conn *sql.Conn

	maxPingTimeOut  time.Duration
	maxQueryTimeOut time.Duration
}

// New return
// Reference: blogger.com
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

func (c *pgsqlConf) Connect(ctx context.Context) error {
	conn, err := c.pool.Conn(ctx)
	c.conn = conn
	//defer conn.Close()
	return err
}

func (c *pgsqlConf) Ping(ctx context.Context) error {

	ctx, cancel := context.WithTimeout(ctx, c.maxPingTimeOut)
	defer cancel()

	return c.pool.PingContext(ctx)
}

func (c *pgsqlConf) ExecContext(ctx context.Context, q string) (sql.Result, error) {

	ctx, cancel := context.WithTimeout(ctx, c.maxQueryTimeOut)
	defer cancel()

	res, err := c.pool.ExecContext(ctx, q)

	return res, err
}

func (c *pgsqlConf) Close() error {
	return c.pool.Close()
}
