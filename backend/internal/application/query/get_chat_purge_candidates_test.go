package query

import (
	"context"
	"testing"

	"gravel_bot/internal/domain/entity"
)

func TestGetChatPurgeCandidatesSelection(t *testing.T) {
	members := []*entity.ChatMember{
		{TelegramUserID: 1, FirstName: "Anna", Username: "anna"},                // проехал, без приза
		{TelegramUserID: 2, FirstName: "Boris", Username: "boris"},              // владелец приза → protected
		{TelegramUserID: 3, FirstName: "Cesar", Username: "cesar"},              // участник без приза
		{TelegramUserID: 4, FirstName: "Dana", Username: "dana"},                // не участник
		{TelegramUserID: 5, FirstName: "Admin", Username: "adm", IsAdmin: true}, // админ → исключён
		{TelegramUserID: 6, FirstName: "Bot", Username: "bot", IsBot: true},     // бот → исключён
	}
	gifts := []*entity.Gift{{ID: 1, UserID: 2, EventID: 77}}
	participants := []*entity.Participant{
		{UserID: 1, Result: &entity.Result{ID: 1}}, // финишировал
		{UserID: 3}, // зарегистрирован, не финишировал
	}

	h := NewGetChatPurgeCandidatesHandler(
		&purgeChatMemberRepoFake{members: members},
		&purgeGiftRepoFake{gifts: gifts},
		&purgeParticipantRepoFake{participants: participants},
	)

	result, err := h.Handle(context.Background(), 77)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if result.ProtectedGiftOwners != 1 {
		t.Fatalf("protected gift owners = %d, want 1", result.ProtectedGiftOwners)
	}
	if len(result.Candidates) != 3 {
		t.Fatalf("candidates = %d, want 3: %+v", len(result.Candidates), result.Candidates)
	}

	byID := map[int64]ChatPurgeCandidate{}
	for _, c := range result.Candidates {
		byID[c.UserID] = c
	}
	if _, ok := byID[2]; ok {
		t.Fatal("gift owner must not be a candidate")
	}
	if _, ok := byID[5]; ok {
		t.Fatal("admin must not be a candidate")
	}
	if _, ok := byID[6]; ok {
		t.Fatal("bot must not be a candidate")
	}
	if byID[1].Reason != "проехал, приза нет" {
		t.Fatalf("finisher reason = %q", byID[1].Reason)
	}
	if byID[3].Reason != "участник без приза" {
		t.Fatalf("registered reason = %q", byID[3].Reason)
	}
	if byID[4].Reason != "не участник, приза нет" {
		t.Fatalf("non-participant reason = %q", byID[4].Reason)
	}
	if byID[1].Label != "Anna (@anna)" {
		t.Fatalf("label = %q", byID[1].Label)
	}
}

// --- Fakes ---

type purgeChatMemberRepoFake struct {
	members []*entity.ChatMember
	deleted []int64
}

func (r *purgeChatMemberRepoFake) Upsert(ctx context.Context, m *entity.ChatMember) error { return nil }
func (r *purgeChatMemberRepoFake) BulkUpsert(ctx context.Context, m []*entity.ChatMember) error {
	return nil
}
func (r *purgeChatMemberRepoFake) Delete(ctx context.Context, id int64) error {
	r.deleted = append(r.deleted, id)
	return nil
}
func (r *purgeChatMemberRepoFake) GetAll(ctx context.Context) ([]*entity.ChatMember, error) {
	return r.members, nil
}
func (r *purgeChatMemberRepoFake) Count(ctx context.Context) (int, error) {
	return len(r.members), nil
}

type purgeGiftRepoFake struct {
	gifts []*entity.Gift
}

func (r *purgeGiftRepoFake) Create(ctx context.Context, g *entity.Gift) error { return nil }
func (r *purgeGiftRepoFake) CreateWithAttachments(ctx context.Context, g *entity.Gift, a []*entity.GiftAttachment) error {
	return nil
}
func (r *purgeGiftRepoFake) Update(ctx context.Context, g *entity.Gift) error { return nil }
func (r *purgeGiftRepoFake) UpdateWithCriteria(ctx context.Context, g *entity.Gift, c []uint) error {
	return nil
}
func (r *purgeGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	return nil, nil
}
func (r *purgeGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return r.gifts, nil
}
func (r *purgeGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, s entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *purgeGiftRepoFake) ListByEventPaged(ctx context.Context, eventID uint, s *entity.GiftReviewStatus, limit, offset int) ([]*entity.Gift, int, error) {
	return nil, 0, nil
}
func (r *purgeGiftRepoFake) CountsByReviewStatus(ctx context.Context, eventID uint) (map[string]int, error) {
	return nil, nil
}
func (r *purgeGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *purgeGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *purgeGiftRepoFake) AddAttachment(ctx context.Context, a *entity.GiftAttachment) error {
	return nil
}
func (r *purgeGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	return nil, nil
}

type purgeParticipantRepoFake struct {
	participants []*entity.Participant
}

func (r *purgeParticipantRepoFake) Create(ctx context.Context, p *entity.Participant) error {
	return nil
}
func (r *purgeParticipantRepoFake) Update(ctx context.Context, p *entity.Participant) error {
	return nil
}
func (r *purgeParticipantRepoFake) FindByID(ctx context.Context, id uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *purgeParticipantRepoFake) FindByUserAndEvent(ctx context.Context, userID int64, eventID uint) (*entity.Participant, error) {
	return nil, nil
}
func (r *purgeParticipantRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return r.participants, nil
}
func (r *purgeParticipantRepoFake) UpdateNotes(ctx context.Context, id uint, notes string) error {
	return nil
}
func (r *purgeParticipantRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *purgeParticipantRepoFake) DeleteWithResultCriteria(ctx context.Context, id uint) error {
	return nil
}
func (r *purgeParticipantRepoFake) GetFinishedByEvent(ctx context.Context, eventID uint) ([]*entity.Participant, error) {
	return nil, nil
}
