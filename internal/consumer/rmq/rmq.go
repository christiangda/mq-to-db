package rmq

import (
	"errors"
	"fmt"
	"time"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/consumer"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	uuid "github.com/google/uuid"
)

// Consumer is a RabbitMQ consumer configuration
// Implement Consumer.Consumer interface
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel

	appName            string
	uri                string
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
		appName:            c.Application.Name,
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

	conn, err := amqp.DialConfig(
		c.uri,
		amqpConfig,
	)
	if err != nil {
		log.Fatal(err)
	}
	//defer conn.Close()
	c.conn = conn

	ch, err := c.conn.Channel()
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
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
		log.Fatal(err)
	}

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
func (c *Consumer) Consume() (consumer.Iterator, error) {

	id := c.newConsumerID()

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

	return &Iterator{messages: msgs, ch: c.channel, id: id}, nil
}

// newConsumerID generate a unique consumer id compose
// by '<application name>-<queue name>-<uuid>'
func (c *Consumer) newConsumerID() string {
	return fmt.Sprintf("%s-%s-%s", c.appName, c.queue.name, uuid.New().String())
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

// Iterator iterates over consume messages
// Implements Consumer.Iterator
type Iterator struct {
	id       string
	ch       *amqp.Channel
	messages <-chan amqp.Delivery
}

// Next returns the next message in the iterator.
func (i *Iterator) Next() (*consumer.Messages, error) {

	d, ok := <-i.messages
	if !ok {
		return nil, errors.New("Channel is closed")
	}

	m := &consumer.Messages{}
	m.MessageID = d.MessageId
	m.Priority = consumer.Priority(d.Priority)
	m.Timestamp = d.Timestamp
	m.ContentType = d.ContentType
	m.Acknowledger = &Acknowledger{d.Acknowledger, d.DeliveryTag}
	m.Payload = d.Body

	return m, nil
}

// Close closes the channel of the Iterator.
func (i *Iterator) Close() error {
	if err := i.ch.Cancel(i.id, false); err != nil {
		return err
	}

	return i.ch.Close()
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
