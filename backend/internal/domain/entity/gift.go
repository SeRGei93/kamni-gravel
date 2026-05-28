package entity

import (
	"fmt"
	"time"

	"gravel_bot/internal/domain/valueobject"
)

// GiftReviewStatus представляет статус проверки подарка администратором.
type GiftReviewStatus string

const (
	GiftReviewStatusPendingReview GiftReviewStatus = "pending_review"
	GiftReviewStatusApproved      GiftReviewStatus = "approved"
)

// NewGiftReviewStatus создаёт и валидирует статус проверки подарка.
func NewGiftReviewStatus(value string) (GiftReviewStatus, error) {
	status := GiftReviewStatus(value)
	if !status.IsValid() {
		return "", fmt.Errorf("invalid gift review status: %s", value)
	}
	return status, nil
}

// IsValid проверяет валидность статуса проверки подарка.
func (s GiftReviewStatus) IsValid() bool {
	switch s {
	case GiftReviewStatusPendingReview, GiftReviewStatusApproved:
		return true
	}
	return false
}

// String возвращает строковое представление статуса проверки подарка.
func (s GiftReviewStatus) String() string {
	return string(s)
}

// Gift представляет подарок от участника
type Gift struct {
	ID             uint
	UserID         int64
	EventID        uint
	Description    string
	GenderFilter   string // all, male, female
	BikeTypeFilter string // all, gravel, mtb, road, single_speed, tandem
	ReviewStatus   GiftReviewStatus
	Place          *int // место (позиция), nil если не задано
	PlaceRule      valueobject.GiftPlaceRule
	CreatedAt      time.Time

	// Связанные сущности
	User        *User
	Attachments []GiftAttachment
	Criteria    []*Criteria // Критерии, привязанные к подарку
}

// HasAttachments проверяет, есть ли у подарка прикреплённые файлы
func (g *Gift) HasAttachments() bool {
	return len(g.Attachments) > 0
}

// HasCriteria проверяет, есть ли у подарка привязанные критерии
func (g *Gift) HasCriteria() bool {
	return len(g.Criteria) > 0
}

// HasPlaceRule проверяет, есть ли у подарка новое правило места.
func (g *Gift) HasPlaceRule() bool {
	return !g.PlaceRule.IsNone()
}

// HasPlaceConstraint проверяет, ограничивает ли подарок распределение местом.
func (g *Gift) HasPlaceConstraint() bool {
	return g.PlaceRule.HasPlaceConstraint() || g.Place != nil
}

// FirstLegacyPlace возвращает место для временной совместимости со старым API.
func (g *Gift) FirstLegacyPlace() *int {
	if place := g.PlaceRule.FirstLegacyPlace(); place != nil {
		return place
	}
	return g.Place
}

// GiftAttachment представляет прикреплённый файл к подарку
type GiftAttachment struct {
	ID             uint
	GiftID         uint
	TelegramFileID string
	FileType       string // photo, document
}
