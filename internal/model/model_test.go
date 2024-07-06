package model

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func TestEvent_AddParticipant(t *testing.T) {
	event := Event{
		ChatId:       1,
		Creator:      &Participant{Name: "Player 0", TelegramId: getIntPointer(0)},
		Title:        "Football",
		Participants: make([]*Participant, 0),
		Created:      time.Now(),
		Active:       true,
	}

	participant := &Participant{
		Name:       "Player 1",
		TelegramId: getIntPointer(1),
		InvitedBy:  nil,
	}
	added := event.AddParticipant(participant)
	assert.NotNil(t, added, "Operation result incorrect")
	assert.ElementsMatch(t, event.Participants, []*Participant{participant}, "Participant was not added")
}

func TestEvent_AddSameParticipantTwice(t *testing.T) {
	event := Event{
		ChatId:       1,
		Creator:      &Participant{Name: "Player 0", TelegramId: getIntPointer(0)},
		Title:        "Football",
		Participants: make([]*Participant, 0),
		Created:      time.Now(),
		Active:       true,
	}

	newParticipant := &Participant{
		Name:       "Player 1",
		TelegramId: getIntPointer(1),
		InvitedBy:  nil,
	}
	added := event.AddParticipant(newParticipant)
	assert.True(t, added, "Operation result incorrect")
	assert.False(t, event.AddParticipant(newParticipant), "Operation result incorrect")
	assert.ElementsMatch(t, event.Participants, []*Participant{newParticipant}, "Participant was added twice")
}

func TestEvent_AddDifferentParticipants(t *testing.T) {
	event := Event{
		ChatId:       1,
		Creator:      &Participant{Name: "Player 0", TelegramId: getIntPointer(0)},
		Title:        "Football",
		Participants: make([]*Participant, 0),
		Created:      time.Now(),
		Active:       true,
	}

	p1 := &Participant{
		Name:       "Player 1",
		TelegramId: getIntPointer(1),
		InvitedBy:  nil,
	}
	event.AddParticipant(p1)

	p2 := &Participant{
		Name:       "Player 2",
		TelegramId: getIntPointer(2),
		InvitedBy:  nil,
	}
	event.AddParticipant(p2)

	assert.ElementsMatch(t, event.Participants, []*Participant{p1, p2}, "Participants added incorrectly")
}

func TestEvent_RemoveParticipant(t *testing.T) {
	event := Event{
		ChatId:       1,
		Creator:      &Participant{Name: "Player 0", TelegramId: getIntPointer(0)},
		Title:        "Football",
		Participants: make([]*Participant, 0),
		Created:      time.Now(),
		Active:       true,
	}

	participant := &Participant{
		Name:       "Player 1",
		TelegramId: getIntPointer(1),
		InvitedBy:  nil,
	}
	added := event.AddParticipant(participant)
	assert.True(t, added, "Operation result incorrect")
	removedParticipant := event.RemoveParticipant(participant.Id())
	assert.Equal(t, participant, removedParticipant, "Operation result is not correct")
	assert.Emptyf(t, event.Participants, "Participant was not removed")
}

func TestEvent_RemoveParticipantWhenMany(t *testing.T) {
	event := Event{
		ChatId:       1,
		Creator:      &Participant{Name: "Player 0", TelegramId: getIntPointer(0)},
		Title:        "Football",
		Participants: make([]*Participant, 0),
		Created:      time.Now(),
		Active:       true,
	}

	p1 := &Participant{
		Name:       "Player 1",
		TelegramId: getIntPointer(1),
		InvitedBy:  nil,
	}
	event.AddParticipant(p1)

	p2 := &Participant{
		Name:       "Player 2",
		TelegramId: getIntPointer(2),
		InvitedBy:  nil,
	}
	event.AddParticipant(p2)

	p3 := &Participant{
		Name:       "Player 3",
		TelegramId: getIntPointer(3),
		InvitedBy:  nil,
	}
	event.AddParticipant(p3)

	removedParticipant := event.RemoveParticipant(p2.Id())

	assert.Equalf(t, p2, removedParticipant, "Operation result is not correct")
	assert.ElementsMatch(t, event.Participants, []*Participant{p1, p3}, "Participant was not removed")
}

func TestEvent_RemoveParticipantByIndex(t *testing.T) {
	event := Event{
		ChatId:       1,
		Creator:      &Participant{Name: "Player 0", TelegramId: getIntPointer(0)},
		Title:        "Football",
		Participants: make([]*Participant, 0),
		Created:      time.Now(),
		Active:       true,
	}

	p1 := &Participant{
		Name:       "Player 1",
		TelegramId: getIntPointer(1),
		InvitedBy:  nil,
	}
	event.AddParticipant(p1)

	p2 := &Participant{
		Name:       "Player 2",
		TelegramId: getIntPointer(2),
		InvitedBy:  nil,
	}
	event.AddParticipant(p2)

	p3 := &Participant{
		Name:       "Player 3",
		TelegramId: getIntPointer(3),
		InvitedBy:  nil,
	}
	event.AddParticipant(p3)

	removedParticipant := event.RemoveParticipantByNumber(1)

	assert.Equalf(t, p1, removedParticipant, "Operation result is not correct")
	assert.ElementsMatch(t, event.Participants, []*Participant{p2, p3}, "Participant was not removed")
}

func TestEvent_FindParticipant(t *testing.T) {
	event := Event{
		ChatId:       1,
		Creator:      &Participant{Name: "Player 0", TelegramId: getIntPointer(0)},
		Title:        "Football",
		Participants: make([]*Participant, 0),
		Created:      time.Now(),
		Active:       true,
	}

	event.AddParticipant(&Participant{
		Name:       "Player 1",
		TelegramId: getIntPointer(1),
		InvitedBy:  nil,
	})

	p2 := &Participant{
		Name:       "Player 2",
		TelegramId: getIntPointer(2),
		InvitedBy:  nil,
	}
	event.AddParticipant(p2)

	event.AddParticipant(&Participant{
		Name:       "Player 3",
		TelegramId: getIntPointer(3),
		InvitedBy:  nil,
	})

	foundParticipant := event.FindParticipant(strconv.Itoa(2))

	assert.Equalf(t, p2, foundParticipant, "Operation result is not correct")
}

func TestMarkPaid(t *testing.T) {
	e := &Event{
		Participants: []*Participant{
			{
				Number:     1,
				Name:       "Alice",
				TelegramId: getIntPointer(111),
			},
			{
				Number:        2,
				Name:          "Bob",
				TelegramId:    getIntPointer(222),
				PaymentStatus: PaymentStatus{Paid: false},
			},
			{
				Number:        3,
				Name:          "Charlie",
				TelegramId:    getIntPointer(333),
				PaymentStatus: PaymentStatus{Paid: false},
			},
		},
	}

	e.MarkPaid(e.Participants[1].Id())
	assert.Equal(t, PaymentStatus{Paid: true}, e.Participants[1].PaymentStatus)
}

func getIntPointer(id int64) *int64 {
	return &id
}
