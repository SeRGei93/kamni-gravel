package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
)

func noSleep(context.Context, time.Duration) bool { return true }

func TestExecuteChatPurgeProtectsGiftOwners(t *testing.T) {
	kicker := &purgeKickerFake{}
	chatRepo := &purgeCmdChatMemberRepoFake{}
	h := NewExecuteChatPurgeHandler(chatRepo, &purgeCmdGiftRepoFake{
		gifts: []*entity.Gift{{ID: 1, UserID: 2, EventID: 77}}, // 2 — владелец приза
	}, kicker)
	h.sleep = noSleep

	result, err := h.Handle(context.Background(), ExecuteChatPurgeCommand{
		EventID: 77,
		UserIDs: []int64{1, 2, 3}, // 2 защищён, хоть и прислан
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if result.Kicked != 2 || result.Protected != 1 {
		t.Fatalf("kicked=%d protected=%d, want kicked=2 protected=1", result.Kicked, result.Protected)
	}
	for _, id := range kicker.kicked {
		if id == 2 {
			t.Fatal("gift owner must never be kicked")
		}
	}
	if len(chatRepo.deleted) != 2 {
		t.Fatalf("roster deletions = %d, want 2", len(chatRepo.deleted))
	}
}

func TestExecuteChatPurgeSkipsNotInChat(t *testing.T) {
	kicker := &purgeKickerFake{notInChat: map[int64]bool{5: true}}
	h := NewExecuteChatPurgeHandler(&purgeCmdChatMemberRepoFake{}, &purgeCmdGiftRepoFake{}, kicker)
	h.sleep = noSleep

	result, err := h.Handle(context.Background(), ExecuteChatPurgeCommand{EventID: 1, UserIDs: []int64{5, 6}})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if result.Skipped != 1 || result.Kicked != 1 || result.Failed != 0 {
		t.Fatalf("skipped=%d kicked=%d failed=%d, want skipped=1 kicked=1 failed=0", result.Skipped, result.Kicked, result.Failed)
	}
}

func TestExecuteChatPurgeCountsFailures(t *testing.T) {
	kicker := &purgeKickerFake{failing: map[int64]bool{7: true}}
	h := NewExecuteChatPurgeHandler(&purgeCmdChatMemberRepoFake{}, &purgeCmdGiftRepoFake{}, kicker)
	h.sleep = noSleep

	result, err := h.Handle(context.Background(), ExecuteChatPurgeCommand{EventID: 1, UserIDs: []int64{7}})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if result.Failed != 1 || result.Kicked != 0 {
		t.Fatalf("failed=%d kicked=%d, want failed=1 kicked=0", result.Failed, result.Kicked)
	}
}

func TestExecuteChatPurgeNotConfigured(t *testing.T) {
	h := NewExecuteChatPurgeHandler(&purgeCmdChatMemberRepoFake{}, &purgeCmdGiftRepoFake{}, nil)
	_, err := h.Handle(context.Background(), ExecuteChatPurgeCommand{EventID: 1, UserIDs: []int64{1}})
	if !errors.Is(err, ErrChatPurgeNotConfigured) {
		t.Fatalf("err = %v, want ErrChatPurgeNotConfigured", err)
	}
}

// --- Fakes ---

type purgeKickerFake struct {
	kicked    []int64
	notInChat map[int64]bool
	failing   map[int64]bool
}

func (k *purgeKickerFake) Kick(ctx context.Context, userID int64) error {
	if k.notInChat[userID] {
		return ErrMemberNotInChat
	}
	if k.failing[userID] {
		return errors.New("boom")
	}
	k.kicked = append(k.kicked, userID)
	return nil
}

type purgeCmdChatMemberRepoFake struct {
	deleted []int64
}

func (r *purgeCmdChatMemberRepoFake) Upsert(ctx context.Context, m *entity.ChatMember) error {
	return nil
}
func (r *purgeCmdChatMemberRepoFake) BulkUpsert(ctx context.Context, m []*entity.ChatMember) error {
	return nil
}
func (r *purgeCmdChatMemberRepoFake) Delete(ctx context.Context, id int64) error {
	r.deleted = append(r.deleted, id)
	return nil
}
func (r *purgeCmdChatMemberRepoFake) GetAll(ctx context.Context) ([]*entity.ChatMember, error) {
	return nil, nil
}
func (r *purgeCmdChatMemberRepoFake) Count(ctx context.Context) (int, error) { return 0, nil }

type purgeCmdGiftRepoFake struct {
	gifts []*entity.Gift
}

func (r *purgeCmdGiftRepoFake) Create(ctx context.Context, g *entity.Gift) error { return nil }
func (r *purgeCmdGiftRepoFake) CreateWithAttachments(ctx context.Context, g *entity.Gift, a []*entity.GiftAttachment) error {
	return nil
}
func (r *purgeCmdGiftRepoFake) Update(ctx context.Context, g *entity.Gift) error { return nil }
func (r *purgeCmdGiftRepoFake) UpdateWithCriteria(ctx context.Context, g *entity.Gift, c []uint) error {
	return nil
}
func (r *purgeCmdGiftRepoFake) FindByID(ctx context.Context, id uint) (*entity.Gift, error) {
	return nil, nil
}
func (r *purgeCmdGiftRepoFake) FindByEvent(ctx context.Context, eventID uint) ([]*entity.Gift, error) {
	return r.gifts, nil
}
func (r *purgeCmdGiftRepoFake) FindByEventAndReviewStatus(ctx context.Context, eventID uint, s entity.GiftReviewStatus) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *purgeCmdGiftRepoFake) ListByEventPaged(ctx context.Context, eventID uint, s *entity.GiftReviewStatus, limit, offset int) ([]*entity.Gift, int, error) {
	return nil, 0, nil
}
func (r *purgeCmdGiftRepoFake) CountsByReviewStatus(ctx context.Context, eventID uint) (map[string]int, error) {
	return nil, nil
}
func (r *purgeCmdGiftRepoFake) FindByUser(ctx context.Context, userID int64) ([]*entity.Gift, error) {
	return nil, nil
}
func (r *purgeCmdGiftRepoFake) Delete(ctx context.Context, id uint) error { return nil }
func (r *purgeCmdGiftRepoFake) AddAttachment(ctx context.Context, a *entity.GiftAttachment) error {
	return nil
}
func (r *purgeCmdGiftRepoFake) GetAttachments(ctx context.Context, giftID uint) ([]*entity.GiftAttachment, error) {
	return nil, nil
}
