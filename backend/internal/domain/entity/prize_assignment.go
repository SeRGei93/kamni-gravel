package entity

import "time"

// PrizeAssignment представляет назначение приза участнику
type PrizeAssignment struct {
	ID            uint
	ParticipantID uint
	GiftID        uint
	Comment       string
	AssignedAt    time.Time
	
	// Связанные сущности
	Participant *Participant
	Gift        *Gift
}
