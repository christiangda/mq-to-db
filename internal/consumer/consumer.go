package consumer

import "time"

// This package is an abstraction layer for queue consumers

// Consumer interface to be implemented for any kind of queue consumer
type Consumer interface {
	Connect()
	Consume() <-chan Messages
	Close()
}

// Messages struct with message payload
type Messages struct {
	ContentType     string
	ContentEncoding string
	MessageId       string
	ConsumerTag     string
	Timestamp       time.Time
	Exchange        string
	RoutingKey      string
	Payload         []byte
}
