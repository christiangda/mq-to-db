package storage

import (
	"context"
	"database/sql"
)

// Store interface
type Store interface {
	Connect(ctx context.Context) error
	ExecContext(ctx context.Context, q string) (sql.Result, error)
	Ping(ctx context.Context) error
	Close() error
}
