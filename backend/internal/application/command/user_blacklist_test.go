package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

func TestAddUserBlacklistHandlerTrimsReasonAndUpserts(t *testing.T) {
	repo := &userBlacklistRepoFake{}
	h := NewAddUserBlacklistHandler(repo)

	entry, err := h.Handle(context.Background(), AddUserBlacklistCommand{
		TelegramUserID: 123,
		Reason:         "  spam registrations  ",
	})
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if entry.Reason != "spam registrations" {
		t.Fatalf("reason mismatch: got %q", entry.Reason)
	}
	if !repo.upsertCalled || repo.upserted.TelegramUserID != 123 {
		t.Fatalf("upsert mismatch: called=%t entry=%#v", repo.upsertCalled, repo.upserted)
	}
}

func TestAddUserBlacklistHandlerRejectsInvalidTelegramUserID(t *testing.T) {
	repo := &userBlacklistRepoFake{}
	h := NewAddUserBlacklistHandler(repo)

	_, err := h.Handle(context.Background(), AddUserBlacklistCommand{TelegramUserID: 0})
	if !errors.Is(err, ErrInvalidTelegramUserID) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrInvalidTelegramUserID)
	}
	if repo.upsertCalled {
		t.Fatal("upsert should not be called for invalid telegram user id")
	}
}

func TestUpdateUserBlacklistReasonHandlerMapsNotFound(t *testing.T) {
	h := NewUpdateUserBlacklistReasonHandler(&userBlacklistRepoFake{updateErr: repository.ErrUserBlacklistEntryNotFound})

	_, err := h.Handle(context.Background(), UpdateUserBlacklistReasonCommand{
		TelegramUserID: 123,
		Reason:         "new reason",
	})
	if !errors.Is(err, ErrUserBlacklistNotFound) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrUserBlacklistNotFound)
	}
}

func TestRemoveUserBlacklistHandlerMapsNotFound(t *testing.T) {
	h := NewRemoveUserBlacklistHandler(&userBlacklistRepoFake{deleteErr: repository.ErrUserBlacklistEntryNotFound})

	err := h.Handle(context.Background(), RemoveUserBlacklistCommand{TelegramUserID: 123})
	if !errors.Is(err, ErrUserBlacklistNotFound) {
		t.Fatalf("error mismatch: got %v, want %v", err, ErrUserBlacklistNotFound)
	}
}

type userBlacklistRepoFake struct {
	entries      []*entity.UserBlacklist
	blacklisted  bool
	upsertCalled bool
	upserted     *entity.UserBlacklist
	updateErr    error
	deleteErr    error
}

func (r *userBlacklistRepoFake) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	return r.entries, nil
}

func (r *userBlacklistRepoFake) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	for _, entry := range r.entries {
		if entry.TelegramUserID == telegramUserID {
			return entry, nil
		}
	}
	return nil, repository.ErrUserBlacklistEntryNotFound
}

func (r *userBlacklistRepoFake) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	return r.blacklisted, nil
}

func (r *userBlacklistRepoFake) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	r.upsertCalled = true
	r.upserted = entry
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	entry.UpdatedAt = entry.CreatedAt
	return nil
}

func (r *userBlacklistRepoFake) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	return &entity.UserBlacklist{TelegramUserID: telegramUserID, Reason: reason}, nil
}

func (r *userBlacklistRepoFake) Delete(ctx context.Context, telegramUserID int64) error {
	return r.deleteErr
}
