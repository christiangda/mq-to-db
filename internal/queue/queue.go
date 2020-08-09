package queue

// This package is an abstraction layer for queue consumers

// Consummer interface to be implemented for any kind of queue consumer
type Consummer interface {
	Connect()
	Consume() <-chan Messages
	Close()
}

// Messages struct with message payload
type Messages struct {
	Payload []byte
	Length  int
}
