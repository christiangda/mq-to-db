package rmq

import (
	"fmt"

	"github.com/christiangda/mq-to-db/internal/config"
	"github.com/christiangda/mq-to-db/internal/queue"
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
	channel *amqp.Channel

	server                   string
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
func New(c *config.Config) (queue.Consummer, error) {

	// TODO: Improve error message and validations

	if c.RabbitMQ.Server == "" {
		return nil, ErrRabbitMqServerAddressEmpty
	}

	if c.RabbitMQ.Port == 0 {
		return nil, ErrRabbitMQServerPortEmpty
	}

	if c.RabbitMQ.Queue.Name == "" {
		return nil, ErrRabbitMQServerQueueEmpty
	}

	return &rabbitMQConf{
		server:                   c.RabbitMQ.Server,
		port:                     c.RabbitMQ.Port,
		requestedHeartbeat:       c.RabbitMQ.RequestedHeartbeat,
		connectionTimeout:        c.RabbitMQ.ConnectionTimeout,
		networkRecoveryInterval:  c.RabbitMQ.NetworkRecoveryInterval,
		consumingQuote:           c.RabbitMQ.ConsumingQuote,
		automaticRecoveryEnabled: c.RabbitMQ.AutomaticRecoveryEnabled,
		username:                 c.RabbitMQ.Username,
		password:                 c.RabbitMQ.Password,
		virtualHost:              c.RabbitMQ.VirtualHost,
		isNoAck:                  c.RabbitMQ.IsNoAck,
		exclusive:                c.RabbitMQ.Exclusive,
		queue: struct {
			name       string
			routingKey string
			durable    bool
			autoDelete bool
			args       map[string]interface{}
		}{
			c.RabbitMQ.Queue.Name,
			c.RabbitMQ.Queue.RoutingKey,
			c.RabbitMQ.Queue.Durable,
			c.RabbitMQ.Queue.AutoDelete,
			c.RabbitMQ.Queue.Args,
		},
		exchange: struct {
			name       string
			kind       string
			durable    bool
			autoDelete bool
			args       map[string]interface{}
		}{
			c.RabbitMQ.Exchange.Name,
			c.RabbitMQ.Exchange.Kind,
			c.RabbitMQ.Exchange.Durable,
			c.RabbitMQ.Exchange.AutoDelete,
			c.RabbitMQ.Exchange.Args,
		},
	}, nil
}

func (c *rabbitMQConf) Connect() {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/", c.username, c.username, c.server, c.port))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ server \"%s:%d\" with user \"%s\" ", c.server, c.port, c.username)
	}
	defer conn.Close()

	ch, err := conn.Channel()
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
		log.Fatal("Failed to declare an exchange \"%s\"", c.exchange.name)
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
		log.Fatal("Failed to declare a queue \"%s\"", c.queue.name)
	}

	err = c.channel.QueueBind(
		q.Name,
		c.queue.routingKey,
		c.exchange.name,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to bind a queue \"%s\"", c.queue.name)
	}
}

func (c *rabbitMQConf) Consume() <-chan queue.Messages {
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
		log.Fatal("Failed to register a consumer queue \"%s\"", c.queue.name)
	}

	out := make(chan queue.Messages)

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
		out <- queue.Messages{
			Payload: msg.Body,
			Length:  0,
		}
	}
	close(out)

	return out
}

func (c *rabbitMQConf) Close() {
	c.Close()
}
