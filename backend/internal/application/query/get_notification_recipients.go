package query

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"gravel_bot/internal/domain/entity"
	"gravel_bot/internal/domain/repository"
)

// NotificationRecipientFilter задаёт срез участников для админской рассылки.
type NotificationRecipientFilter string

const (
	NotificationRecipientFilterAll                     NotificationRecipientFilter = "all"
	NotificationRecipientFilterFinishedWithoutGift     NotificationRecipientFilter = "finished_without_gift"
	NotificationRecipientFilterGiftWithoutFinish       NotificationRecipientFilter = "gift_without_finish"
	NotificationRecipientFilterPendingManualGiftOwners NotificationRecipientFilter = "pending_manual_gift_owners"
)

// NewNotificationRecipientFilter валидирует значение фильтра из HTTP-запроса.
func NewNotificationRecipientFilter(value string) (NotificationRecipientFilter, error) {
	filter := NotificationRecipientFilter(strings.TrimSpace(value))
	if filter == "" {
		filter = NotificationRecipientFilterAll
	}

	switch filter {
	case NotificationRecipientFilterAll,
		NotificationRecipientFilterFinishedWithoutGift,
		NotificationRecipientFilterGiftWithoutFinish,
		NotificationRecipientFilterPendingManualGiftOwners:
		return filter, nil
	default:
		return "", fmt.Errorf("invalid notification recipient filter: %s", value)
	}
}

// NotificationRecipient — участник, которому можно отправить личное сообщение.
type NotificationRecipient struct {
	UserID                int64
	Label                 string
	Username              string
	Status                string
	HasGift               bool
	HasPendingManualGifts bool
}

// GetNotificationRecipientsHandler подбирает участников активного события для
// админской рассылки. Фильтр «pending_manual_gift_owners» выбирает владельцев
// призов с ручным распределением, для которых ещё не выбран получатель.
type GetNotificationRecipientsHandler struct {
	participantRepo repository.ParticipantRepository
	giftRepo        repository.GiftRepository
}

// NewGetNotificationRecipientsHandler создаёт handler выборки получателей.
func NewGetNotificationRecipientsHandler(
	participantRepo repository.ParticipantRepository,
	giftRepo repository.GiftRepository,
) *GetNotificationRecipientsHandler {
	return &GetNotificationRecipientsHandler{
		participantRepo: participantRepo,
		giftRepo:        giftRepo,
	}
}

// Handle возвращает уникальных участников события, соответствующих фильтру.
func (h *GetNotificationRecipientsHandler) Handle(
	ctx context.Context,
	eventID uint,
	filter NotificationRecipientFilter,
) ([]NotificationRecipient, error) {
	participants, err := h.participantRepo.FindByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("find event participants: %w", err)
	}

	gifts, err := h.giftRepo.FindByEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("find event gifts: %w", err)
	}
	giftOwners := notificationGiftOwners(gifts)

	pendingManualGiftOwners := map[int64]struct{}{}
	if filter == NotificationRecipientFilterPendingManualGiftOwners {
		pendingManualGiftOwners = notificationPendingManualGiftOwners(gifts)
	}

	recipients := make([]NotificationRecipient, 0, len(participants))
	seen := make(map[int64]struct{}, len(participants))
	for _, participant := range participants {
		if participant == nil || participant.UserID <= 0 {
			continue
		}
		if _, alreadyAdded := seen[participant.UserID]; alreadyAdded {
			continue
		}

		_, hasGift := giftOwners[participant.UserID]
		_, hasPendingManualGifts := pendingManualGiftOwners[participant.UserID]
		if !matchesNotificationRecipientFilter(participant, filter, hasGift, hasPendingManualGifts) {
			continue
		}

		seen[participant.UserID] = struct{}{}
		recipients = append(recipients, NotificationRecipient{
			UserID:                participant.UserID,
			Label:                 notificationRecipientLabel(participant.User, participant.UserID),
			Username:              notificationRecipientUsername(participant.User),
			Status:                notificationRecipientStatus(participant),
			HasGift:               hasGift,
			HasPendingManualGifts: hasPendingManualGifts,
		})
	}

	sort.SliceStable(recipients, func(i, j int) bool {
		left := strings.ToLower(recipients[i].Label)
		right := strings.ToLower(recipients[j].Label)
		if left == right {
			return recipients[i].UserID < recipients[j].UserID
		}
		return left < right
	})

	return recipients, nil
}

func notificationGiftOwners(gifts []*entity.Gift) map[int64]struct{} {
	owners := make(map[int64]struct{}, len(gifts))
	for _, gift := range gifts {
		if gift != nil && gift.UserID > 0 {
			owners[gift.UserID] = struct{}{}
		}
	}
	return owners
}

func notificationPendingManualGiftOwners(gifts []*entity.Gift) map[int64]struct{} {
	owners := make(map[int64]struct{})
	for _, gift := range gifts {
		if gift == nil || gift.UserID <= 0 || !gift.ManualDistribution || gift.ManualRecipientParticipantID != nil {
			continue
		}
		owners[gift.UserID] = struct{}{}
	}
	return owners
}

func matchesNotificationRecipientFilter(
	participant *entity.Participant,
	filter NotificationRecipientFilter,
	hasGift bool,
	hasPendingManualGifts bool,
) bool {
	switch filter {
	case NotificationRecipientFilterAll:
		return true
	case NotificationRecipientFilterFinishedWithoutGift:
		return participant.IsRanked() && participant.IsFinished() && !hasGift
	case NotificationRecipientFilterGiftWithoutFinish:
		return hasGift && !participant.IsFinished()
	case NotificationRecipientFilterPendingManualGiftOwners:
		return hasPendingManualGifts
	default:
		return false
	}
}

func notificationRecipientLabel(user *entity.User, userID int64) string {
	if user == nil {
		return fmt.Sprintf("id:%d", userID)
	}

	name := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(user.FirstName),
		strings.TrimSpace(user.LastName),
	}, " "))
	username := strings.TrimSpace(user.Username)

	switch {
	case name != "" && username != "":
		return fmt.Sprintf("%s (@%s)", name, username)
	case name != "":
		return name
	case username != "":
		return "@" + username
	default:
		return fmt.Sprintf("id:%d", userID)
	}
}

func notificationRecipientUsername(user *entity.User) string {
	if user == nil {
		return ""
	}
	return strings.TrimSpace(user.Username)
}

func notificationRecipientStatus(participant *entity.Participant) string {
	if participant.IsDisqualified() {
		return "дисквалифицирован"
	}
	if participant.IsDNF() {
		return "сошёл с дистанции"
	}
	if participant.IsFinished() {
		return "проехал"
	}
	return "не проехал"
}
