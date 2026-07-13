package command

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"

	"gravel_bot/internal/application/query"
)

// ErrManualGiftNoUnawardedParticipants means every participant already has a prize.
var ErrManualGiftNoUnawardedParticipants = errors.New("no participants without prizes")

// randomManualGiftRecipientIDsReader returns recipient IDs that are eligible
// for a manual gift and have no automatic or manual prize.
type randomManualGiftRecipientIDsReader interface {
	Handle(ctx context.Context, query query.GetEligibleUnawardedParticipantIDsQuery) ([]uint, error)
}

// AssignRandomManualGiftRecipientCommand selects an unawarded participant for a
// manual gift owned by the verified Telegram user.
type AssignRandomManualGiftRecipientCommand struct {
	GiftID  uint
	EventID uint
	Actor   ManualGiftRecipientActor
}

// AssignRandomManualGiftRecipientHandler assigns a manual gift to a randomly
// selected participant who currently has no automatic or manual prize.
type AssignRandomManualGiftRecipientHandler struct {
	participantIDsReader randomManualGiftRecipientIDsReader
	setRecipientHandler  *SetManualGiftRecipientHandler
	randomIndex          func(max int) (int, error)
}

// NewAssignRandomManualGiftRecipientHandler creates a handler that uses
// crypto/rand so the recipient cannot be predicted by a client.
func NewAssignRandomManualGiftRecipientHandler(
	participantIDsReader randomManualGiftRecipientIDsReader,
	setRecipientHandler *SetManualGiftRecipientHandler,
) *AssignRandomManualGiftRecipientHandler {
	return newAssignRandomManualGiftRecipientHandler(
		participantIDsReader,
		setRecipientHandler,
		cryptoRandomIndex,
	)
}

func newAssignRandomManualGiftRecipientHandler(
	participantIDsReader randomManualGiftRecipientIDsReader,
	setRecipientHandler *SetManualGiftRecipientHandler,
	randomIndex func(max int) (int, error),
) *AssignRandomManualGiftRecipientHandler {
	return &AssignRandomManualGiftRecipientHandler{
		participantIDsReader: participantIDsReader,
		setRecipientHandler:  setRecipientHandler,
		randomIndex:          randomIndex,
	}
}

// Handle chooses an unawarded participant and delegates the protected write to
// SetManualGiftRecipientHandler, which validates the gift owner and event.
func (h *AssignRandomManualGiftRecipientHandler) Handle(
	ctx context.Context,
	cmd AssignRandomManualGiftRecipientCommand,
) (uint, error) {
	if _, err := h.setRecipientHandler.manualGiftForCommand(ctx, SetManualGiftRecipientCommand{
		GiftID:  cmd.GiftID,
		EventID: cmd.EventID,
		Actor:   cmd.Actor,
	}); err != nil {
		return 0, err
	}

	candidateIDs, err := h.participantIDsReader.Handle(ctx, query.GetEligibleUnawardedParticipantIDsQuery{EventID: cmd.EventID})
	if err != nil {
		return 0, fmt.Errorf("find random manual gift recipient candidates: %w", err)
	}
	if len(candidateIDs) == 0 {
		log.Printf("WARN random manual gift recipient unavailable: gift_id=%d event_id=%d reason=no_unawarded_participants", cmd.GiftID, cmd.EventID)
		return 0, ErrManualGiftNoUnawardedParticipants
	}

	index, err := h.randomIndex(len(candidateIDs))
	if err != nil {
		return 0, fmt.Errorf("choose random manual gift recipient: %w", err)
	}
	if index < 0 || index >= len(candidateIDs) {
		return 0, fmt.Errorf("choose random manual gift recipient: invalid random index %d", index)
	}

	recipientID := candidateIDs[index]
	if err := h.setRecipientHandler.Handle(ctx, SetManualGiftRecipientCommand{
		GiftID:                 cmd.GiftID,
		EventID:                cmd.EventID,
		Actor:                  cmd.Actor,
		RecipientParticipantID: &recipientID,
	}); err != nil {
		return 0, err
	}

	log.Printf("INFO [FIX:manual-recipient-eligibility] random manual gift recipient assigned: gift_id=%d event_id=%d candidate_count=%d recipient_participant_id=%d", cmd.GiftID, cmd.EventID, len(candidateIDs), recipientID)
	return recipientID, nil
}

func cryptoRandomIndex(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("random range must be positive: %d", max)
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, fmt.Errorf("read secure random value: %w", err)
	}
	return int(value.Int64()), nil
}
