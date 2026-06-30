package valueobject

import "testing"

func TestNewParticipantStatus(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    ParticipantStatus
		wantErr bool
	}{
		{name: "empty defaults to active", value: "", want: ParticipantStatusActive},
		{name: "active", value: "active", want: ParticipantStatusActive},
		{name: "dnf", value: "dnf", want: ParticipantStatusDNF},
		{name: "disqualified", value: "disqualified", want: ParticipantStatusDisqualified},
		{name: "unknown", value: "retired", wantErr: true},
		{name: "wrong case", value: "DNF", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewParticipantStatus(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q, got status %q", tt.value, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("NewParticipantStatus(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestParticipantStatusDisplayName(t *testing.T) {
	tests := map[ParticipantStatus]string{
		ParticipantStatusActive:       "Участвует",
		ParticipantStatusDNF:          "Сошёл с дистанции",
		ParticipantStatusDisqualified: "Дисквалификация",
	}
	for status, want := range tests {
		if got := status.DisplayName(); got != want {
			t.Errorf("%q.DisplayName() = %q, want %q", status, got, want)
		}
	}
}
