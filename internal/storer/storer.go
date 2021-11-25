package storer

import (
	"context"
	"encoding/json"

	"github.com/christiangda/mq-to-db/internal/consumer"
	"github.com/christiangda/mq-to-db/internal/messages"
	"github.com/christiangda/mq-to-db/internal/metrics"
	"github.com/christiangda/mq-to-db/internal/storage"
	log "github.com/sirupsen/logrus"
)

// Results is a return value from processor
type Results struct {
	By           string      // who belongs this results
	RowsAffected int64       // Numbers of row affected by the execution of the query
	Reason       string      // Why the error
	Content      interface{} // only filled when error exist
	Error        error       // Error occurred if exist
}

// ToJSON export the configuration in JSON format
func (r *Results) ToJSON() string {
	out, _ := json.Marshal(r)
	return string(out)
}

// Storer ...
type Storer interface {
	Store(m consumer.Messages) Results
}

// storerConf is a type of function that store the consumer.Messages
type storerConf struct {
	ctx  context.Context
	st   storage.Store
	mtrs *metrics.Metrics
}

// New ...
func New(ctx context.Context, st storage.Store, mtrs *metrics.Metrics) Storer {
	return &storerConf{
		ctx:  ctx,
		st:   st,
		mtrs: mtrs,
	}
}

// Store ...
func (s *storerConf) Store(m consumer.Messages) Results {
	log.Debugf("Processing message: %s", m.Payload)

	s.mtrs.StorerMessagesTotal.Inc()

	sqlm, err := messages.NewSQL(m.Payload) // serialize message payload as SQL message type
	if err != nil {

		s.mtrs.StorerMessagesErrorsTotal.Inc()
		s.mtrs.StorerSQLMessagesErrorsTotal.Inc()

		if err = m.Reject(false); err != nil {

			s.mtrs.StorerMessagesRejectedTotal.Inc()

			return Results{
				Error:   err,
				Content: m.MessageID,
				Reason:  "Impossible to serialize message to SQL type and reject the message from queue system",
			}
		}
		return Results{
			Error:   err,
			Content: m.Payload,
			Reason:  "Impossible to serialize message to SQL type",
		}
	}

	log.Debugf("Executing SQL sentence: %s", sqlm.Content.Sentence)
	s.mtrs.StorerSQLMessagesTotal.Inc()

	result, err := s.st.ExecContext(s.ctx, sqlm.Content.Sentence)
	if err != nil {

		s.mtrs.StorerMessagesErrorsTotal.Inc()
		s.mtrs.StorerSQLMessagesToDBErrorsTotal.Inc()

		if err = m.Reject(false); err != nil {

			s.mtrs.StorerMessagesRejectedTotal.Inc()

			return Results{
				Error:   err,
				Content: m.MessageID,
				Reason:  "Impossible to execute sentence into database and reject the message from queue system",
			}
		}
		return Results{
			Error:   err,
			Content: sqlm.Content.Sentence,
			Reason:  "Impossible to execute sentence into database",
		}
	}

	s.mtrs.StorerSQLMessagesToDBTotal.Inc()

	rows, err := result.RowsAffected()
	if err != nil {
		if err = m.Reject(false); err != nil {

			s.mtrs.StorerMessagesRejectedTotal.Inc()

			return Results{
				Error:   err,
				Content: m.MessageID,
				Reason:  "Impossible get result from database and reject the message from queue system",
			}
		}
		return Results{
			Error:   err,
			Content: sqlm.Content.Sentence,
			Reason:  "Impossible get result from database",
		}
	}
	log.Debugf("SQL Execution return: %v", rows)

	if err := m.Ack(); err != nil {
		if err = m.Reject(false); err != nil {

			s.mtrs.StorerMessagesRejectedTotal.Inc()

			return Results{
				Error:   err,
				Content: m.MessageID,
				Reason:  "Impossible execute ack and reject the message from queue system",
			}
		}
		return Results{
			Error:   err,
			Content: sqlm.Content.Sentence,
			Reason:  "Impossible execute ack into the queue system",
		}
	}
	log.Debugf("Ack the message: %s", sqlm.ToJSON())
	s.mtrs.StorerMessagesAckTotal.Inc()

	return Results{RowsAffected: rows}
}
