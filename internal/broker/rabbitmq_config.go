package broker

import (
	"errors"
	"fmt"
	"time"
)

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
