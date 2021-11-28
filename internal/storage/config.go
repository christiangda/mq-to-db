package storage

import (
	"time"
)

// Config is used to pass as argument to Store constructors
type Config struct {
	MaxPingTimeOut  time.Duration
	MaxQueryTimeOut time.Duration
}
