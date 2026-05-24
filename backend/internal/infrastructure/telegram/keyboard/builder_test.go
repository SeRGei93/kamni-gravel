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
			menu: MainMenu(),
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

func TestBuilderCreatesRows(t *testing.T) {
	menu := NewBuilder().
		AddButton("A", "a").
		AddRow(Button("B", "b"), Button("C", "c")).
		Build()

	if got := len(menu.InlineKeyboard); got != 2 {
		t.Fatalf("row count mismatch: got %d, want 2", got)
	}
	if got := len(menu.InlineKeyboard[1]); got != 2 {
		t.Fatalf("second row button count mismatch: got %d, want 2", got)
	}
}

func callbackData(menu models.InlineKeyboardMarkup) []string {
	var data []string
	for _, row := range menu.InlineKeyboard {
		for _, button := range row {
			data = append(data, button.CallbackData)
		}
	}
	return data
}
