package worker

import (
	"context"
	"fmt"
	"sync"

	"github.com/christiangda/mq-to-db/internal/consumer"
	log "github.com/sirupsen/logrus"
)

// Pool is a worker group that runs a number of tasks at a
// configured concurrency.
type Pool struct {
	ctx context.Context // app context
	wg  *sync.WaitGroup // workers coordinator

	name       string
	maxWorkers int
	Workers    []Worker

	// This is used to comunicate with workers and send the job, once it is finished they need to send the result
	// to WorkQueue.
	// Good reference: https://www.goin5minutes.com/blog/channel_over_channel/
	WorkerPool chan chan consumer.Messages
	WorkQueue  chan consumer.Messages
}

// NewPool initializes a new pool with the given tasks and
// at the given concurrency.
func NewPool(ctx context.Context, wg *sync.WaitGroup, maxWorkers int, name string) *Pool {
	return &Pool{
		ctx:        ctx,
		wg:         wg,
		maxWorkers: maxWorkers,

		name:       name,
		WorkerPool: make(chan chan consumer.Messages, maxWorkers), // buffered channel
		WorkQueue:  make(chan consumer.Messages),
	}
}

// ConsumeFrom runs all work within the pool and blocks until it's
// finished.
func (p *Pool) ConsumeFrom(cFunc func(id string) (<-chan consumer.Messages, error)) {

	var msgs <-chan consumer.Messages
	var err error

	// Start workers
	for i := 0; i < p.maxWorkers; i++ {

		// creates a worker id
		id := fmt.Sprintf("%s-w-%d", p.name, i)

		log.Printf("Creating worker: %s", id)
		worker := NewWorker(id, p.WorkerPool)
		worker.Start()

		// register in dispatcher's workers
		//p.Workers = append(p.Workers, worker)

		// execute consumer function
		msgs, err = cFunc(id)
		if err != nil {
			log.Error(err)
		}
	}

	// this is the dispatcher logic
	go func(msgs <-chan consumer.Messages) {
		for {

			select {

			case job := <-msgs:

				log.Info("Received job request")

				// a job request has been received
				go func(job consumer.Messages) {
					// try to obtain a worker job channel that is available.
					// this will block until a worker is idle
					jobChannel := <-p.WorkerPool

					// dispatch the job to the worker job channel
					jobChannel <- job
				}(job)
			}
		}
	}(msgs)
}

// Worker encapsulates a work item that should go in a work
// pool.
type Worker struct {
	ID        string
	JobsQueue chan consumer.Messages

	workerPool chan chan consumer.Messages
	quit       chan bool
}

// NewWorker return a new worker
func NewWorker(id string, pool chan chan consumer.Messages) Worker {
	return Worker{
		ID:        id,
		JobsQueue: make(chan consumer.Messages),

		workerPool: pool,
		quit:       make(chan bool),
	}
}

// Start initialize a worker
func (w Worker) Start() {

	go func() {
		for {
			// register the current worker into the worker queue.
			// this step news to be done
			w.workerPool <- w.JobsQueue

			select {
			case job := <-w.JobsQueue:

				go func(m consumer.Messages) {

				}(job)

			case <-w.quit:
				// we have received a signal to stop
				return
			}
		}
	}()
}

// Stop signals the worker to stop listening for work requests.
func (w Worker) Stop() {
	go func() { // non-blocking call
		w.quit <- true
	}()
}
