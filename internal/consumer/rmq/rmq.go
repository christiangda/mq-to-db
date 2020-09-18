package rmq

import (
	"sync"
	"time"

	"github.com/christiangda/mq-to-db/internal/consumer"
	"github.com/streadway/amqp"
)

// rmq is a RabbitMQ consumer configuration
// Implement Consumer.Consumer interface
type rmq struct {
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
func New(c *consumer.Config) (consumer.Consumer, error) {

	uri := c.GetURI()

	return &rmq{
		name:               c.Name,
		uri:                uri,
		requestedHeartbeat: c.RequestedHeartbeat,
		virtualHost:        c.VirtualHost,
		queue: struct {
			name       string
			routingKey string
			durable    bool
			autoDelete bool
			exclusive  bool
			autoACK    bool
			args       map[string]interface{}
		}{
			c.Queue.Name,
			c.Queue.RoutingKey,
			c.Queue.Durable,
			c.Queue.AutoDelete,
			c.Queue.Exclusive,
			c.Queue.AutoACK,
			c.Queue.Args,
		},
		exchange: struct {
			name       string
			kind       string
			durable    bool
			autoDelete bool
			args       map[string]interface{}
		}{
			c.Exchange.Name,
			c.Exchange.Kind,
			c.Exchange.Durable,
			c.Exchange.AutoDelete,
			c.Exchange.Args,
		},
	}, nil
}

// Connect to RabbitMQ server and channel
func (c *rmq) Connect() error {

	amqpConfig := amqp.Config{}

	amqpConfig.Heartbeat = c.requestedHeartbeat
	if c.virtualHost != "" {
		amqpConfig.Vhost = c.virtualHost
	}

	conn, err := amqp.DialConfig(
		c.uri,
		amqpConfig,
	)
	if err != nil {
		return err
	}
	//defer conn.Close()
	c.conn = conn

	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	//defer ch.Close()
	c.channel = ch

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
		return err
	}

	q, err := c.channel.QueueDeclare(
		c.queue.name,
		c.queue.durable,
		c.queue.autoDelete,
		c.queue.exclusive,
		false, // no-wait
		c.queue.args,
	)
	if err != nil {
		return err
	}

	err = c.channel.QueueBind(
		q.Name,
		c.queue.routingKey,
		c.exchange.name,
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

// Consume messages from the queue channel
func (c *rmq) Consume(id string) (<-chan consumer.Messages, error) {

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

	var wg sync.WaitGroup
	// This channels is used to be filled by messages comming from
	// the queue system
	// This is part of "producer-consume queue pattern"
	out := make(chan consumer.Messages, len(msgs)+1)

	// NOTE: This is necessary to consume the original channel without blocking it
	wg.Add(1)
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
		wg.Done()
	}()

	return out, nil
}

// Close the channel connection
func (c *rmq) Close() error {
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
