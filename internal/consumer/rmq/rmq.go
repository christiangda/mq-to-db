package rmq

import (
	"fmt"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/consumer"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"golang.org/x/xerrors"
)

var (
	log = logrus.New()

	// ErrRabbitMqServerAddressEmpty is returned when
	ErrRabbitMqServerAddressEmpty = xerrors.New("RabbitMQ server IPAddres cannot be empty")

	// ErrRabbitMQServerPortEmpty is returned when
	ErrRabbitMQServerPortEmpty = xerrors.New("RabbitMQ server Port cannot be empty")

	// ErrRabbitMQServerQueueEmpty is returned when
	ErrRabbitMQServerQueueEmpty = xerrors.New("RabbitMQ server Queue cannot be empty")
)

type rabbitMQConf struct {
	conn    *amqp.Connection
	channel *amqp.Channel

	address                  string
	port                     int
	requestedHeartbeat       int
	connectionTimeout        int
	networkRecoveryInterval  int
	consumingQuote           int
	automaticRecoveryEnabled bool
	username                 string
	password                 string
	virtualHost              string
	isNoAck                  bool
	exclusive                bool
	queue                    struct {
		name       string
		routingKey string
		durable    bool
		autoDelete bool
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

	// TODO: Improve error message and validations

	if c.Consumer.Address == "" {
		return nil, ErrRabbitMqServerAddressEmpty
	}

	if c.Consumer.Port == 0 {
		return nil, ErrRabbitMQServerPortEmpty
	}

	if c.Consumer.Queue.Name == "" {
		return nil, ErrRabbitMQServerQueueEmpty
	}

	return &rabbitMQConf{
		address:                  c.Consumer.Address,
		port:                     c.Consumer.Port,
		requestedHeartbeat:       c.Consumer.RequestedHeartbeat,
		connectionTimeout:        c.Consumer.ConnectionTimeout,
		networkRecoveryInterval:  c.Consumer.NetworkRecoveryInterval,
		consumingQuote:           c.Consumer.ConsumingQuote,
		automaticRecoveryEnabled: c.Consumer.AutomaticRecoveryEnabled,
		username:                 c.Consumer.Username,
		password:                 c.Consumer.Password,
		virtualHost:              c.Consumer.VirtualHost,
		isNoAck:                  c.Consumer.IsNoAck,
		exclusive:                c.Consumer.Exclusive,
		queue: struct {
			name       string
			routingKey string
			durable    bool
			autoDelete bool
			args       map[string]interface{}
		}{
			c.Consumer.Queue.Name,
			c.Consumer.Queue.RoutingKey,
			c.Consumer.Queue.Durable,
			c.Consumer.Queue.AutoDelete,
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
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", c.username, c.username, c.address, c.port))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ server \"%s:%d\" with user \"%s\" ", c.address, c.port, c.username)
	}
	//defer conn.Close()
	c.conn = conn

	ch, err := c.conn.Channel()
	if err != nil {
		log.Fatal("Failed to open channel")
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
		log.Fatalf("Failed to declare an exchange \"%s\"", c.exchange.name)
	}

	q, err := c.channel.QueueDeclare(
		c.queue.name,
		c.queue.durable,
		c.queue.autoDelete,
		true,  // exclusive
		false, // no-wait
		c.queue.args,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue \"%s\"", c.queue.name)
	}

	err = c.channel.QueueBind(
		q.Name,
		c.queue.routingKey,
		c.exchange.name,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind a queue \"%s\"", c.queue.name)
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
		log.Fatalf("Failed to register a consumer queue \"%s\"", c.queue.name)
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
