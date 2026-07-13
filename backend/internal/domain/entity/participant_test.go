package entity

import (
	"testing"

	"gravel_bot/internal/domain/valueobject"
)

func TestParticipantIsEligibleForManualGift(t *testing.T) {
	tests := []struct {
		name        string
		participant *Participant
		want        bool
	}{
		{
			name:        "finished active participant",
			participant: &Participant{Status: valueobject.ParticipantStatusActive, Result: &Result{}},
			want:        true,
		},
		{
			name:        "dnf participant without submitted result",
			participant: &Participant{Status: valueobject.ParticipantStatusDNF},
			want:        true,
		},
		{
			name:        "dns participant without result",
			participant: &Participant{Status: valueobject.ParticipantStatusActive},
			want:        false,
		},
		{
			name:        "disqualified participant with result",
			participant: &Participant{Status: valueobject.ParticipantStatusDisqualified, Result: &Result{}},
			want:        false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.participant.IsEligibleForManualGift(); got != testCase.want {
				t.Fatalf("IsEligibleForManualGift() = %t, want %t", got, testCase.want)
			}
		})
	}
}
