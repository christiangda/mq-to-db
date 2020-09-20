package kafka

import (
	"github.com/christiangda/mq-to-db/internal/consumer"
)

// Kafka implement Consumer.Consumer interface
type Kafka struct{}

// New return a new instance of consumer.Consumer
func New(c *consumer.Config) (*Kafka, error) {
	return &Kafka{}, nil
}

// Connect ...
func (c *Kafka) Connect() error {
	return nil
}

// Consume ...
func (c *Kafka) Consume(id string) (<-chan consumer.Messages, error) {
	//cm := "this is a channel"
	m := make(<-chan consumer.Messages)
	return m, nil
	//return &Iterator{messages: m, ch: &cm, id: "1"}, nil
}

// Close ...
func (c *Kafka) Close() error { return nil }

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
