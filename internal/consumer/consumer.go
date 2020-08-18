package consumer

import (
	"io"
	"time"
)

// This package is an abstraction layer for queue consumers

// Consumer interface to be implemented for any kind of queue consumer
type Consumer interface {
	Connect()
	Consume() (Iterator, error)
	Close()
}

type Iterator interface {
	Next() (*Messages, error)
	io.Closer
}

// Priority represents a priority level for message queue
type Priority uint8

// Acknowledger
type Acknowledger interface {
	Ack() error
	Reject(requeue bool) error
}

// Messages represent the structure received into the consumer
type Messages struct {
	ContentType     string
	ContentEncoding string
	MessageId       string
	Priority        Priority
	ConsumerTag     string
	Timestamp       time.Time
	Exchange        string
	RoutingKey      string
	Payload         []byte
	Acknowledger
}
