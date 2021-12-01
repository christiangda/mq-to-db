package broker

import (
	"context"
	"errors"
	"os"

	"github.com/cenkalti/backoff"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	conf    *Config

	name string
	uri  string

	BrokerReconnection *prometheus.CounterVec
	BrokerMessages     *prometheus.CounterVec
}

// New create a new rabbitmq consumer with implements consumer.Consumer interface
func NewRabbitMQ(conf *Config) (*RabbitMQ, error) {
	// TODO: validate the config
	uri, err := conf.GetURI()
	if err != nil {
		return nil, err
	}

	return &RabbitMQ{
		uri:  uri,
		conf: conf,

		BrokerReconnection: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Subsystem: "broker",
			Name:      "reconnections_total",
			Help:      "Number of broker reconnections",
		},
			[]string{
				// Consumer name
				"name",
			}),
		BrokerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Subsystem: "broker",
			Name:      "messages_total",
			Help:      "Number of messages consumed.",
		},
			[]string{
				// Consumer name
				"name",
			}),
	}, nil
}

func (rmq *RabbitMQ) Connect() error {
	if rmq.conn == nil || rmq.conn.IsClosed() {

		amqpConfig := amqp.Config{}

		amqpConfig.Heartbeat = rmq.conf.RequestedHeartbeat
		if rmq.conf.VirtualHost != "" {
			amqpConfig.Vhost = rmq.conf.VirtualHost
		}

		conn, err := amqp.DialConfig(
			rmq.uri,
			amqpConfig,
		)
		if err != nil {
			return err
		}
		rmq.conn = conn
	}

	return nil
}

func (rmq *RabbitMQ) Close() error {
	if rmq.conn != nil && !rmq.conn.IsClosed() {
		return rmq.conn.Close()
	}
	return nil
}

func (rmq *RabbitMQ) getConnection() (*amqp.Connection, error) {
	if rmq.conn == nil || rmq.conn.IsClosed() {
		return nil, errors.New("connection is close")
	}

	return rmq.conn, nil
}

func (rmq *RabbitMQ) GetChannel() (*amqp.Channel, error) {
	chn, err := rmq.conn.Channel()
	if err != nil {
		return nil, err
	}

	return chn, nil
}

func (rmq *RabbitMQ) Start() error {
	conn, err := rmq.getConnection()
	if err != nil {
		return err
	}

	go rmq.connectionListener(conn.NotifyClose(make(chan *amqp.Error)))

	chn, err := conn.Channel()
	if err != nil {
		return err
	}

	if err = chn.ExchangeDeclare(
		rmq.conf.Exchange.Name,
		rmq.conf.Exchange.Kind,
		rmq.conf.Exchange.Durable,
		rmq.conf.Exchange.AutoDelete,
		false, // internal
		false, // no-wait
		rmq.conf.Exchange.Args,
	); err != nil {
		return err
	}

	q, err := chn.QueueDeclare(
		rmq.conf.Queue.Name,
		rmq.conf.Queue.Durable,
		rmq.conf.Queue.AutoDelete,
		rmq.conf.Queue.Exclusive,
		false, // no-wait
		rmq.conf.Queue.Args,
	)
	if err != nil {
		return err
	}

	if err = chn.QueueBind(
		q.Name,
		rmq.conf.Queue.RoutingKey,
		rmq.conf.Exchange.Name,
		false, // no-wait
		nil,
	); err != nil {
		return err
	}

	if err := chn.Qos(rmq.conf.Queue.PrefetchCount, 0, false); err != nil {
		return err
	}

	rmq.channel = chn

	return nil
}

// consume creates a new consumer and starts consuming the messages.
// If this is called more than once, there will be multiple consumers
// running. All consumers operate in a round robin fashion to distribute
// message load.
func (rmq *RabbitMQ) Consume(ctx context.Context, id string) (<-chan amqp.Delivery, error) {
	out := make(chan amqp.Delivery)
	go func() {
		defer close(out)

		msgs, err := rmq.channel.Consume(
			rmq.conf.Queue.Name,
			id,                       // consumer id
			rmq.conf.Queue.AutoACK,   // auto-ack
			rmq.conf.Queue.Exclusive, // exclusive
			false,                    // no-local
			false,                    // no-wait
			nil,                      // args
		)
		if err != nil {
			log.Errorf("Error consuming: %s", err)
			return
		}

	loop:
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					log.Warn("RabbitMQ Consume: Channel closed")
					break loop
				}

				log.Tracef("Received message: %s, consumer: %s", msg.Body, id)
				out <- msg
				rmq.BrokerMessages.With(prometheus.Labels{"name": id}).Inc()

			case <-ctx.Done():

				log.WithFields(log.Fields{"consumer": id}).Warnf("Stoping consumer")
				rmq.BrokerMessages.With(prometheus.Labels{"name": id}).Desc()

				rmq.channel.Close()
				rmq.conn.Close()
				break loop
			}
		}
	}()

	return out, nil
}

// connectionListener attempts to reconnect to the server and
// reopens the channel for set amount of time if the connection is
// closed unexpectedly.
func (rmq *RabbitMQ) connectionListener(closed <-chan *amqp.Error) {
	// If you do not want to reconnect in the case of manual disconnection
	// via RabbitMQ UI or Server restart, handle `amqp.ConnectionForced`
	// error code.
	err := <-closed
	if err != nil {
		repeater := backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10), context.Background())

		if err := backoff.Retry(func() error {
			rmq.BrokerReconnection.WithLabelValues("rabbitmq").Inc()

			// Attempt to get the connection to the database
			if err := rmq.Connect(); err == nil {
				log.Info("Reconnected")

				if err := rmq.Start(); err == nil {
					return nil
				}
			}

			return nil
		}, repeater); err != nil {
			os.Exit(0)
		}
	} else {
		log.Info("Connection closed normally, will not reconnect")
		os.Exit(0)
	}
}
