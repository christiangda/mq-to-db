package consumer

import (
	"errors"
	"fmt"
	"time"
)

// This package is an abstraction layer for queue consumers
// any kind of consumer could implements interfaces here
// additionally could be used to create tests stubs

// Config used to configure consumers
type Config struct {
	Name               string
	Address            string
	Port               int
	RequestedHeartbeat time.Duration
	Username           string
	Password           string
	VirtualHost        string
	Queue              Queue
	Exchange           Exchange
}

type Queue struct {
	Name          string
	RoutingKey    string
	Durable       bool
	AutoDelete    bool
	Exclusive     bool
	AutoACK       bool
	PrefetchCount int
	PrefetchSize  int
	Args          map[string]interface{}
}

type Exchange struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Args       map[string]interface{}
}

// GetURI return the consumer URI
func (cc *Config) GetURI() (string, error) {
	if cc.Address == "" || cc.Port == 0 {
		return "", errors.New("address or port empty")
	}

	if cc.Username == "" {
		return fmt.Sprintf("amqp://%s:%d/", cc.Address, cc.Port), nil
	}

	if cc.Password == "" {
		return fmt.Sprintf("amqp://%s:@%s:%d/", cc.Username, cc.Address, cc.Port), nil
	}

	return fmt.Sprintf("amqp://%s:%v@%s:%d/", cc.Username, cc.Password, cc.Address, cc.Port), nil
}

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
		return errors.New("error acknowledging message: " + m.MessageID)
	}
	return m.Acknowledger.Ack()
}

// Reject is called when the job errors. The parameter is true if and only if the
// job should be put back in the queue.
func (m *Messages) Reject(requeue bool) error {
	if m.Acknowledger == nil {
		return errors.New("error rejecting message: " + m.MessageID)
	}
	return m.Acknowledger.Reject(requeue)
}
