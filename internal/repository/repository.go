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

func NewEventRepository(ctx context.Context, settings GcpSettings) *EventRepository {
	opts := make([]option.ClientOption, 0)
	if settings.CredentialsFilePath != nil {
		opts = append(opts, option.WithCredentialsFile(*settings.CredentialsFilePath))
	}
	dsClient, err := datastore.NewClient(ctx, settings.ProjectName, opts...)
	if err != nil {
		log.Fatal().Msgf("Datastore client initialization failed: %s.", err)
	}

	return &EventRepository{
		dsClient: dsClient,
	}
}

func (r EventRepository) Save(ctx context.Context, event *model.Event) *model.Event {
	key := datastore.NameKey("Event", event.Id(), nil)
	_, err := r.dsClient.Put(ctx, key, event)
	if err != nil {
		log.Error().Msgf("Failed saving the event %s: %s", event.Id(), err)
		return nil
	}
	return event
}

func (r EventRepository) GetActiveEvent(ctx context.Context, chatId int64) *model.Event {
	query := datastore.NewQuery("Event").
		FilterField("ChatId", "=", chatId).
		FilterField("Active", "=", true).
		Limit(1)

	iter := r.dsClient.Run(ctx, query)
	var event model.Event
	_, err := iter.Next(&event)
	if err != nil && err != iterator.Done {
		log.Error().Msgf("Failed getting event for the chat %s.", chatId)
		return nil
	}
	return &event
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
		_, ce := tx.Commit()
		if ce != nil {
			log.Error().Err(e)
			return ce
		}
		return nil
	}, opts...)
	return r, err
}
