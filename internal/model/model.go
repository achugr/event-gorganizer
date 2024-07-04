package model

import (
	"fmt"
	"strconv"
	"time"
)

type Event struct {
	ChatId       int64
	Creator      *Participant
	Title        string
	Participants []*Participant `datastore:",noindex"`
	Created      time.Time
	Active       bool
}

type Participant struct {
	Number        int
	Name          string
	TelegramId    *int64
	InvitedBy     *Participant
	PaymentStatus PaymentStatus
}

type PaymentStatus struct {
	Paid bool
}

func (p Participant) Id() string {
	if p.TelegramId != nil {
		return strconv.FormatInt(*p.TelegramId, 10)
	} else {
		return p.Name
	}
}

func (e *Event) Id() string {
	return fmt.Sprintf("%d-%d", e.ChatId, e.Created.Unix())
}

func (e *Event) FindParticipant(id string) *Participant {
	for _, p := range e.Participants {
		if p.Id() == id {
			return p
		}
	}
	return nil
}

func (e *Event) AddParticipant(participant *Participant) bool {
	existing := e.FindParticipant(participant.Id())
	if existing == nil {
		if len(e.Participants) > 0 {
			participant.Number = (e.Participants)[len(e.Participants)-1].Number + 1
		} else {
			participant.Number = 1
		}
		e.Participants = append(e.Participants, participant)
		return true
	}
	return false
}

func (e *Event) RemoveParticipant(id string) *Participant {
	for idx, p := range e.Participants {
		if p.Id() == id {
			return e.removeParticipantByIndex(idx)
		}
	}
	return nil
}

func (e *Event) RemoveParticipantByNumber(number int) *Participant {
	for idx, p := range e.Participants {
		if p.Number == number {
			return e.removeParticipantByIndex(idx)
		}
	}
	return nil
}

func (e *Event) MarkPaid(id string) {
	for _, p := range e.Participants {
		if p.Id() == id {
			p.PaymentStatus = PaymentStatus{Paid: true}
		}
	}
}

func (e *Event) MarkPaidByNumber(number int) {
	for _, p := range e.Participants {
		if p.Number == number {
			p.PaymentStatus = PaymentStatus{Paid: true}
		}
	}
}

func (e *Event) removeParticipantByIndex(idx int) *Participant {
	if idx < len(e.Participants) {
		removed := e.Participants[idx]
		e.Participants = append(e.Participants[:idx], e.Participants[idx+1:]...)
		return removed
	}
	return nil
}
