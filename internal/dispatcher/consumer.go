package dispatcher

import (
	"context"
	"fmt"
	"sync"

	"github.com/christiangda/mq-to-db/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// this interface is used to consume the methods from the queue system
type ConsumerService interface {
	Connect() error
	Start() error
	Consume(ctx context.Context, id string) (<-chan model.Messages, error)
	Close() error
}

// Consumer represent the consumer of the queue system
type Consumer struct {
	ctx      context.Context
	consumer ConsumerService
	conf     Config
	messages []<-chan model.Messages

	// Consumers
	DispatcherConsumerRunning  *prometheus.GaugeVec
	DispatcherConsumerMessages *prometheus.CounterVec
}

// NewConsumer create a new consumer
func NewConsumer(ctx context.Context, consumer ConsumerService, conf Config) *Consumer {
	return &Consumer{
		ctx:      ctx,
		consumer: consumer,
		conf:     conf,
		messages: make([]<-chan model.Messages, conf.ConsumerConcurrency),

		DispatcherConsumerRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "mq-to-db",
			Subsystem: "dispatcher",
			Name:      "consumer_running",
			Help:      "Number of consumer running",
		},
			[]string{
				// Consumer name
				"name",
			}),
		DispatcherConsumerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Subsystem: "dispatcher",
			Name:      "consumer_messages_total",
			Help:      "Number of messages consumed",
		},
			[]string{
				// Consumer name
				"name",
			}),
	}
}

func (c *Consumer) Consume() <-chan model.Messages {
	if err := c.consumer.Connect(); err != nil {
		log.Errorf("Error connecting to queue system: %s", err)
	}

	if err := c.consumer.Start(); err != nil {
		log.Errorf("Error starting consumer: %s", err)
	}

	// Start Consumers
	log.WithFields(log.Fields{"concurrency": c.conf.ConsumerConcurrency}).Infof("Starting consumers")
	for i := 0; i < c.conf.ConsumerConcurrency; i++ {
		// ids for consumers
		id := fmt.Sprintf("%s-%s-consumer-%d", c.conf.HostName, c.conf.ApplicationName, i)
		c.messages[i] = c.Consumer(c.ctx, id)
	}

	// Merge all channels from consumers in only one channel of type <-chan consumer.Messages
	return c.mergeMessages(c.ctx, c.messages...)
}

// messageConsumer consume messages from queue system and return the messages as a channel of them
func (c *Consumer) Consumer(ctx context.Context, id string) <-chan model.Messages {
	out := make(chan model.Messages)
	go func() {
		defer close(out)

		log.WithFields(log.Fields{"consumer": id}).Infof("Starting consumer")

		// prometheus metrics
		c.DispatcherConsumerRunning.With(prometheus.Labels{"name": id}).Inc()

		// reading from message queue
		msgs, err := c.consumer.Consume(ctx, id)
		if err != nil {
			log.Error(err)
		}

		// loop to dispatch the messages read to the channel consummed from storage workers
		for m := range msgs {
			select {

			case out <- m: // put messages consumed into the out chan
				c.DispatcherConsumerMessages.With(prometheus.Labels{"name": id}).Inc()

			case <-ctx.Done(): // When main routine cancel

				log.WithFields(log.Fields{"consumer": id}).Warnf("Stoping distpatcher consumer")
				c.DispatcherConsumerRunning.With(prometheus.Labels{"name": id}).Dec()

				// closes the consumer queue and connection
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					c.consumer.Close() // TODO: Close the consumer connection not the mq connection
					wg.Done()
				}()
				wg.Wait()

				return // go out of the for loop
			}
		}
	}()

	return out
}

// this function merge all the channels data receive as slice of channels and return a merged channel with the data
// bassically convert (...<-chan model.Messages) --> (<-chan model.Messages)
func (c *Consumer) mergeMessages(ctx context.Context, channels ...<-chan model.Messages) <-chan model.Messages {
	var wg sync.WaitGroup
	out := make(chan model.Messages)

	// internal function to merge channels in only one
	multiplex := func(channel <-chan model.Messages) {
		defer wg.Done()
		for ch := range channel {
			select {
			case out <- ch:
			case <-ctx.Done():
				return
			}
		}
	}

	// one routine per every channel in channels arg
	wg.Add(len(channels))
	for _, ch := range channels {
		go multiplex(ch)
	}

	// waiting until fished every go multiplex(ch)
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
