package query

import (
	"context"
	"fmt"
	"strings"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// ChatPurgeCandidate — участник чата, предложенный к киковке.
type ChatPurgeCandidate struct {
	UserID   int64
	Label    string
	Username string
	Reason   string
}

// ChatPurgeCandidatesResult — результат подбора кандидатов на чистку.
type ChatPurgeCandidatesResult struct {
	Candidates          []ChatPurgeCandidate
	ProtectedGiftOwners int
}

// GetChatPurgeCandidatesHandler подбирает кандидатов на чистку публичного чата:
// текущие участники ростера без приза, исключая ботов и админов чата.
type GetChatPurgeCandidatesHandler struct {
	chatMemberRepo  repository.ChatMemberRepository
	giftRepo        repository.GiftRepository
	participantRepo repository.ParticipantRepository
}

// NewGetChatPurgeCandidatesHandler создаёт новый handler.
func NewGetChatPurgeCandidatesHandler(
	chatMemberRepo repository.ChatMemberRepository,
	giftRepo repository.GiftRepository,
	participantRepo repository.ParticipantRepository,
) *GetChatPurgeCandidatesHandler {
	return &GetChatPurgeCandidatesHandler{
		chatMemberRepo:  chatMemberRepo,
		giftRepo:        giftRepo,
		participantRepo: participantRepo,
	}
}

// Handle возвращает кандидатов на чистку для события eventID.
func (h *GetChatPurgeCandidatesHandler) Handle(ctx context.Context, eventID uint) (ChatPurgeCandidatesResult, error) {
	var result ChatPurgeCandidatesResult

	members, err := h.chatMemberRepo.GetAll(ctx)
	if err != nil {
		return result, fmt.Errorf("get chat members: %w", err)
	}

	gifts, err := h.giftRepo.FindByEvent(ctx, eventID)
	if err != nil {
		return result, fmt.Errorf("find event gifts: %w", err)
	}
	giftOwners := make(map[int64]struct{}, len(gifts))
	for _, gift := range gifts {
		if gift != nil && gift.UserID > 0 {
			giftOwners[gift.UserID] = struct{}{}
		}
	}

	participants, err := h.participantRepo.FindByEvent(ctx, eventID)
	if err != nil {
		return result, fmt.Errorf("find event participants: %w", err)
	}
	finished := make(map[int64]struct{}, len(participants))
	registered := make(map[int64]struct{}, len(participants))
	for _, participant := range participants {
		if participant == nil || participant.UserID <= 0 {
			continue
		}
		registered[participant.UserID] = struct{}{}
		if participant.IsFinished() {
			finished[participant.UserID] = struct{}{}
		}
	}

	candidates := make([]ChatPurgeCandidate, 0, len(members))
	protected := 0
	for _, member := range members {
		if member == nil || member.TelegramUserID <= 0 {
			continue
		}
		if _, hasGift := giftOwners[member.TelegramUserID]; hasGift {
			protected++
			continue
		}
		if member.IsBot || member.IsAdmin {
			continue
		}

		candidates = append(candidates, ChatPurgeCandidate{
			UserID:   member.TelegramUserID,
			Label:    chatMemberLabel(member),
			Username: member.Username,
			Reason:   chatPurgeReason(member.TelegramUserID, finished, registered),
		})
	}

	result.Candidates = candidates
	result.ProtectedGiftOwners = protected
	return result, nil
}

func chatMemberLabel(member *entity.ChatMember) string {
	name := strings.TrimSpace(member.FullName())
	username := strings.TrimSpace(member.Username)
	switch {
	case name != "" && username != "":
		return fmt.Sprintf("%s (@%s)", name, username)
	case name != "":
		return name
	case username != "":
		return "@" + username
	default:
		return fmt.Sprintf("id:%d", member.TelegramUserID)
	}
}

func chatPurgeReason(userID int64, finished, registered map[int64]struct{}) string {
	if _, ok := finished[userID]; ok {
		return "проехал, приза нет"
	}
	if _, ok := registered[userID]; ok {
		return "участник без приза"
	}
	return "не участник, приза нет"
}
