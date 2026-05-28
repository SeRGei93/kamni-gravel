package keyboard

import (
	"reflect"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestMenusPreserveCallbackData(t *testing.T) {
	tests := []struct {
		name string
		menu models.InlineKeyboardMarkup
		want []string
	}{
		{
			name: "main menu",
			menu: MainMenu(false, false, "", nil),
			want: []string{},
		},
		{
			name: "main menu participant",
			menu: MainMenu(true, true, "", nil),
			want: []string{"withdraw_participation", "add_gift", "submit_result", "event_conditions"},
		},
		{
			name: "main menu not participant",
			menu: MainMenu(true, false, "", nil),
			want: []string{"register", "add_gift", "event_conditions"},
		},
		{
			name: "bike type menu",
			menu: BikeTypeMenu(),
			want: []string{"bike_gravel", "bike_mtb", "bike_road", "bike_single_speed", "bike_tandem", "cancel"},
		},
		{
			name: "gender menu",
			menu: GenderMenu(),
			want: []string{"gender_male", "gender_female", "cancel"},
		},
		{
			name: "registration consent menu",
			menu: RegistrationConsentMenu(),
			want: []string{"registration_accept_conditions", "registration_decline_conditions"},
		},
		{
			name: "gift photo menu",
			menu: GiftPhotoMenu(),
			want: []string{"finish_gift", "skip_photos", "cancel"},
		},
		{
			name: "gift draft menu without description",
			menu: GiftDraftMenu(false),
			want: []string{"restart_gift", "cancel"},
		},
		{
			name: "gift draft menu with description",
			menu: GiftDraftMenu(true),
			want: []string{"finish_gift", "restart_gift", "cancel"},
		},
		{
			name: "gift confirmation menu",
			menu: GiftConfirmationMenu(),
			want: []string{"confirm_gift", "restart_gift", "cancel"},
		},
		{
			name: "confirm menu",
			menu: ConfirmMenu("confirm_yes", "confirm_no"),
			want: []string{"confirm_yes", "confirm_no"},
		},
		{
			name: "back to main",
			menu: BackToMainMenu(),
			want: []string{"start"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := callbackData(tt.menu)
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("callback data mismatch: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMainMenuAddsOptionalWebAppButton(t *testing.T) {
	const miniappURL = "https://example.com/miniapp/gifts"

	menu := MainMenu(true, false, miniappURL, nil)

	if got := callbackData(menu); !reflect.DeepEqual(got, []string{"register", "add_gift", "event_conditions"}) {
		t.Fatalf("callback data mismatch: got %v", got)
	}

	var webAppButton *models.InlineKeyboardButton
	for _, row := range menu.InlineKeyboard {
		for i := range row {
			if row[i].WebApp != nil {
				webAppButton = &row[i]
			}
		}
	}

	if webAppButton == nil {
		t.Fatal("web app button not found")
	}
	if webAppButton.CallbackData != "" {
		t.Fatalf("web app button callback data mismatch: got %q, want empty", webAppButton.CallbackData)
	}
	if got := webAppButton.Text; got != "🏆 Призовой фонд" {
		t.Fatalf("web app button text mismatch: got %q", got)
	}
	if got := webAppButton.WebApp.URL; got != miniappURL {
		t.Fatalf("web app URL mismatch: got %q, want %q", got, miniappURL)
	}
}

func TestMainMenuOmitsWebAppButtonWhenURLIsEmpty(t *testing.T) {
	menu := MainMenu(true, false, "", nil)

	for _, row := range menu.InlineKeyboard {
		for _, button := range row {
			if button.WebApp != nil {
				t.Fatalf("unexpected web app button: %#v", button)
			}
		}
	}
}

func TestMainMenuUsesDeepLinks(t *testing.T) {
	menu := MainMenu(true, false, "", &MainMenuDeepLinks{
		Register:   "https://t.me/gravel_bot?start=register",
		Conditions: "https://t.me/gravel_bot?start=conditions",
	})

	var registerButton *models.InlineKeyboardButton
	var conditionsButton *models.InlineKeyboardButton
	for _, row := range menu.InlineKeyboard {
		for i := range row {
			switch row[i].Text {
			case "✅ Принять участие":
				registerButton = &row[i]
			case "‼️ Условия участия":
				conditionsButton = &row[i]
			}
		}
	}

	if registerButton == nil || registerButton.CallbackData != "" || registerButton.URL != "https://t.me/gravel_bot?start=register" {
		t.Fatalf("register deep link mismatch: %#v", registerButton)
	}
	if conditionsButton == nil || conditionsButton.CallbackData != "" || conditionsButton.URL != "https://t.me/gravel_bot?start=conditions" {
		t.Fatalf("conditions deep link mismatch: %#v", conditionsButton)
	}
}

func TestPublicMenuBuildsDeepLinks(t *testing.T) {
	menu := PublicMenu(
		"https://example.com/miniapp/gifts",
		"https://t.me/gravel_bot?start=register",
		"https://t.me/gravel_bot?start=conditions",
	)

	if got := callbackData(menu); len(got) != 0 {
		t.Fatalf("callback data mismatch: got %v", got)
	}

	var registerFound bool
	var miniappFound bool
	var conditionsFound bool

	for _, row := range menu.InlineKeyboard {
		for _, button := range row {
			switch button.Text {
			case "✅ Принять участие":
				registerFound = button.URL == "https://t.me/gravel_bot?start=register"
			case "🏆 Призовой фонд":
				miniappFound = button.WebApp != nil && button.WebApp.URL == "https://example.com/miniapp/gifts"
			case "‼️ Условия участия":
				conditionsFound = button.URL == "https://t.me/gravel_bot?start=conditions"
			}
		}
	}

	if !registerFound {
		t.Fatal("register deep link not found in public menu")
	}
	if !miniappFound {
		t.Fatal("miniapp button not found in public menu")
	}
	if !conditionsFound {
		t.Fatal("conditions deep link not found in public menu")
	}
}

func TestBuilderCreatesRows(t *testing.T) {
	menu := NewBuilder().
		AddButton("A", "a").
		AddButtonWebApp("Miniapp", "https://example.com/app").
		AddRow(Button("B", "b"), Button("C", "c")).
		Build()

	if got := len(menu.InlineKeyboard); got != 3 {
		t.Fatalf("row count mismatch: got %d, want 3", got)
	}
	if got := len(menu.InlineKeyboard[2]); got != 2 {
		t.Fatalf("third row button count mismatch: got %d, want 2", got)
	}
	if webApp := menu.InlineKeyboard[1][0].WebApp; webApp == nil || webApp.URL != "https://example.com/app" {
		t.Fatalf("web app button mismatch: got %#v", webApp)
	}
}

func callbackData(menu models.InlineKeyboardMarkup) []string {
	var data []string
	for _, row := range menu.InlineKeyboard {
		for _, button := range row {
			if button.CallbackData == "" {
				continue
			}
			data = append(data, button.CallbackData)
		}
	}
	return data
}
