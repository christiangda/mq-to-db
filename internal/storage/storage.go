package storage

import "context"

// Store interface
type Store interface {
	Ping(ctx context.Context) error
	Close() error
}
