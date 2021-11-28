package queue

import (
	"time"

	"github.com/christiangda/mq-to-db/internal/model"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// RabbitMQ is a RabbitMQ consumer configuration
// Implement Consumer.Consumer interface
type RabbitMQ struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	notifyClose chan *amqp.Error

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
func NewRabbitMQ(c *Config) (*RabbitMQ, error) {
	uri, err := c.GetURI()
	if err != nil {
		return nil, err
	}

	return &RabbitMQ{
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
func (rmq *RabbitMQ) Connect() error {
	amqpConfig := amqp.Config{}

	amqpConfig.Heartbeat = rmq.requestedHeartbeat
	if rmq.virtualHost != "" {
		amqpConfig.Vhost = rmq.virtualHost
	}

	if rmq.conn == nil || rmq.conn.IsClosed() {
		conn, err := amqp.DialConfig(
			rmq.uri,
			amqpConfig,
		)
		if err != nil {
			return err
		}

		rmq.conn = conn
		rmq.notifyClose = conn.NotifyClose(make(chan *amqp.Error))
	}

	ch, err := rmq.conn.Channel()
	if err != nil {
		return err
	}
	// defer ch.Close()
	rmq.channel = ch

	err = ch.Qos(
		rmq.queue.PrefetchCount,
		rmq.queue.PrefetchSize,
		false, // global
	)

	if err != nil {
		return err
	}

	err = rmq.channel.ExchangeDeclare(
		rmq.exchange.name,
		rmq.exchange.kind,
		rmq.exchange.durable,
		rmq.exchange.autoDelete,
		false, // internal
		false, // no-wait
		rmq.exchange.args,
	)
	if err != nil {
		return err
	}

	q, err := rmq.channel.QueueDeclare(
		rmq.queue.name,
		rmq.queue.durable,
		rmq.queue.autoDelete,
		rmq.queue.exclusive,
		false, // no-wait
		rmq.queue.args,
	)
	if err != nil {
		return err
	}

	err = rmq.channel.QueueBind(
		q.Name,
		rmq.queue.routingKey,
		rmq.exchange.name,
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

// Consume messages from the queue channel
func (rmq *RabbitMQ) Consume(id string) (<-chan model.Messages, error) {
	// Register a consumer
	msgs, err := rmq.channel.Consume(
		rmq.queue.name,
		id,                  // consumer id
		rmq.queue.autoACK,   // auto-ack
		rmq.queue.exclusive, // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)
	if err != nil {
		return nil, err
	}

	// This channels is used to be filled by messages comming from
	// the queue system
	// This is part of "producer-consume queue pattern"
	out := make(chan model.Messages, len(msgs)+1)

	// NOTE: This is necessary to consume the original channel without blocking it
	go func() {
		defer close(out)
		for {
			select {
			case msg := <-msgs:
				log.Tracef("Received message: %s, consumer: %s", msg.Body, id)

				out <- model.Messages{
					MessageID:    msg.MessageId,
					Priority:     model.Priority(msg.Priority),
					Timestamp:    msg.Timestamp,
					ContentType:  msg.ContentType,
					Acknowledger: &Acknowledger{msg.Acknowledger, msg.DeliveryTag},
					Payload:      msg.Body,
				}
			case err := <-rmq.notifyClose:
				log.Warnf("RabbitMQ connection close, err: %s", err.Error())
				time.Sleep(5 * time.Second)
				// rmq.Connect()
			}
		}

		// for msg := range msgs {
		// 	out <- model.Messages{
		// 		MessageID:    msg.MessageId,
		// 		Priority:     model.Priority(msg.Priority),
		// 		Timestamp:    msg.Timestamp,
		// 		ContentType:  msg.ContentType,
		// 		Acknowledger: &Acknowledger{msg.Acknowledger, msg.DeliveryTag},
		// 		Payload:      msg.Body,
		// 	}

		// 	// // delegates an acknowledgement
		// 	// if err := msg.Ack(false); err != nil {
		// 	// 	log.Warnf("Error delegating acknowle message: %s", err)
		// 	// }
		// }
		// 		close(out)
	}()

	return out, nil
}

// Close the channel connection
func (rmq *RabbitMQ) Close() error {
	if err := rmq.channel.Close(); err != nil {
		return err
	}
	if err := rmq.conn.Close(); err != nil {
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
