package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/christiangda/mq-to-db/internal/model"
	"github.com/christiangda/mq-to-db/internal/repository"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type Repository interface {
	Store(m model.Messages) repository.Results
}

type Storer struct {
	ctx  context.Context
	repo Repository
	conf Config

	StorageWorkerRunning            *prometheus.GaugeVec
	StorageWorkerMessages           *prometheus.CounterVec
	StorageWorkerProcessingDuration *prometheus.HistogramVec
}

func NewStorer(ctx context.Context, repo Repository, conf Config) *Storer {
	return &Storer{
		ctx:  ctx,
		repo: repo,
		conf: conf,

		StorageWorkerRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "mq-to-db",
			Name:      "storage_worker_running",
			Help:      "Number of Storage Workers running",
		},
			[]string{
				// Storage Worker name
				"name",
			}),
		StorageWorkerMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "mq-to-db",
			Name:      "storage_worker_messages_total",
			Help:      "Number of messages consumed my storage_workers.",
		},
			[]string{
				// Storage Worker name
				"name",
			}),
		StorageWorkerProcessingDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "mq-to-db",
			Name:      "storage_worker_process_duration_seconds",
			Help:      "Amount of time spent storing messages",
			Buckets:   []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15}, // Observation buckets
		},
			[]string{
				// Storage Worker name
				"name",
			}),
	}
}

func (s *Storer) Store(msgs <-chan model.Messages) <-chan repository.Results {
	// Slice of workers
	sliceChanStorageWorkers := make([]<-chan repository.Results, s.conf.StorageWorkers)

	// Start storage workers
	log.WithFields(log.Fields{"concurrency": s.conf.StorageWorkers}).Infof("Starting storage workers")
	for i := 0; i < s.conf.StorageWorkers; i++ {
		// ids for storage workers
		id := fmt.Sprintf("%s-%s-storage-worker-%d", s.conf.HostName, s.conf.ApplicationName, i)
		sliceChanStorageWorkers[i] = s.messageProcessor(s.ctx, id, msgs)
	}

	// Merge all channels from workers in only one channel of type <-chan repository.Results
	return s.mergeResultsChans(s.ctx, sliceChanStorageWorkers...)
}

// This function consume messages from queue system and return the messages as a channel of them
func (s *Storer) messageProcessor(ctx context.Context, id string, chanMsgs <-chan model.Messages) <-chan repository.Results {
	out := make(chan repository.Results)
	go func() {
		defer close(out)

		log.WithFields(log.Fields{"worker": id}).Infof("Starting storage worker")

		// prometheus metrics
		s.StorageWorkerRunning.With(prometheus.Labels{"name": id}).Inc()

		// loop to dispatch the messages read to the channel consummed from storage workers
		for {
			select {

			case m := <-chanMsgs:
				startTime := time.Now()

				r := s.repo.Store(m) // proccess and storage message into db
				r.By = id            // fill who execute it
				out <- r

				s.StorageWorkerMessages.With(prometheus.Labels{"name": id}).Inc()

				s.StorageWorkerProcessingDuration.With(
					prometheus.Labels{"name": id},
				).Observe(time.Since(startTime).Seconds())

			case <-ctx.Done(): // When main routine cancel

				log.WithFields(log.Fields{"worker": id}).Warnf("Stoping storage worker")
				s.StorageWorkerRunning.With(prometheus.Labels{"name": id}).Dec()

				return // go out of the for loop

			}
		}
	}()

	return out
}

// mergeResultsChans merge all the channels of storer.Results in only one
// bassically convert (...<-chan storer.Results) --> (<-chan storer.Results)
func (s *Storer) mergeResultsChans(ctx context.Context, channels ...<-chan repository.Results) <-chan repository.Results {
	var wg sync.WaitGroup
	out := make(chan repository.Results)

	// internal function to merge channels in only one
	multiplex := func(channel <-chan repository.Results) {
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
