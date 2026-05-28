package telegram

import (
	"strings"
	"testing"

	"gravel_bot/internal/domain/entity"
)

func TestAdminGiftNotificationTextContainsPublicReadyGiftData(t *testing.T) {
	b := &Bot{}
	text := b.adminGiftNotificationText(&entity.Gift{
		ID:             77,
		EventID:        55,
		UserID:         12345,
		Description:    "Лабуба за 1 и 10 место",
		GenderFilter:   "female",
		BikeTypeFilter: "gravel",
		ReviewStatus:   entity.GiftReviewStatusPendingReview,
		Attachments:    []entity.GiftAttachment{{ID: 1}, {ID: 2}},
		User:           &entity.User{ID: 12345, Username: "alex", FirstName: "Alex", LastName: "Rider"},
	}, telegramCaptionLimit)

	if text == "" {
		t.Fatal("notification text must not be empty")
	}
	for _, token := range []string{"Новый приз", "От: Alex Rider (@alex)", "Описание: Лабуба за 1 и 10 место", "Гендер: 👩 Женский", "Велосипед: 🚵 Гравийник"} {
		if !strings.Contains(text, token) {
			t.Fatalf("notification text missing token %q in %q", token, text)
		}
	}
	for _, internalToken := range []string{"Новый подарок на проверку", "ID подарка", "ID события", "Статус", "pending_review", "Фото: 2", "12345"} {
		if strings.Contains(text, internalToken) {
			t.Fatalf("notification text exposes internal token %q in %q", internalToken, text)
		}
	}
	if strings.Contains(text, `\n`) {
		t.Fatalf("notification text should contain real newlines, got escaped newlines in %q", text)
	}
}

func TestAdminGiftNotificationTextNilGift(t *testing.T) {
	b := &Bot{}
	text := b.adminGiftNotificationText(nil, telegramCaptionLimit)
	if text == "" {
		t.Fatal("notification text must not be empty for nil gift")
	}
}

func TestAdminGiftNotificationTextFallsBackToUsernameOrUserID(t *testing.T) {
	b := &Bot{}

	usernameOnly := b.adminGiftNotificationText(&entity.Gift{
		UserID:         12345,
		Description:    "Фляга",
		GenderFilter:   "all",
		BikeTypeFilter: "road",
		User:           &entity.User{ID: 12345, Username: "@alex"},
	}, telegramCaptionLimit)
	if !strings.Contains(usernameOnly, "От: @alex") {
		t.Fatalf("username fallback mismatch: %q", usernameOnly)
	}
	if strings.Contains(usernameOnly, "12345") {
		t.Fatalf("username fallback should hide user id: %q", usernameOnly)
	}

	userIDOnly := b.adminGiftNotificationText(&entity.Gift{
		UserID:         12345,
		Description:    "Фляга",
		GenderFilter:   "all",
		BikeTypeFilter: "road",
	}, telegramCaptionLimit)
	if !strings.Contains(userIDOnly, "От: user_id: 12345") {
		t.Fatalf("user id fallback mismatch: %q", userIDOnly)
	}
}

func TestAdminGiftNotificationTextTruncatesOnlyDescriptionToCaptionLimit(t *testing.T) {
	b := &Bot{}
	longDescription := strings.Repeat("очень длинное описание ", 80)

	text := b.adminGiftNotificationText(&entity.Gift{
		UserID:         12345,
		Description:    longDescription,
		GenderFilter:   "male",
		BikeTypeFilter: "single_speed",
		User:           &entity.User{ID: 12345, FirstName: "Alex"},
	}, telegramCaptionLimit)

	if got := len([]rune(text)); got > telegramCaptionLimit {
		t.Fatalf("caption text exceeds limit: got %d, want <= %d", got, telegramCaptionLimit)
	}
	for _, token := range []string{"От: Alex", "Гендер: 👨 Мужской", "Велосипед: ⚡️ Фикс"} {
		if !strings.Contains(text, token) {
			t.Fatalf("required token disappeared after truncation: %q in %q", token, text)
		}
	}
	if !strings.Contains(text, "...") {
		t.Fatalf("long description should be truncated with ellipsis: %q", text)
	}
}

func TestAdminGiftNotificationHTMLTextAddsHiddenMiniappLinkAndEscapesData(t *testing.T) {
	b := &Bot{}

	text := b.adminGiftNotificationHTMLText(&entity.Gift{
		UserID:         12345,
		Description:    "Фляга <тест> & ремешок",
		GenderFilter:   "female",
		BikeTypeFilter: "road",
		User:           &entity.User{ID: 12345, Username: "alex&co", FirstName: "Alex", LastName: "<Rider>"},
	}, telegramCaptionLimit, "https://t.me/GravelBot?startapp")

	for _, token := range []string{
		"От: Alex &lt;Rider&gt; (@alex&amp;co)",
		"Описание: Фляга &lt;тест&gt; &amp; ремешок",
		"Гендер: 👩 Женский",
		"Велосипед: 🚴 Шоссе",
		`<a href="https://t.me/GravelBot?startapp">призовой фонд</a>`,
	} {
		if !strings.Contains(text, token) {
			t.Fatalf("html notification text missing token %q in %q", token, text)
		}
	}
}

func TestAdminGiftMiniappTelegramLinkUsesBotUsername(t *testing.T) {
	b := &Bot{
		botUsername: "@GravelBot",
		miniappURL:  "https://example.com/miniapp/gifts",
	}

	link, ok := b.adminGiftMiniappTelegramLink()
	if !ok {
		t.Fatal("miniapp Telegram link should be available")
	}
	if link != "https://t.me/GravelBot?startapp" {
		t.Fatalf("miniapp Telegram link mismatch: got %q", link)
	}
}

func TestAdminGiftMiniappTelegramLinkMissingUsername(t *testing.T) {
	b := &Bot{miniappURL: "https://example.com/miniapp/gifts"}

	link, ok := b.adminGiftMiniappTelegramLink()
	if ok || link != "" {
		t.Fatalf("miniapp Telegram link should be unavailable without bot username, ok=%t link=%q", ok, link)
	}
}
