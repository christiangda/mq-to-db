package rmq

import (
	"fmt"
	"time"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/consumer"
	log "github.com/christiangda/mq-to-db/internal/logger"
	"github.com/streadway/amqp"
)

// Consumer is a RabbitMQ consumer configuration
// Implement Consumer.Consumer interface
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel

	name string
	uri  string

	requestedHeartbeat time.Duration
	virtualHost        string
	queue              struct {
		name       string
		routingKey string
		durable    bool
		autoDelete bool
		exclusive  bool
		autoACK    bool
		args       map[string]interface{}
	}
	exchange struct {
		name       string
		kind       string
		durable    bool
		autoDelete bool
		args       map[string]interface{}
	}
}

// New create a new rabbitmq consumer with implements consumer.Consumer interface
func New(c *config.Config) (consumer.Consumer, error) {

	uri := fmt.Sprintf("amqp://%s:%s@%s:%d/", c.Consumer.Username, c.Consumer.Password, c.Consumer.Address, c.Consumer.Port)

	return &Consumer{
		name:               c.Application.Name,
		uri:                uri,
		requestedHeartbeat: c.Consumer.RequestedHeartbeat,
		virtualHost:        c.Consumer.VirtualHost,
		queue: struct {
			name       string
			routingKey string
			durable    bool
			autoDelete bool
			exclusive  bool
			autoACK    bool
			args       map[string]interface{}
		}{
			c.Consumer.Queue.Name,
			c.Consumer.Queue.RoutingKey,
			c.Consumer.Queue.Durable,
			c.Consumer.Queue.AutoDelete,
			c.Consumer.Queue.Exclusive,
			c.Consumer.Queue.AutoACK,
			c.Consumer.Queue.Args,
		},
		exchange: struct {
			name       string
			kind       string
			durable    bool
			autoDelete bool
			args       map[string]interface{}
		}{
			c.Consumer.Exchange.Name,
			c.Consumer.Exchange.Kind,
			c.Consumer.Exchange.Durable,
			c.Consumer.Exchange.AutoDelete,
			c.Consumer.Exchange.Args,
		},
	}, nil
}

// Connect to RabbitMQ server and channel
func (c *Consumer) Connect() {

	amqpConfig := amqp.Config{}

	amqpConfig.Heartbeat = c.requestedHeartbeat
	if c.virtualHost != "" {
		amqpConfig.Vhost = c.virtualHost
	}

	log.Debugf("Connecting to: %s", c.uri)
	conn, err := amqp.DialConfig(
		c.uri,
		amqpConfig,
	)
	if err != nil {
		log.Fatal(err)
	}
	//defer conn.Close()
	c.conn = conn

	log.Debug("Getting Channel")
	ch, err := c.conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	//defer ch.Close()
	c.channel = ch

	log.Debugf("Declaring channel exchange: %s", c.exchange.name)
	err = c.channel.ExchangeDeclare(
		c.exchange.name,
		c.exchange.kind,
		c.exchange.durable,
		c.exchange.autoDelete,
		false, // internal
		false, // no-wait
		c.exchange.args,
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Declaring channel queue: %s", c.queue.name)
	q, err := c.channel.QueueDeclare(
		c.queue.name,
		c.queue.durable,
		c.queue.autoDelete,
		c.queue.exclusive,
		false, // no-wait
		c.queue.args,
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Binding queue: %s to exchange: %s using routing key: %s", q.Name, c.exchange.name, c.queue.routingKey)
	err = c.channel.QueueBind(
		q.Name,
		c.queue.routingKey,
		c.exchange.name,
		false, // no-wait
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
}

// Consume messages from the channel
func (c *Consumer) Consume(id string) (<-chan consumer.Messages, error) {

	// Register a consumer
	msgs, err := c.channel.Consume(
		c.queue.name,
		id,                // consumer id
		c.queue.autoACK,   // auto-ack
		c.queue.exclusive, // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		return nil, err
	}

	// This channels is used to be filled by messages comming from
	// the queue system
	// This is part of "producer-consume queue pattern"
	out := make(chan consumer.Messages)

	// NOTE: This is necessary to consume the original channel without blocking it
	go func() {
		for d := range msgs {
			out <- consumer.Messages{
				MessageID:    d.MessageId,
				Priority:     consumer.Priority(d.Priority),
				Timestamp:    d.Timestamp,
				ContentType:  d.ContentType,
				Acknowledger: &Acknowledger{d.Acknowledger, d.DeliveryTag},
				Payload:      d.Body,
			}
		}
		close(out)
	}()

	return out, nil
}

// Close the channel connection
func (c *Consumer) Close() error {
	if err := c.channel.Close(); err != nil {
		return err
	}
	if err := c.conn.Close(); err != nil {
		return err
	}
	return nil
}

// Acknowledger implements the Acknowledger for AMQP library.
type Acknowledger struct {
	ack amqp.Acknowledger
	id  uint64
}

// Ack signals acknowledgement.
func (a *Acknowledger) Ack() error {
	return a.ack.Ack(a.id, false)
}

// Reject signals rejection. If requeue is false, the job will go to the buried
// queue until Queue.RepublishBuried() is called.
func (a *Acknowledger) Reject(requeue bool) error {
	return a.ack.Reject(a.id, requeue)
}
