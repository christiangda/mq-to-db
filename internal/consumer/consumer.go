package consumer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/christiangda/mq-to-db/internal/messages"
	"github.com/christiangda/mq-to-db/internal/storage"
	log "github.com/sirupsen/logrus"
)

// This package is an abstraction layer for queue consumers
// any kind of consumer could implements interfaces here
// additionally could be used to create tests stubs

// Consumer interface to be implemented for any kind of queue consumer
type Consumer interface {
	Connect()
	Consume(id string) (<-chan Messages, error)
	Close() error
}

// Priority represents a priority level for message queue
type Priority uint8

// Acknowledger represents the object in charge of acknowledgement
type Acknowledger interface {
	Ack() error
	Reject(requeue bool) error
}

// Messages represent the structure received into the consumer
type Messages struct {
	ContentType     string
	ContentEncoding string
	MessageID       string
	Priority        Priority
	ConsumerTag     string
	Timestamp       time.Time
	Exchange        string
	RoutingKey      string
	Payload         []byte
	Acknowledger
}

// Ack is called when the job is finished.
func (m *Messages) Ack() error {
	if m.Acknowledger == nil {
		return errors.New("Error acknowledging message: " + m.MessageID)
	}
	return m.Acknowledger.Ack()
}

// Reject is called when the job errors. The parameter is true if and only if the
// job should be put back in the queue.
func (m *Messages) Reject(requeue bool) error {
	if m.Acknowledger == nil {
		return errors.New("Error rejecting message: " + m.MessageID)
	}
	return m.Acknowledger.Reject(requeue)
}

// Worker is a job processor witch imply consume a message
// from queue and store this into the Database
type Worker struct {
	ctx context.Context
	wg  *sync.WaitGroup
	id  string
	st  storage.Store
}

// NewWorker execute a new worker
func NewWorker(ctx context.Context, wg *sync.WaitGroup, id string, st storage.Store) *Worker {
	return &Worker{
		ctx: ctx,
		wg:  wg,
		id:  id,
		st:  st,
	}
}

// Start a worker
func (w *Worker) Start(msgs <-chan Messages, err error) {

	if err != nil {
		log.Errorf("Worker: %v, Error iterating over consumer: %s", w.id, err)
		return
	}

	defer w.wg.Done()
	log.Infof("Worker: %v, Starting worker", w.id)

	for {
		select {

		case <-w.ctx.Done(): // Received application context cancellation
			log.Infof("Worker: %v, Application context cancel() received", w.id)
			log.Infof("Worker: %v, Stoping worker", w.id)
			return // avoid leaking of this goroutine when ctx is done.

		case qcm := <-msgs:

			log.Infof("Worker: %v, Consumed message Payload: %s", w.id, qcm.Payload)

			// try to convert the message payload to a SQL message type
			sqlm, err := messages.NewSQL(qcm.Payload)
			if err != nil {
				log.Errorf("Worker: %v, Error creating SQL Message: %s", w.id, err)

				if err := qcm.Reject(false); err != nil {
					log.Errorf("Worker: %v, Error rejecting rabbitmq message: %v", w.id, err)
				}
			} else {

				res, err := w.st.ExecContext(w.ctx, sqlm.Content.Sentence)
				if err != nil {
					log.Errorf("Worker: %v, Error storing SQL payload: %v", w.id, err)

					if err := qcm.Reject(false); err != nil {
						log.Errorf("Worker: %v, Error rejecting rabbitmq message: %v", w.id, err)
					}
				} else {

					if err := qcm.Ack(); err != nil {
						log.Errorf("Worker: %v, Error executing ack on rabbitmq message: %v", w.id, err)
					}

					log.Debugf("Worker: %v, SQL message: %s", w.id, sqlm.ToJSON())

					r, err := res.RowsAffected()
					if err != nil {
						log.Errorf("Worker: %v, Error getting SQL result id: %v", w.id, err)
					}
					log.Debugf("Worker: %v, DB Execution Result: %v", w.id, r)
				}

			}
		}
	}
}
