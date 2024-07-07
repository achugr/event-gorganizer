package tgbot

import (
	"event-gorganizer/internal/model"
	"fmt"
	"time"
)

type Event struct {
	Id           string
	ChatId       int64
	Creator      Participant
	Title        string
	Participants []Participant
	Created      time.Time
	Active       bool
}

type Participant struct {
	Number        int
	Name          string
	Title         string
	PaymentStatus PaymentStatus
}

type PaymentStatus struct {
	Paid bool
}

func NewEventView(e *model.Event) Event {

	var participants []Participant
	for _, p := range e.Participants {
		participants = append(participants, NewParticipantView(p))
	}

	return Event{
		Id:     e.Id(),
		ChatId: e.ChatId,
		Creator: Participant{
			Number:        e.Creator.Number,
			Name:          e.Creator.Name,
			Title:         getTitle(*e.Creator),
			PaymentStatus: PaymentStatus{Paid: e.Creator.PaymentStatus.Paid},
		},
		Title:        e.Title,
		Participants: participants,
		Created:      e.Created,
		Active:       e.Active,
	}
}

func NewParticipantView(p *model.Participant) Participant {
	return Participant{
		Number:        p.Number,
		Name:          p.Name,
		Title:         getTitle(*p),
		PaymentStatus: PaymentStatus{Paid: p.PaymentStatus.Paid},
	}
}

func getTitle(p model.Participant) string {
	var title string

	if p.InvitedBy != nil {
		title = fmt.Sprintf("#%d: %s (invited by @%s)", p.Number, p.Name, p.InvitedBy.Name)
	} else {
		title = fmt.Sprintf("#%d: %s", p.Number, p.Name)
	}

	if p.PaymentStatus.Paid {
		title += " 💰✅"
	}

	return title
}
