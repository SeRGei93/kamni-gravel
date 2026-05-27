package telegram

import (
	"strings"
	"testing"

	"gravel_bot/internal/domain/entity"
)

func TestAdminGiftSummaryTextContainsRequiredMetadata(t *testing.T) {
	b := &Bot{}
	text := b.adminGiftSummaryText(&entity.Gift{
		ID:             77,
		EventID:        55,
		UserID:         12345,
		GenderFilter:   "female",
		BikeTypeFilter: "gravel",
		ReviewStatus:   entity.GiftReviewStatusPendingReview,
		Attachments:    []entity.GiftAttachment{{ID: 1}, {ID: 2}},
	})

	if text == "" {
		t.Fatal("summary text must not be empty")
	}
	for _, token := range []string{"ID подарка: 77", "ID события: 55", "Пользователь: 12345", "Фильтр пола: female", "Фильтр велосипеда: gravel", "Фото: 2"} {
		if !strings.Contains(text, token) {
			t.Fatalf("summary text missing token %q in %q", token, text)
		}
	}
	if strings.Contains(text, `\n`) {
		t.Fatalf("summary text should contain real newlines, got escaped newlines in %q", text)
	}
	if !strings.Contains(text, "\n\nID подарка: 77") {
		t.Fatalf("summary text missing expected newline formatting: %q", text)
	}
}

func TestAdminGiftSummaryTextNilGift(t *testing.T) {
	b := &Bot{}
	text := b.adminGiftSummaryText(nil)
	if text == "" {
		t.Fatal("summary text must not be empty for nil gift")
	}
}
