package memory

import (
	"context"
	"database/sql"

	"github.com/christiangda/mq-to-db/internal/storage"
)

type memoryConf struct {
}

// New ...
func New(c *storage.Config) (storage.Store, error) {
	return &memoryConf{}, nil
}

// Connect ...
func (c *memoryConf) Connect(ctx context.Context) error {
	return nil
}

// Ping ...
func (c *memoryConf) Ping(ctx context.Context) error {
	return nil
}

// ExecContext ...
func (c *memoryConf) ExecContext(ctx context.Context, q string) (sql.Result, error) {
	return nil, nil
}

// Close ...
func (c *memoryConf) Close() error {
	return nil
}

// Stats ...
func (c *memoryConf) Stats() sql.DBStats {
	return sql.DBStats{}
}
