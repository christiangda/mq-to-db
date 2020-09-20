package memory

import (
	"context"
	"database/sql"

	"github.com/christiangda/mq-to-db/internal/storage"
)

// MEM is an implementation of storage.store interface
type MEM struct {
}

// New ...
func New(c *storage.Config) (*MEM, error) {
	return &MEM{}, nil
}

// Connect ...
func (c *MEM) Connect(ctx context.Context) error {
	return nil
}

// Ping ...
func (c *MEM) Ping(ctx context.Context) error {
	return nil
}

// ExecContext ...
func (c *MEM) ExecContext(ctx context.Context, q string) (sql.Result, error) {
	return nil, nil
}

// Close ...
func (c *MEM) Close() error {
	return nil
}

// Stats ...
func (c *MEM) Stats() sql.DBStats {
	return sql.DBStats{}
}
