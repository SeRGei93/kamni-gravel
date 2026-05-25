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
			menu: MainMenu(""),
			want: []string{"register", "add_gift", "submit_result", "info"},
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
			name: "gift photo menu",
			menu: GiftPhotoMenu(),
			want: []string{"finish_gift", "skip_photos", "cancel"},
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
			if got := callbackData(tt.menu); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("callback data mismatch: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMainMenuAddsOptionalWebAppButton(t *testing.T) {
	const miniappURL = "https://example.com/miniapp/gifts"

	menu := MainMenu(miniappURL)

	if got := callbackData(menu); !reflect.DeepEqual(got, []string{"register", "add_gift", "submit_result", "info"}) {
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
	if got := webAppButton.Text; got != "🎁 Смотреть подарки" {
		t.Fatalf("web app button text mismatch: got %q", got)
	}
	if got := webAppButton.WebApp.URL; got != miniappURL {
		t.Fatalf("web app URL mismatch: got %q, want %q", got, miniappURL)
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
