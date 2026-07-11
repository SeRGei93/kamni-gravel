package dto

import (
	"encoding/json"
	"testing"

	"gravel_bot/internal/domain/entity"
)

func TestFromGiftExposesManualModeWithoutRecipientIdentity(t *testing.T) {
	recipientID := uint(30)
	publicGift := FromGift(&entity.Gift{
		ID:                           1,
		ManualDistribution:           true,
		ManualRecipientParticipantID: &recipientID,
		ManualRecipient:              &entity.Participant{ID: recipientID, UserID: 900},
	})
	if !publicGift.ManualDistribution {
		t.Fatal("public gift should expose manual_distribution")
	}
	body, err := json.Marshal(publicGift)
	if err != nil {
		t.Fatalf("marshal public gift: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal public gift: %v", err)
	}
	if _, exists := decoded["manual_recipient"]; exists {
		t.Fatalf("public gift response leaks manual recipient: %s", body)
	}
	if _, exists := decoded["recipient"]; exists {
		t.Fatalf("public gift response leaks recipient: %s", body)
	}
}
