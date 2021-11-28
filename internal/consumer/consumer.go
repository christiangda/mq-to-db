// this package is an adapter for the consumer to the broker
package consumer

import (
	"context"
	"sync"

	"github.com/christiangda/mq-to-db/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type BrokerService interface {
	Connect() error
	Start() error
	Consume(ctx context.Context, id string) (<-chan amqp.Delivery, error)
	Close() error
}

type Consumer struct {
	ctx    context.Context
	broker BrokerService

	// Consumers
	ConsumerRunning  *prometheus.GaugeVec
	ConsumerMessages *prometheus.CounterVec
}

func NewMessageConsumer(ctx context.Context, broker BrokerService) *Consumer {
	return &Consumer{
		ctx:    ctx,
		broker: broker,

		ConsumerRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "mq-to-db",
			Subsystem: "consumer",
			Name:      "running",
			Help:      "Number of consumer running",
		},
			[]string{
				// Consumer name
				"name",
			}),
		ConsumerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Subsystem: "consumer",
			Name:      "messages_total",
			Help:      "Number of messages consumed.",
		},
			[]string{
				// Consumer name
				"name",
			}),
	}
}

func (c *Consumer) Connect() error {
	return c.broker.Connect()
}

func (c *Consumer) Start() error {
	return c.broker.Start()
}

func (c *Consumer) Close() error {
	return c.broker.Close()
}

// Consume consume messages from queue system and return the messages as a channel of them
func (c *Consumer) Consume(ctx context.Context, id string) (<-chan model.Messages, error) {
	out := make(chan model.Messages)

	go func() {
		defer close(out)

		msgs, err := c.broker.Consume(ctx, id)
		if err != nil {
			log.Error(err)
			return
		}

		for {
			select {
			case msg := <-msgs:

				log.Tracef("Received message: %s, consumer: %s", msg.Body, id)

				out <- model.Messages{
					MessageID:    msg.MessageId,
					Priority:     model.Priority(msg.Priority),
					Timestamp:    msg.Timestamp,
					ContentType:  msg.ContentType,
					Acknowledger: msg.Acknowledger,
					DeliveryTag:  msg.DeliveryTag,
					Payload:      msg.Body,
				}
				c.ConsumerMessages.With(prometheus.Labels{"name": id}).Inc()

			case <-ctx.Done():

				log.WithFields(log.Fields{"consumer": id}).Warnf("Stoping consumer")
				c.ConsumerRunning.With(prometheus.Labels{"name": id}).Dec()
				// closes the consumer queue and connection
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					c.broker.Close() // TODO: Close the consumer connection not the mq connection
					wg.Done()
				}()
				wg.Wait()

				return
			}
		}

		// log.WithFields(log.Fields{"consumer": id}).Infof("Starting consumer")
		// c.ConsumerRunning.With(prometheus.Labels{"name": id}).Inc()

		// for msg := range msgs {
		// 	out <- model.Messages{
		// 		MessageID:    msg.MessageId,
		// 		Priority:     model.Priority(msg.Priority),
		// 		Timestamp:    msg.Timestamp,
		// 		ContentType:  msg.ContentType,
		// 		Acknowledger: msg.Acknowledger,
		// 		DeliveryTag:  msg.DeliveryTag,
		// 		Payload:      msg.Body,
		// 	}
		// 	c.ConsumerMessages.With(prometheus.Labels{"name": id}).Inc()
		// }
		// close(out)
	}()

	return out, nil
}
