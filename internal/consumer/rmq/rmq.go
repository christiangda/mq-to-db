package rmq

import (
	"time"

	"github.com/christiangda/mq-to-db/internal/consumer"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// RMQ is a RabbitMQ consumer configuration
// Implement Consumer.Consumer interface
type RMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	closed  <-chan *amqp.Error

	name string
	uri  string

	requestedHeartbeat time.Duration
	virtualHost        string
	queue              struct {
		name          string
		routingKey    string
		durable       bool
		autoDelete    bool
		exclusive     bool
		autoACK       bool
		PrefetchCount int
		PrefetchSize  int
		args          map[string]interface{}
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
func New(c *consumer.Config) (*RMQ, error) {
	uri, err := c.GetURI()
	if err != nil {
		return nil, err
	}

	return &RMQ{
		name:               c.Name,
		uri:                uri,
		requestedHeartbeat: c.RequestedHeartbeat,
		virtualHost:        c.VirtualHost,
		queue: struct {
			name          string
			routingKey    string
			durable       bool
			autoDelete    bool
			exclusive     bool
			autoACK       bool
			PrefetchCount int
			PrefetchSize  int
			args          map[string]interface{}
		}{
			c.Queue.Name,
			c.Queue.RoutingKey,
			c.Queue.Durable,
			c.Queue.AutoDelete,
			c.Queue.Exclusive,
			c.Queue.AutoACK,
			c.Queue.PrefetchCount,
			c.Queue.PrefetchSize,
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
func (c *RMQ) Connect() error {
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
	// defer conn.Close()
	c.conn = conn

	c.closed = conn.NotifyClose(make(chan *amqp.Error))

	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	// defer ch.Close()
	c.channel = ch

	err = ch.Qos(
		c.queue.PrefetchCount,
		c.queue.PrefetchSize,
		false, // global
	)

	if err != nil {
		return err
	}

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
func (c *RMQ) Consume(id string) (<-chan consumer.Messages, error) {
	// //TODO: define the buffer size
	out := make(chan consumer.Messages, 10)

	go func() {
		defer close(out)

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
			log.Error(err)
		}

	loop:
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					log.Warn("RabbitMQ Consume: Channel closed")
					break loop
				}

				out <- consumer.Messages{
					MessageID:    msg.MessageId,
					Priority:     consumer.Priority(msg.Priority),
					Timestamp:    msg.Timestamp,
					ContentType:  msg.ContentType,
					Acknowledger: &Acknowledger{msg.Acknowledger, msg.DeliveryTag},
					Payload:      msg.Body,
				}

			case <-c.closed:
				log.Fatal("RabbitMQ Connection closed")
				c.Close()
				break loop
			}
		}
	}()

	return out, nil
}

// Close the channel connection
func (c *RMQ) Close() error {
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
