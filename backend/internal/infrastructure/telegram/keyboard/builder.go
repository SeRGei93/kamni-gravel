package keyboard

import (
	"github.com/go-telegram/bot/models"
)

// Builder предоставляет удобные методы для создания inline клавиатур
type Builder struct {
	rows [][]models.InlineKeyboardButton
}

// NewBuilder создаёт новый builder
func NewBuilder() *Builder {
	return &Builder{
		rows: make([][]models.InlineKeyboardButton, 0),
	}
}

// Button создаёт inline кнопку с callback data.
func Button(text, callbackData string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         text,
		CallbackData: callbackData,
	}
}

// ButtonURL создаёт inline кнопку со ссылкой.
func ButtonURL(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text: text,
		URL:  url,
	}
}

// AddRow добавляет новый ряд кнопок
func (b *Builder) AddRow(buttons ...models.InlineKeyboardButton) *Builder {
	b.rows = append(b.rows, buttons)
	return b
}

// AddButton добавляет кнопку в новый ряд
func (b *Builder) AddButton(text, callbackData string) *Builder {
	button := Button(text, callbackData)
	b.rows = append(b.rows, []models.InlineKeyboardButton{button})
	return b
}

// AddButtonURL добавляет кнопку с URL в новый ряд
func (b *Builder) AddButtonURL(text, url string) *Builder {
	button := ButtonURL(text, url)
	b.rows = append(b.rows, []models.InlineKeyboardButton{button})
	return b
}

// Build создаёт InlineKeyboardMarkup
func (b *Builder) Build() models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{
		InlineKeyboard: b.rows,
	}
}

// MainMenu создаёт главное меню
func MainMenu() models.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			Button("🚴 Зарегистрироваться", "register"),
		).
		AddRow(
			Button("🎁 Добавить подарок", "add_gift"),
		).
		AddRow(
			Button("🏁 Отправить результат", "submit_result"),
		).
		AddRow(
			Button("ℹ️ Информация", "info"),
		).
		Build()
}

// BikeTypeMenu создаёт меню выбора типа велосипеда
func BikeTypeMenu() models.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			Button("🚵 Гравийник", "bike_gravel"),
			Button("🏔 МТБ", "bike_mtb"),
		).
		AddRow(
			Button("🚴 Шоссе", "bike_road"),
			Button("🔧 Фикс", "bike_single_speed"),
		).
		AddRow(
			Button("👥 Тандем", "bike_tandem"),
		).
		AddRow(
			Button("❌ Отмена", "cancel"),
		).
		Build()
}

// GenderMenu создаёт меню выбора пола
func GenderMenu() models.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			Button("👨 Мужской", "gender_male"),
			Button("👩 Женский", "gender_female"),
		).
		AddRow(
			Button("❌ Отмена", "cancel"),
		).
		Build()
}

// CancelMenu создаёт меню с кнопкой отмены
func CancelMenu() models.InlineKeyboardMarkup {
	return NewBuilder().
		AddButton("❌ Отмена", "cancel").
		Build()
}

// GiftPhotoMenu создаёт меню для добавления фото подарка
func GiftPhotoMenu() models.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			Button("✅ Завершить", "finish_gift"),
			Button("⏭ Пропустить", "skip_photos"),
		).
		AddRow(
			Button("❌ Отмена", "cancel"),
		).
		Build()
}

// GiftConfirmationMenu создаёт меню подтверждения подарка перед сохранением.
func GiftConfirmationMenu() models.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			Button("✅ Подтвердить", "confirm_gift"),
			Button("🔄 Заполнить заново", "restart_gift"),
		).
		AddRow(
			Button("❌ Отмена", "cancel"),
		).
		Build()
}

// ConfirmMenu создаёт меню подтверждения
func ConfirmMenu(confirmData, cancelData string) models.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			Button("✅ Да", confirmData),
			Button("❌ Нет", cancelData),
		).
		Build()
}

// BackToMainMenu создаёт кнопку возврата в главное меню
func BackToMainMenu() models.InlineKeyboardMarkup {
	return NewBuilder().
		AddButton("🏠 Главное меню", "start").
		Build()
}
