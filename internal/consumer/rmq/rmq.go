package rmq

import (
	"fmt"
	"time"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/consumer"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type rabbitMQConf struct {
	conn    *amqp.Connection
	channel *amqp.Channel

	address            string
	port               int
	requestedHeartbeat time.Duration
	username           string
	password           string
	virtualHost        string
	queue              struct {
		name       string
		routingKey string
		durable    bool
		autoDelete bool
		exclusive  bool
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

// New create a new rabbitmq consumer
func New(c *config.Config) (consumer.Consumer, error) {

	return &rabbitMQConf{
		address:            c.Consumer.Address,
		port:               c.Consumer.Port,
		requestedHeartbeat: c.Consumer.RequestedHeartbeat,
		username:           c.Consumer.Username,
		password:           c.Consumer.Password,
		virtualHost:        c.Consumer.VirtualHost,
		queue: struct {
			name       string
			routingKey string
			durable    bool
			autoDelete bool
			exclusive  bool
			args       map[string]interface{}
		}{
			c.Consumer.Queue.Name,
			c.Consumer.Queue.RoutingKey,
			c.Consumer.Queue.Durable,
			c.Consumer.Queue.AutoDelete,
			c.Consumer.Queue.Exclusive,
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
func (c *rabbitMQConf) Connect() {

	conn, err := amqp.DialConfig(
		fmt.Sprintf("amqp://%s:%s@%s:%d/", c.username, c.username, c.address, c.port),
		amqp.Config{
			Heartbeat: c.requestedHeartbeat,
			Vhost:     c.virtualHost,
		},
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
func (c *rabbitMQConf) Consume() <-chan consumer.Messages {

	msgs, err := c.channel.Consume(
		c.queue.name,
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Fatal(err)
	}

	out := make(chan consumer.Messages)

	// the best way to consume a channel!
	// go func() {
	// 	for msg := range msgs {
	// 		out <- queue.Messages{
	// 			Payload: msg.Body,
	// 			Length:  0,
	// 		}
	// 	}
	// 	close(out)
	// }()

	for msg := range msgs {
		out <- consumer.Messages{
			ConsumerTag:     msg.ConsumerTag,
			ContentEncoding: msg.ContentEncoding,
			ContentType:     msg.ContentType,
			MessageId:       msg.MessageId,
			Timestamp:       msg.Timestamp,
			Exchange:        msg.Exchange,
			RoutingKey:      msg.RoutingKey,
			Payload:         msg.Body,
		}
	}
	close(out)

	return out
}

// Close the channel connection
func (c *rabbitMQConf) Close() {
	c.channel.Close()
	c.conn.Close()
}
