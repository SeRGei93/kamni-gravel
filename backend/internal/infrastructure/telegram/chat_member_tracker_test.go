package telegram

import (
	"context"
	"testing"

	"github.com/go-telegram/bot/models"

	"gravel_bot/internal/domain/entity"
)

func TestHandleChatMemberUpdateUpsertOnJoin(t *testing.T) {
	repo := &chatMemberRepoFake{}
	b := &Bot{publicChatID: -100, chatMemberRepo: repo}

	b.handleChatMemberUpdate(context.Background(), &models.ChatMemberUpdated{
		Chat: models.Chat{ID: -100},
		NewChatMember: models.ChatMember{
			Type:   models.ChatMemberTypeMember,
			Member: &models.ChatMemberMember{User: &models.User{ID: 42, Username: "rider", FirstName: "Anna"}},
		},
	})

	if len(repo.upserted) != 1 || repo.upserted[0].TelegramUserID != 42 || repo.upserted[0].IsAdmin {
		t.Fatalf("upsert mismatch: %+v", repo.upserted)
	}
	if len(repo.deleted) != 0 {
		t.Fatalf("unexpected deletions: %v", repo.deleted)
	}
}

func TestHandleChatMemberUpdateAdminFlag(t *testing.T) {
	repo := &chatMemberRepoFake{}
	b := &Bot{publicChatID: -100, chatMemberRepo: repo}

	b.handleChatMemberUpdate(context.Background(), &models.ChatMemberUpdated{
		Chat: models.Chat{ID: -100},
		NewChatMember: models.ChatMember{
			Type:          models.ChatMemberTypeAdministrator,
			Administrator: &models.ChatMemberAdministrator{User: models.User{ID: 7, FirstName: "Adm"}},
		},
	})

	if len(repo.upserted) != 1 || !repo.upserted[0].IsAdmin {
		t.Fatalf("admin upsert mismatch: %+v", repo.upserted)
	}
}

func TestHandleChatMemberUpdateDeleteOnLeave(t *testing.T) {
	repo := &chatMemberRepoFake{}
	b := &Bot{publicChatID: -100, chatMemberRepo: repo}

	for _, member := range []models.ChatMember{
		{Type: models.ChatMemberTypeLeft, Left: &models.ChatMemberLeft{User: &models.User{ID: 1}}},
		{Type: models.ChatMemberTypeBanned, Banned: &models.ChatMemberBanned{User: &models.User{ID: 2}}},
		{Type: models.ChatMemberTypeRestricted, Restricted: &models.ChatMemberRestricted{User: &models.User{ID: 3}, IsMember: false}},
	} {
		b.handleChatMemberUpdate(context.Background(), &models.ChatMemberUpdated{
			Chat:          models.Chat{ID: -100},
			NewChatMember: member,
		})
	}

	if len(repo.deleted) != 3 {
		t.Fatalf("deletions = %v, want [1 2 3]", repo.deleted)
	}
	if len(repo.upserted) != 0 {
		t.Fatalf("unexpected upserts: %+v", repo.upserted)
	}
}

func TestHandleChatMemberUpdateIgnoresOtherChat(t *testing.T) {
	repo := &chatMemberRepoFake{}
	b := &Bot{publicChatID: -100, chatMemberRepo: repo}

	b.handleChatMemberUpdate(context.Background(), &models.ChatMemberUpdated{
		Chat: models.Chat{ID: -999},
		NewChatMember: models.ChatMember{
			Type:   models.ChatMemberTypeMember,
			Member: &models.ChatMemberMember{User: &models.User{ID: 42}},
		},
	})

	if len(repo.upserted) != 0 || len(repo.deleted) != 0 {
		t.Fatal("update outside public chat must be ignored")
	}
}

type chatMemberRepoFake struct {
	upserted []*entity.ChatMember
	deleted  []int64
}

func (r *chatMemberRepoFake) Upsert(ctx context.Context, m *entity.ChatMember) error {
	r.upserted = append(r.upserted, m)
	return nil
}
func (r *chatMemberRepoFake) BulkUpsert(ctx context.Context, m []*entity.ChatMember) error {
	r.upserted = append(r.upserted, m...)
	return nil
}
func (r *chatMemberRepoFake) Delete(ctx context.Context, id int64) error {
	r.deleted = append(r.deleted, id)
	return nil
}
func (r *chatMemberRepoFake) GetAll(ctx context.Context) ([]*entity.ChatMember, error) {
	return nil, nil
}
func (r *chatMemberRepoFake) Count(ctx context.Context) (int, error) { return 0, nil }
