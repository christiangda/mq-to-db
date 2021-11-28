package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/christiangda/mq-to-db/internal/messages"
	"github.com/christiangda/mq-to-db/internal/metrics"
	"github.com/christiangda/mq-to-db/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

//go:generate go run github.com/golang/mock/mockgen@v1.6.0 -package=mocks -destination=../../mocks/repository/messages_mocks.go -source=messages.go StorageService

// StorageService is the interface to consume SQL service methods
type StorageService interface {
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
	ctx context.Context
	sql StorageService

	RepositoryMessagesTotal              prometheus.Counter
	RepositoryMessagesErrorsTotal        prometheus.Counter
	RepositorySQLMessagesTotal           prometheus.Counter
	RepositorySQLMessagesErrorsTotal     prometheus.Counter
	RepositorySQLMessagesToDBTotal       prometheus.Counter
	RepositorySQLMessagesToDBErrorsTotal prometheus.Counter
	RepositoryMessagesAckTotal           prometheus.Counter
	RepositoryMessagesRejectedTotal      prometheus.Counter
}

// NewMessageRepository return a new Message Repository instance
func NewMessageRepository(ctx context.Context, sql StorageService, mtrs *metrics.Metrics) *MessageRepository {
	out := &MessageRepository{
		ctx: ctx,
		sql: sql,

		RepositoryMessagesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Name:      "storer_messages_total",
			Help:      "Number of messages processed by storer.",
		},
		),
		RepositoryMessagesErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Name:      "storer_messages_errors_total",
			Help:      "Number of messages with errors processed by storer.",
		},
		),
		RepositorySQLMessagesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Name:      "storer_sql_messages_total",
			Help:      "Number of sql messages processed by storer.",
		},
		),
		RepositorySQLMessagesErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Name:      "storer_sql_messages_errors_total",
			Help:      "Number of sql messages with errors processed by storer.",
		},
		),
		RepositorySQLMessagesToDBTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Name:      "storer_sql_messages_to_db_total",
			Help:      "Number of sql messages sent to database by storer.",
		},
		),
		RepositorySQLMessagesToDBErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Name:      "storer_sql_messages_to_db_errors_total",
			Help:      "Number of sql messages with errors sent to database by storer.",
		},
		),
		RepositoryMessagesAckTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Name:      "storer_messages_ack_total",
			Help:      "Number of messages ack into mq system.",
		},
		),
		RepositoryMessagesRejectedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "mq_to_db",
			Name:      "storer_messages_rejected_total",
			Help:      "Number of messages rejected into mq system.",
		},
		),
	}

	prometheus.MustRegister(out.RepositoryMessagesTotal)
	prometheus.MustRegister(out.RepositoryMessagesErrorsTotal)
	prometheus.MustRegister(out.RepositorySQLMessagesTotal)
	prometheus.MustRegister(out.RepositorySQLMessagesErrorsTotal)
	prometheus.MustRegister(out.RepositorySQLMessagesToDBTotal)
	prometheus.MustRegister(out.RepositorySQLMessagesToDBErrorsTotal)
	prometheus.MustRegister(out.RepositoryMessagesAckTotal)
	prometheus.MustRegister(out.RepositoryMessagesRejectedTotal)

	return out
}

// Store stores a message in the database
func (mr *MessageRepository) Store(msg model.Messages) Results {
	log.Debugf("Processing message: %s", msg.Payload)

	mr.RepositoryMessagesTotal.Inc()

	sqlm, err := messages.NewSQL(msg.Payload) // serialize message payload as SQL message type
	if err != nil {

		mr.RepositoryMessagesErrorsTotal.Inc()
		mr.RepositorySQLMessagesErrorsTotal.Inc()

		if err = msg.Reject(false); err != nil {

			mr.RepositoryMessagesRejectedTotal.Inc()

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
	mr.RepositorySQLMessagesTotal.Inc()

	result, err := mr.sql.ExecContext(mr.ctx, sqlm.Content.Sentence)
	if err != nil {

		mr.RepositoryMessagesErrorsTotal.Inc()
		mr.RepositorySQLMessagesToDBErrorsTotal.Inc()

		if err = msg.Reject(false); err != nil {

			mr.RepositoryMessagesRejectedTotal.Inc()

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

	mr.RepositorySQLMessagesToDBTotal.Inc()

	rows, err := result.RowsAffected()
	if err != nil {
		if err = msg.Reject(false); err != nil {

			mr.RepositoryMessagesRejectedTotal.Inc()

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

			mr.RepositoryMessagesRejectedTotal.Inc()

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
	mr.RepositoryMessagesAckTotal.Inc()

	return Results{RowsAffected: rows}
}
