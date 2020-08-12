package kafka

import (
	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/consumer"
)

type kafkaConf struct{}

func New(c *config.Config) (consumer.Consumer, error) {
	return &kafkaConf{}, nil
}

func (c *kafkaConf) Connect() {}

func (c *kafkaConf) Consume() (<-chan consumer.Messages, error) {
	out := make(chan consumer.Messages)
	return out, nil
}

func (c *kafkaConf) Close() {}
