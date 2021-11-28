package model

import (
	"errors"
	"time"
)

// Priority represents a priority level for message queue
type Priority uint8

// Acknowledger represents the object in charge of acknowledgement
// this is used to consume the methods from the broker Acknowledgements system
type Acknowledger interface {
	Ack(tag uint64, multiple bool) error
	Nack(tag uint64, multiple bool, requeue bool) error
	Reject(tag uint64, requeue bool) error
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
	DeliveryTag uint64
}

// Ack is called when the job is finished.
func (m *Messages) Ack(multiple bool) error {
	if m.Acknowledger == nil {
		return errors.New("error acknowledging message: " + m.MessageID)
	}
	return m.Acknowledger.Ack(m.DeliveryTag, multiple)
}

// Reject is called when the job errors. The parameter is true if and only if the
// job should be put back in the queue.
func (m *Messages) Reject(requeue bool) error {
	if m.Acknowledger == nil {
		return errors.New("error rejecting message: " + m.MessageID)
	}
	return m.Acknowledger.Reject(m.DeliveryTag, requeue)
}

func (m *Messages) Nack(multiple bool, requeue bool) error {
	if m.Acknowledger == nil {
		return errors.New("error rejecting message: " + m.MessageID)
	}
	return m.Acknowledger.Nack(m.DeliveryTag, multiple, requeue)
}
