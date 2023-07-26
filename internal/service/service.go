package service

import (
	"context"
	"event-gorganizer/internal/model"
	"event-gorganizer/internal/repository"
	"github.com/rs/zerolog/log"
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

func (s *EventService) CreateNewEvent(ctx context.Context, chatId int64, creator *model.Participant, title string) *model.Event {
	r, err := repository.ExecTx(ctx, s.repo, false,
		func() (*model.Event, error) {
			prevEvent := s.repo.GetActiveEvent(ctx, chatId)
			prevEvent.Active = false
			s.repo.Save(ctx, prevEvent)
			newEvent := &model.Event{
				ChatId:       chatId,
				Creator:      creator,
				Title:        title,
				Created:      time.Now(),
				Participants: make([]*model.Participant, 0),
				Active:       true,
			}
			s.repo.Save(ctx, newEvent)
			return newEvent, nil
		})
	if err != nil {
		log.Error().Err(err)
	}
	return r
}

func (s *EventService) GetActiveEvent(ctx context.Context, chatId int64) *model.Event {
	res, err := repository.ExecTx(ctx, s.repo, true,
		func() (*model.Event, error) {
			return s.repo.GetActiveEvent(ctx, chatId), nil
		})
	if err != nil {
		log.Error().Err(err)
	}
	return res
}

func (s *EventService) AddNewParticipant(ctx context.Context, chatId int64, participant *model.Participant) *model.Event {
	res, err := repository.ExecTx(ctx, s.repo, false,
		func() (*model.Event, error) {
			event := s.GetActiveEvent(ctx, chatId)
			if event.AddParticipant(participant) != nil {
				s.repo.Save(ctx, event)
			}
			return event, nil
		})
	if err != nil {
		log.Error().Err(err)
	}
	return res
}

func (s *EventService) RemoveParticipant(ctx context.Context, chatId int64, participant *model.Participant) *model.Participant {
	res, err := repository.ExecTx(ctx, s.repo, false,
		func() (*model.Participant, error) {
			event := s.GetActiveEvent(ctx, chatId)
			removed := event.RemoveParticipant(participant.Id())
			if removed != nil {
				s.repo.Save(ctx, event)
			}
			return removed, nil
		})
	if err != nil {
		log.Error().Err(err)
	}
	return res
}

func (s *EventService) RemoveParticipantByNumber(ctx context.Context, chatId int64, idx int) *model.Participant {
	res, err := repository.ExecTx(ctx, s.repo, false,
		func() (*model.Participant, error) {
			event := s.GetActiveEvent(ctx, chatId)
			removed := event.RemoveParticipantByNumber(idx)
			if removed != nil {
				s.repo.Save(ctx, event)
			}
			return removed, nil
		})
	if err != nil {
		log.Error().Err(err)
	}
	return res
}
