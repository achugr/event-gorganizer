package service

import (
	"context"
	"event-gorganizer/internal/model"
	"event-gorganizer/internal/repository"
	"time"
)

type EventService struct {
	repo *repository.EventRepository
}

func NewService(repo *repository.EventRepository) *EventService {
	return &EventService{
		repo: repo,
	}
}

func (s *EventService) CreateNewEvent(ctx context.Context, chatId int64, creator *model.Participant, title string) (*model.Event, error) {
	tx, err := repository.ExecTx(ctx, s.repo, false,
		func() (*model.Event, error) {
			prevEvent, err := s.repo.GetActiveEvent(ctx, chatId)
			if err != nil {
				return nil, err
			}
			prevEvent.Active = false
			_, err = s.repo.Save(ctx, prevEvent)
			if err != nil {
				return nil, err
			}
			newEvent := &model.Event{
				ChatId:       chatId,
				Creator:      creator,
				Title:        title,
				Created:      time.Now(),
				Participants: make([]*model.Participant, 0),
				Active:       true,
			}
			newEvent, err = s.repo.Save(ctx, newEvent)
			if err != nil {
				return nil, err
			}
			return newEvent, nil
		})
	return tx, err
}

func (s *EventService) GetActiveEvent(ctx context.Context, chatId int64) (*model.Event, error) {
	tx, err := repository.ExecTx(ctx, s.repo, true,
		func() (*model.Event, error) {
			event, err := s.repo.GetActiveEvent(ctx, chatId)
			if err != nil {
				return nil, err
			}
			return event, nil
		})
	return tx, err
}

func (s *EventService) AddNewParticipant(ctx context.Context, chatId int64, participant *model.Participant) (*model.Participant, error) {
	return repository.ExecTx(ctx, s.repo, false,
		func() (*model.Participant, error) {
			event, err := s.GetActiveEvent(ctx, chatId)
			if err != nil {
				return nil, err
			}
			if event.AddParticipant(participant) {
				_, err = s.repo.Save(ctx, event)
				if err != nil {
					return nil, err
				}
			}
			return participant, nil
		})
}

func (s *EventService) RemoveParticipant(ctx context.Context, chatId int64, participant *model.Participant) (*model.Participant, error) {
	return repository.ExecTx(ctx, s.repo, false,
		func() (*model.Participant, error) {
			event, err := s.GetActiveEvent(ctx, chatId)
			if err != nil {
				return nil, err
			}
			removed := event.RemoveParticipant(participant.Id())
			if removed != nil {
				event, err = s.repo.Save(ctx, event)
				if err != nil {
					return nil, err
				}
			}
			return removed, nil
		})
}

func (s *EventService) RemoveParticipantByNumber(ctx context.Context, chatId int64, idx int) (*model.Participant, error) {
	return repository.ExecTx(ctx, s.repo, false,
		func() (*model.Participant, error) {
			event, err := s.GetActiveEvent(ctx, chatId)
			if err != nil {
				return nil, err
			}
			removed := event.RemoveParticipantByNumber(idx)
			if removed != nil {
				_, err := s.repo.Save(ctx, event)
				if err != nil {
					return nil, err
				}
			}
			return removed, nil
		})
}

func (s *EventService) MarkPaid(ctx context.Context, chatId int64, participant *model.Participant) error {
	return repository.ExecVoidTx(ctx, s.repo, false,
		func() error {
			event, err := s.GetActiveEvent(ctx, chatId)
			if err != nil {
				return err
			}
			event.MarkPaid(participant.Id())
			event, err = s.repo.Save(ctx, event)
			return nil
		})
}

func (s *EventService) MarkPaidByNumber(ctx context.Context, chatId int64, idx int) error {
	return repository.ExecVoidTx(ctx, s.repo, false,
		func() error {
			event, err := s.GetActiveEvent(ctx, chatId)
			if err != nil {
				return err
			}
			event.MarkPaidByNumber(idx)
			event, err = s.repo.Save(ctx, event)
			return nil
		})
}
