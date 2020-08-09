package pgsqlConf

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
}

// New return
// Reference: blogger.com
func New(c *config.Config) (storage.Store, error) {

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.PostgreSQL.Server,
		c.PostgreSQL.Port,
		c.PostgreSQL.Username,
		c.PostgreSQL.Password,
		c.PostgreSQL.Database,
		c.PostgreSQL.SSLMode,
	)

	pool, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	defer pool.Close()

	pool.SetConnMaxLifetime(0)
	pool.SetMaxIdleConns(3)
	pool.SetMaxOpenConns(3)

	return &pgsqlConf{
		pool: pool,
	}, nil

}

func (c *pgsqlConf) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	return c.pool.PingContext(ctx)
}

func (c *pgsqlConf) Close() error {
	return c.pool.Close()
}
