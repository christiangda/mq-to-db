package kafka

import (
	"github.com/christiangda/mq-to-db/internal/consumer"
)

// Consumer implement Consumer.Consumer interface
type Consumer struct{}

// New return a new instance of consumer.Consumer
func New(c *consumer.Config) (consumer.Consumer, error) {
	return &Consumer{}, nil
}

// Connect ...
func (c *Consumer) Connect() error {
	return nil
}

// Consume ...
func (c *Consumer) Consume(id string) (<-chan consumer.Messages, error) {
	//cm := "this is a channel"
	m := make(<-chan consumer.Messages)
	return m, nil
	//return &Iterator{messages: m, ch: &cm, id: "1"}, nil
}

// Close ...
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
