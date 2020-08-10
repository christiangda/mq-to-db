package memory

import (
	"context"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/storage"
)

type memoryConf struct {
}

func New(c *config.Config) (storage.Store, error) {
	return &memoryConf{}, nil
}

func (c *memoryConf) Ping(ctx context.Context) error {
	return nil
}

func (c *memoryConf) Close() error {
	return nil
}
