package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/application/query"
	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/infrastructure/telegram/session"
)

func TestBotSilentlyIgnoresBlacklistedUpdates(t *testing.T) {
	const userID int64 = 12345

	tests := []struct {
		name   string
		update *models.Update
		assert func(t *testing.T, manager *session.Manager)
	}{
		{
			name: "start command",
			update: &models.Update{
				ID: 1,
				Message: &models.Message{
					ID:       10,
					From:     &models.User{ID: userID, FirstName: "Blocked"},
					Chat:     models.Chat{ID: userID},
					Text:     "/start",
					Entities: []models.MessageEntity{{Type: models.MessageEntityTypeBotCommand, Offset: 0, Length: 6}},
				},
			},
		},
		{
			name: "callback",
			update: &models.Update{
				ID: 2,
				CallbackQuery: &models.CallbackQuery{
					ID:   "callback-id",
					From: models.User{ID: userID, FirstName: "Blocked"},
					Data: "cancel",
					Message: models.MaybeInaccessibleMessage{
						Message: &models.Message{ID: 11, Chat: models.Chat{ID: userID}},
					},
				},
			},
			assert: func(t *testing.T, manager *session.Manager) {
				if got := manager.GetState(userID); got != session.StateAwaitingGiftDesc {
					t.Fatalf("state mutated: got %s, want %s", got, session.StateAwaitingGiftDesc)
				}
			},
		},
		{
			name: "free text message",
			update: &models.Update{
				ID: 3,
				Message: &models.Message{
					ID:   12,
					From: &models.User{ID: userID, FirstName: "Blocked"},
					Chat: models.Chat{ID: userID},
					Text: "new gift description",
				},
			},
			assert: func(t *testing.T, manager *session.Manager) {
				if _, ok := manager.GetData(userID, "gift_description"); ok {
					t.Fatal("gift_description should not be written for blacklisted user")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := session.NewManager(time.Minute)
			manager.SetState(userID, session.StateAwaitingGiftDesc)
			blacklistRepo := &botUserBlacklistRepoFake{blacklisted: true}
			b := &Bot{
				sessionManager:           manager,
				isUserBlacklistedHandler: query.NewIsUserBlacklistedHandler(blacklistRepo),
			}

			b.handleUpdate(context.Background(), nil, tt.update)

			if blacklistRepo.checkedTelegramUserID != userID {
				t.Fatalf("blacklist check mismatch: got %d, want %d", blacklistRepo.checkedTelegramUserID, userID)
			}
			if tt.assert != nil {
				tt.assert(t, manager)
			}
		})
	}
}

type botUserBlacklistRepoFake struct {
	blacklisted           bool
	checkedTelegramUserID int64
}

func (r *botUserBlacklistRepoFake) List(ctx context.Context) ([]*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *botUserBlacklistRepoFake) FindByTelegramUserID(ctx context.Context, telegramUserID int64) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *botUserBlacklistRepoFake) IsBlacklisted(ctx context.Context, telegramUserID int64) (bool, error) {
	r.checkedTelegramUserID = telegramUserID
	return r.blacklisted, nil
}
func (r *botUserBlacklistRepoFake) Upsert(ctx context.Context, entry *entity.UserBlacklist) error {
	return nil
}
func (r *botUserBlacklistRepoFake) UpdateReason(ctx context.Context, telegramUserID int64, reason string) (*entity.UserBlacklist, error) {
	return nil, nil
}
func (r *botUserBlacklistRepoFake) Delete(ctx context.Context, telegramUserID int64) error {
	return nil
}
