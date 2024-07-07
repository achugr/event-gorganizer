package repository

import (
	"cloud.google.com/go/datastore"
	"context"
	"event-gorganizer/internal/model"
	"fmt"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type EventRepository struct {
	dsClient *datastore.Client
}

type GcpSettings struct {
	ProjectName         string
	CredentialsFilePath *string
}

func NewEventRepository(ctx context.Context, settings GcpSettings) (*EventRepository, error) {
	opts := make([]option.ClientOption, 0)
	if settings.CredentialsFilePath != nil {
		opts = append(opts, option.WithCredentialsFile(*settings.CredentialsFilePath))
	}
	dsClient, err := datastore.NewClient(ctx, settings.ProjectName, opts...)
	if err != nil {
		log.Error().Msgf("Failed to initialize Datastore client: %s.", err)
		return nil, err
	}

	return &EventRepository{
		dsClient: dsClient,
	}, nil
}

func (r *EventRepository) Save(ctx context.Context, event *model.Event) (*model.Event, error) {
	key := datastore.NameKey("Event", event.Id(), nil)
	_, err := r.dsClient.Put(ctx, key, event)
	if err != nil {
		log.Error().Msgf("Failed to save the event %s: %s", event.Id(), err)
		return nil, err
	}
	return event, nil
}

func (r *EventRepository) GetActiveEvent(ctx context.Context, chatId int64) (*model.Event, error) {
	query := datastore.NewQuery("Event").
		FilterField("ChatId", "=", chatId).
		FilterField("Active", "=", true).
		Limit(1)

	iter := r.dsClient.Run(ctx, query)
	var event model.Event
	_, err := iter.Next(&event)
	if err != nil && err != iterator.Done {
		log.Error().Msgf("Failed to get an event for the chat %s: %s.", chatId, err)
		return nil, err
	}
	return &event, nil
}

func ExecTx[R any](ctx context.Context, repo *EventRepository, readonly bool, f func() (*R, error)) (*R, error) {
	var opts []datastore.TransactionOption
	if readonly {
		opts = []datastore.TransactionOption{datastore.ReadOnly}
	}
	var r *R
	_, err := repo.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		result, e := f()
		if e != nil {
			log.Error().Err(e)
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("tx err: %v, rb err: %v", e, rbErr)
			}
		}
		r = result
		return nil
	}, opts...)
	return r, err
}

func ExecVoidTx(ctx context.Context, repo *EventRepository, readonly bool, f func() error) error {
	var opts []datastore.TransactionOption
	if readonly {
		opts = []datastore.TransactionOption{datastore.ReadOnly}
	}
	_, err := repo.dsClient.RunInTransaction(ctx, func(tx *datastore.Transaction) error {
		e := f()
		if e != nil {
			log.Error().Err(e)
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("tx err: %v, rb err: %v", e, rbErr)
			}
		}
		return nil
	}, opts...)
	return err
}
