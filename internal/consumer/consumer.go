package consumer

import (
	"errors"
	"time"
)

// This package is an abstraction layer for queue consumers
// any kind of consumer could implements interfaces here
// additionally could be used to create tests stubs

// Consumer interface to be implemented for any kind of queue consumer
type Consumer interface {
	Connect() error
	Consume(id string) (<-chan Messages, error)
	Close() error
}

// Priority represents a priority level for message queue
type Priority uint8

// Acknowledger represents the object in charge of acknowledgement
type Acknowledger interface {
	Ack() error
	Reject(requeue bool) error
}

// Messages represent the structure received into the consumer
type Messages struct {
	ContentType     string
	ContentEncoding string
	MessageID       string
	Priority        Priority
	ConsumerTag     string
	Timestamp       time.Time
	Exchange        string
	RoutingKey      string
	Payload         []byte
	Acknowledger
}

// Ack is called when the job is finished.
func (m *Messages) Ack() error {
	if m.Acknowledger == nil {
		return errors.New("Error acknowledging message: " + m.MessageID)
	}
	return m.Acknowledger.Ack()
}

// Reject is called when the job errors. The parameter is true if and only if the
// job should be put back in the queue.
func (m *Messages) Reject(requeue bool) error {
	if m.Acknowledger == nil {
		return errors.New("Error rejecting message: " + m.MessageID)
	}
	return m.Acknowledger.Reject(requeue)
}
