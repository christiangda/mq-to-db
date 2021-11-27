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
type Queue interface {
	Connect() error
	Consume(id string) (<-chan model.Messages, error)
	Close() error
}

// Consumer represent the consumer of the queue system
type Consumer struct {
	ctx   context.Context
	queue Queue
	conf  Config

	// Consumers
	ConsumerRunning  *prometheus.GaugeVec
	ConsumerMessages *prometheus.CounterVec
}

// NewConsumer create a new consumer
func NewConsumer(ctx context.Context, queue Queue, conf Config) *Consumer {
	return &Consumer{
		ctx:   ctx,
		queue: queue,
		conf:  conf,

		ConsumerRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "mq-to-db",
			Name:      "consumer_running",
			Help:      "Number of consumer running",
		},
			[]string{
				// Consumer name
				"name",
			}),
		ConsumerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Name:      "consumer_messages_total",
			Help:      "Number of messages consumed my consumers.",
		},
			[]string{
				// Consumer name
				"name",
			}),
	}
}

func (c *Consumer) Consume() <-chan model.Messages {
	// Logic of channels for consumer and for storage
	// it is a go pipeline model https://blog.golang.org/pipelines
	// ********************************************

	// slice of consumers
	// where the consumers will put the chan of consumer.Messages
	// this slice of channels (consumer.Messages) is used to comunicate consumers and storage workers
	// but first every consumer generate a <-chan consumer.Messages witch need to me merge in only
	// one channel before be ready to consume
	sliceChanMessages := make([]<-chan model.Messages, c.conf.ConsumerConcurrency)

	// Start Consumers
	log.WithFields(log.Fields{"concurrency": c.conf.ConsumerConcurrency}).Infof("Starting consumers")
	for i := 0; i < c.conf.ConsumerConcurrency; i++ {
		// ids for consumers
		id := fmt.Sprintf("%s-%s-consumer-%d", c.conf.HostName, c.conf.ApplicationName, i)
		sliceChanMessages[i] = c.messageConsumer(c.ctx, id)
	}

	// Merge all channels from consumers in only one channel of type <-chan consumer.Messages
	return c.mergeMessagesChans(c.ctx, sliceChanMessages...)
}

// messageConsumer consume messages from queue system and return the messages as a channel of them
func (c *Consumer) messageConsumer(ctx context.Context, id string) <-chan model.Messages {
	out := make(chan model.Messages)
	go func() {
		defer close(out)

		log.WithFields(log.Fields{"consumer": id}).Infof("Starting consumer")

		// prometheus metrics
		c.ConsumerRunning.With(prometheus.Labels{"name": id}).Inc()

		// reading from message queue
		msgs, err := c.queue.Consume(id)
		if err != nil {
			log.Error(err)
		}

		// loop to dispatch the messages read to the channel consummed from storage workers
		for m := range msgs {
			select {

			case out <- m: // put messages consumed into the out chan
				c.ConsumerMessages.With(prometheus.Labels{"name": id}).Inc()

			case <-ctx.Done(): // When main routine cancel

				log.WithFields(log.Fields{"consumer": id}).Warnf("Stoping consumer")
				c.ConsumerRunning.With(prometheus.Labels{"name": id}).Dec()

				// closes the consumer queue and connection
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					c.queue.Close() // TODO: Close the consumer connection not the mq connection
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
func (c *Consumer) mergeMessagesChans(ctx context.Context, channels ...<-chan model.Messages) <-chan model.Messages {
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
