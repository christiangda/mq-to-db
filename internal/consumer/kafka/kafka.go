package kafka

import (
	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/consumer"
)

// Implement Consumer.Consumer interface
type Consumer struct{}

func New(c *config.Config) (consumer.Consumer, error) {
	return &Consumer{}, nil
}

func (c *Consumer) Connect() {}

func (c *Consumer) Consume(id string) (<-chan consumer.Messages, error) {
	//cm := "this is a channel"
	m := make(<-chan consumer.Messages)
	return m, nil
	//return &Iterator{messages: m, ch: &cm, id: "1"}, nil
}

func (c *Consumer) Close() error { return nil }

// Acknowledger implements the Acknowledger for AMQP.
type Acknowledger struct {
}

// Ack signals acknowledgement.
func (a *Acknowledger) Ack() error {
	return nil
}

// Reject signals rejection. If requeue is false, the job will go to the buried
// queue until Queue.RepublishBuried() is called.
func (a *Acknowledger) Reject(requeue bool) error {
	return nil
}
