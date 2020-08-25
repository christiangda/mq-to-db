package dispatcher

import (
	"context"
	"fmt"
	"sync"

	"github.com/christiangda/mq-to-db/internal/consumer"
	"github.com/christiangda/mq-to-db/internal/storage"
	log "github.com/sirupsen/logrus"
)

// Processor type is like any function with receive (ctx context.Context, m consumer.Messages, st storage.Store) arguments
// and proccess a m message to store into the st Store.
type Processor func(ctx context.Context, m consumer.Messages, st storage.Store)

// MessagesChannel receive messages to be processed
type MessagesChannel chan consumer.Messages

// MessagesQueue shared with every worker into the pool
type MessagesQueue chan chan consumer.Messages

// Pool is the link between consumer messages and workers
type Pool struct {
	messages MessagesChannel // consumer puts messages here
	queue    MessagesQueue   // shared Messages (channel) between the workers

	ctx context.Context // app context
	wg  sync.WaitGroup  // workers coordinator

	workers    map[string]*worker
	numWorkers int
	name       string
}

// NewPool returns a new pool. A Pool is a mechanisms to comunicate consumer messages and workers
func NewPool(ctx context.Context, num int, namePrefix string, p Processor, st storage.Store) *Pool {

	log.Infof("Creating workers pool: %s, with: %d workers", namePrefix, num)

	ws := make(map[string]*worker)
	ms := make(MessagesChannel)
	q := make(MessagesQueue)

	for i := 0; i < num; i++ {

		// creates a worker id
		id := fmt.Sprintf("%s-w-%d", namePrefix, i)

		log.Printf("Creating worker: %s", id)
		w := newWorker(ctx, id, q, p, st)
		ws[id] = w
	}

	return &Pool{
		messages: ms,
		queue:    q,

		ctx:        ctx,
		wg:         sync.WaitGroup{},
		workers:    ws,
		numWorkers: num,
		name:       namePrefix,
	}
}

// Start the pool of workers and wait until Proccess is called to start processing
func (p *Pool) Start() *Pool {

	// Start workers
	for id, w := range p.workers {
		log.Infof("Starting worker: %s", id)
		w.start()
	}

	// dispatching messages between workers
	go func() {
		for {
			select {
			case msgs := <-p.messages: // listen to a submitted job on messages channel
				qChan := <-p.queue // pull out an available worker from queue
				qChan <- msgs      // submit the messages on the available worker
			}
		}
	}()

	return p
}

// Proccess consume from the message channel and using the processor function proccess these
func (p *Pool) Proccess(msgs <-chan consumer.Messages) {

	log.Info("Starting to process with workers poll")

	go func() { // needs to run into routine, doesn't block main routine
		// put messages on a channel shared with all workers
		for m := range msgs {
			p.messages <- m
		}

	}()
}

// Stop the pool of workers gracefully
func (p *Pool) Stop() {
	go func() {
		for id, w := range p.workers {
			log.Warnf("Stopping worker: %s", id)
			w.stop()
		}
	}()
}

// worker encapsulates a work item that should go in a work
// pool.
type worker struct {
	id   string
	quit chan bool
	ctx  context.Context // app context

	messages MessagesChannel
	queue    MessagesQueue

	processor Processor // The function executed by workers
	st        storage.Store
}

// NewWorker return a new worker
func newWorker(ctx context.Context, id string, q MessagesQueue, p Processor, st storage.Store) *worker {
	return &worker{
		id:   id,
		ctx:  ctx,
		quit: make(chan bool),

		queue:    q,
		messages: make(MessagesChannel),

		processor: p,
		st:        st,
	}
}

// Start a worker
func (w *worker) start() {
	go func() {
		for { // run until no messages to consume or send data over quit channel

			log.Debugf("Worker: %s ready", w.id)
			// when available, put the messages again on the queue
			// and wait to receive a message
			w.queue <- w.messages

			select {

			case m := <-w.messages: // get one message from the messages queue
				// process the messages
				w.processor(w.ctx, m, w.st)
				log.Debugf("Worker: %s done", w.id)
			case <-w.quit:
				// This only occurs when worker is processing and you send quit signal
				close(w.queue) // tell to pool dispatcher that no send more mesages, channel is closed
				log.Debug("Worker: %s stopped", w.id)
				return
			}
		}
	}()
}

// Stop signals the worker to stop listening for work requests.
func (w worker) stop() {
	go func() { // non-blocking call
		w.quit <- true
	}()
}
