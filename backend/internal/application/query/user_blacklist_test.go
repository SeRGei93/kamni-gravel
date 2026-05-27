package query

import (
	"context"
	"errors"
	"testing"

	"gravel_bot/internal/domain/entity"
)

func TestListUserBlacklistHandlerReturnsEntries(t *testing.T) {
	h := NewListUserBlacklistHandler(&queryUserBlacklistRepoFake{
		entries: []*entity.UserBlacklist{{TelegramUserID: 123, Reason: "spam"}},
	})

	entries, err := h.Handle(context.Background())
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if len(entries) != 1 || entries[0].TelegramUserID != 123 {
		t.Fatalf("entries mismatch: %#v", entries)
	}
}

func TestIsUserBlacklistedHandlerValidatesTelegramUserID(t *testing.T) {
	h := NewIsUserBlacklistedHandler(&queryUserBlacklistRepoFake{})

	_, err := h.Handle(context.Background(), IsUserBlacklistedQuery{TelegramUserID: -1})
	if !errors.Is(err, ErrInvalidTelegramUserID) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrInvalidTelegramUserID)
	}
}

func TestIsUserBlacklistedHandlerReturnsState(t *testing.T) {
	repo := &queryUserBlacklistRepoFake{blacklisted: true}
	h := NewIsUserBlacklistedHandler(repo)

	isBlacklisted, err := h.Handle(context.Background(), IsUserBlacklistedQuery{TelegramUserID: 123})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if !isBlacklisted || repo.checkedTelegramUserID != 123 {
		t.Fatalf("blacklist check mismatch: is_blacklisted=%t checked=%d", isBlacklisted, repo.checkedTelegramUserID)
	}
}

type queryUserBlacklistRepoFake struct {
	entries               []*entity.UserBlacklist
	blacklisted           bool
	checkedTelegramUserID int64
}

func (r *queryUserBlacklistRepoFake) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	return r.entries, nil
}
func (r *queryUserBlacklistRepoFake) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *queryUserBlacklistRepoFake) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	r.checkedTelegramUserID = telegramUserID
	return r.blacklisted, nil
}
func (r *queryUserBlacklistRepoFake) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	return nil
}
func (r *queryUserBlacklistRepoFake) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *queryUserBlacklistRepoFake) Delete(ctx context.Context, telegramUserID int64) error {
	return nil
}
