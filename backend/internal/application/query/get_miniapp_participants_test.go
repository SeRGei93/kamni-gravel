package query

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
	"gravel_bot/internal/domain/valueobject"
)

func TestGetMiniappParticipantsHandlerReturnsMinimalDeterministicallySortedOptions(t *testing.T) {
	repo := &miniappParticipantsRepoFake{participants: []*entity.Participant{
		{ID: 3, UserID: 300, EventID: 77, Status: valueobject.ParticipantStatusDisqualified, Notes: "private", User: &entity.User{FirstName: "Zoe", Username: "zoe"}},
		{ID: 2, UserID: 200, EventID: 77, Status: valueobject.ParticipantStatusDNF, Notes: "private", User: &entity.User{FirstName: "Alex", LastName: "Rider", Username: "alex"}},
		{ID: 1, UserID: 100, EventID: 77, Status: valueobject.ParticipantStatusActive, Notes: "private", User: &entity.User{FirstName: "Alex", LastName: "Rider", Username: "alex2"}},
		{ID: 4, UserID: 400, EventID: 77, User: &entity.User{}},
	}}
	handler := NewGetMiniappParticipantsHandler(
		repo,
		&miniappManualRecipientCountRepoFake{counts: map[uint]int{3: 1}},
		&miniappPrizeDistributionReaderFake{results: []*PrizeDistributionResult{{
			ParticipantID: 1,
			MatchedGifts:  []*entity.Gift{{ID: 1}},
		}}},
	)

	options, err := handler.Handle(context.Background(), GetMiniappParticipantsQuery{EventID: 77})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if repo.eventID != 77 || len(options) != 4 {
		t.Fatalf("options = %+v, event_id=%d", options, repo.eventID)
	}
	if options[0].ID != 2 || options[1].ID != 4 || options[2].ID != 1 || options[3].ID != 3 {
		t.Fatalf("deterministic order = [%d %d %d %d]", options[0].ID, options[1].ID, options[2].ID, options[3].ID)
	}
	if options[0].Status != "dnf" || options[1].Status != "active" || options[2].Status != "active" || options[3].Status != "disqualified" {
		t.Fatalf("participant statuses = %+v", options)
	}
	if options[0].HasPrize || options[1].HasPrize || !options[2].HasPrize || !options[3].HasPrize {
		t.Fatalf("participant prize flags = %+v", options)
	}

	body, err := json.Marshal(options)
	if err != nil {
		t.Fatalf("marshal participant options: %v", err)
	}
	for _, prohibitedKey := range []string{"user_id", "notes", "registered_at", "result_link", "elapsed_time_sec"} {
		if containsJSONKey(body, prohibitedKey) {
			t.Fatalf("Mini App participant options leak %q: %s", prohibitedKey, body)
		}
	}
}

func TestGetMiniappParticipantsHandlerWrapsRepositoryFailure(t *testing.T) {
	repoErr := errors.New("database unavailable")
	handler := NewGetMiniappParticipantsHandler(
		&miniappParticipantsRepoFake{err: repoErr},
		&miniappManualRecipientCountRepoFake{},
		&miniappPrizeDistributionReaderFake{},
	)
	if _, err := handler.Handle(context.Background(), GetMiniappParticipantsQuery{EventID: 77}); !errors.Is(err, repoErr) {
		t.Fatalf("Handle error = %v, want wrapped repository error", err)
	}
}

type miniappParticipantsRepoFake struct {
	repository.ParticipantRepository
	eventID      uint
	participants []*entity.Participant
	err          error
}

func (r *miniappParticipantsRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	r.eventID = eventID
	return r.participants, r.err
}

type miniappManualRecipientCountRepoFake struct {
	counts map[uint]int
	err    error
}

func (r *miniappManualRecipientCountRepoFake) ManualRecipientCountsByEvent(context.Context, uint) (map[uint]int, error) {
	return r.counts, r.err
}

type miniappPrizeDistributionReaderFake struct {
	results []*PrizeDistributionResult
	err     error
}

func (r *miniappPrizeDistributionReaderFake) Handle(context.Context, GetPrizeDistributionQuery) ([]*PrizeDistributionResult, error) {
	return r.results, r.err
}
