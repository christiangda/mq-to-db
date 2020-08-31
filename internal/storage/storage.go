package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store interface
type Store interface {
	Connect(ctx context.Context) error
	ExecContext(ctx context.Context, q string) (sql.Result, error)
	Ping(ctx context.Context) error
	Close() error
	Stats() sql.DBStats
}

// Config is used to pass as argument to Store constructors
type Config struct {
	Address  string
	Port     int
	Username string
	Password string
	Database string
	SSLMode  string

	MaxPingTimeOut  time.Duration
	MaxQueryTimeOut time.Duration
	ConnMaxLifetime time.Duration
	MaxIdleConns    int
	MaxOpenConns    int
}

// GetDSN return DSN string connection
func (sc *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		sc.Address,
		sc.Port,
		sc.Username,
		sc.Password,
		sc.Database,
		sc.SSLMode,
	)
}
