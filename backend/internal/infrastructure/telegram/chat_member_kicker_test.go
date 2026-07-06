package telegram

import (
	"context"
	"errors"
	"testing"

	telegrambot "github.com/go-telegram/bot"

	"gravel_bot/internal/application/command"
)

func TestChatMemberKickerBanThenUnban(t *testing.T) {
	api := &kickerAPIFake{}
	k := NewChatMemberKicker(api, -100)

	if err := k.Kick(context.Background(), 42); err != nil {
		t.Fatalf("Kick error: %v", err)
	}
	if len(api.banned) != 1 || api.banned[0] != 42 {
		t.Fatalf("ban calls = %v, want [42]", api.banned)
	}
	if len(api.unbanned) != 1 || api.unbanned[0] != 42 {
		t.Fatalf("unban calls = %v, want [42]", api.unbanned)
	}
}

func TestChatMemberKickerNotInChat(t *testing.T) {
	api := &kickerAPIFake{banErr: errors.New("bad request, USER_NOT_PARTICIPANT")}
	k := NewChatMemberKicker(api, -100)

	err := k.Kick(context.Background(), 42)
	if !errors.Is(err, command.ErrMemberNotInChat) {
		t.Fatalf("err = %v, want ErrMemberNotInChat", err)
	}
	if len(api.unbanned) != 0 {
		t.Fatal("unban must not run after not-in-chat ban error")
	}
}

func TestChatMemberKickerBanError(t *testing.T) {
	api := &kickerAPIFake{banErr: errors.New("network down")}
	k := NewChatMemberKicker(api, -100)

	err := k.Kick(context.Background(), 42)
	if err == nil || errors.Is(err, command.ErrMemberNotInChat) {
		t.Fatalf("err = %v, want generic ban error", err)
	}
}

func TestChatMemberKickerUnbanError(t *testing.T) {
	api := &kickerAPIFake{unbanErr: errors.New("network down")}
	k := NewChatMemberKicker(api, -100)

	if err := k.Kick(context.Background(), 42); err == nil {
		t.Fatal("expected unban error to surface")
	}
}

func TestNewChatMemberKickerFromTokenDisabled(t *testing.T) {
	k, err := NewChatMemberKickerFromToken("", 0)
	if err != nil || k != nil {
		t.Fatalf("disabled kicker: got k=%v err=%v, want nil,nil", k, err)
	}
}

type kickerAPIFake struct {
	banned   []int64
	unbanned []int64
	banErr   error
	unbanErr error
}

func (a *kickerAPIFake) BanChatMember(ctx context.Context, params *telegrambot.BanChatMemberParams) (bool, error) {
	if a.banErr != nil {
		return false, a.banErr
	}
	a.banned = append(a.banned, params.UserID)
	return true, nil
}

func (a *kickerAPIFake) UnbanChatMember(ctx context.Context, params *telegrambot.UnbanChatMemberParams) (bool, error) {
	if a.unbanErr != nil {
		return false, a.unbanErr
	}
	a.unbanned = append(a.unbanned, params.UserID)
	return true, nil
}
