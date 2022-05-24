package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/christiangda/mq-to-db/internal/consumer"
	"github.com/christiangda/mq-to-db/internal/messages"
	"github.com/christiangda/mq-to-db/internal/metrics"
	log "github.com/sirupsen/logrus"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -package=mocks -destination=../../mocks/repository/messages_mocks.go -source=messages.go SQLService

// SQLService is the interface to consume SQL service methods
type SQLService interface {
	ExecContext(ctx context.Context, q string) (sql.Result, error)
}

// type MetricsService interface {
// 	Inc()
// }

// Results is returned by the Store method
type Results struct {
	By           string      // who belongs this results
	RowsAffected int64       // Numbers of row affected by the execution of the query
	Reason       string      // Why the error
	Content      interface{} // only filled when error exist
	Error        error       // Error occurred if exist
}

func (r *Results) ToJSON() string {
	out, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}

	return string(out)
}

// MessageRepository represents a message repository
type MessageRepository struct {
	ctx  context.Context
	sql  SQLService
	mtrs *metrics.Metrics
}

// NewMessageRepository return a new Message Repository instance
func NewMessageRepository(ctx context.Context, sql SQLService, mtrs *metrics.Metrics) *MessageRepository {
	return &MessageRepository{
		ctx:  ctx,
		sql:  sql,
		mtrs: mtrs,
	}
}

// Store stores a message in the database
func (mr *MessageRepository) Store(msg consumer.Messages) Results {
	log.Debugf("Processing message: %s", msg.Payload)

	mr.mtrs.RepositoryMessagesTotal.Inc()

	sqlm, err := messages.NewSQL(msg.Payload) // serialize message payload as SQL message type
	if err != nil {
		mr.mtrs.RepositoryMessagesErrorsTotal.Inc()
		mr.mtrs.RepositorySQLMessagesErrorsTotal.Inc()

		if err = msg.Reject(false); err != nil {
			mr.mtrs.RepositoryMessagesRejectedTotal.Inc()

			return Results{
				Error:   err,
				Content: msg.MessageID,
				Reason:  "Impossible to serialize message to SQL type and reject the message from queue system",
			}
		}
		return Results{
			Error:   err,
			Content: msg.Payload,
			Reason:  "Impossible to serialize message to SQL type",
		}
	}

	log.Debugf("Executing SQL sentence: %s", sqlm.Content.Sentence)
	mr.mtrs.RepositorySQLMessagesTotal.Inc()

	result, err := mr.sql.ExecContext(mr.ctx, sqlm.Content.Sentence)
	if err != nil {
		mr.mtrs.RepositoryMessagesErrorsTotal.Inc()
		mr.mtrs.RepositorySQLMessagesToDBErrorsTotal.Inc()

		if err = msg.Reject(false); err != nil {
			mr.mtrs.RepositoryMessagesRejectedTotal.Inc()

			return Results{
				Error:   err,
				Content: msg.MessageID,
				Reason:  "Impossible to execute sentence into database and reject the message from queue system",
			}
		}
		return Results{
			Error:   err,
			Content: sqlm.Content.Sentence,
			Reason:  "Impossible to execute sentence into database",
		}
	}

	mr.mtrs.RepositorySQLMessagesToDBTotal.Inc()

	rows, err := result.RowsAffected()
	if err != nil {
		if err = msg.Reject(false); err != nil {
			mr.mtrs.RepositoryMessagesRejectedTotal.Inc()

			return Results{
				Error:   err,
				Content: msg.MessageID,
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

	if err := msg.Ack(); err != nil {
		if err = msg.Reject(false); err != nil {
			mr.mtrs.RepositoryMessagesRejectedTotal.Inc()

			return Results{
				Error:   err,
				Content: msg.MessageID,
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
	mr.mtrs.RepositoryMessagesAckTotal.Inc()

	return Results{RowsAffected: rows}
}
