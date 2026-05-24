package keyboard

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Builder предоставляет удобные методы для создания inline клавиатур
type Builder struct {
	rows [][]tgbotapi.InlineKeyboardButton
}

// NewBuilder создаёт новый builder
func NewBuilder() *Builder {
	return &Builder{
		rows: make([][]tgbotapi.InlineKeyboardButton, 0),
	}
}

// AddRow добавляет новый ряд кнопок
func (b *Builder) AddRow(buttons ...tgbotapi.InlineKeyboardButton) *Builder {
	b.rows = append(b.rows, buttons)
	return b
}

// AddButton добавляет кнопку в новый ряд
func (b *Builder) AddButton(text, callbackData string) *Builder {
	button := tgbotapi.NewInlineKeyboardButtonData(text, callbackData)
	b.rows = append(b.rows, []tgbotapi.InlineKeyboardButton{button})
	return b
}

// AddButtonURL добавляет кнопку с URL в новый ряд
func (b *Builder) AddButtonURL(text, url string) *Builder {
	button := tgbotapi.NewInlineKeyboardButtonURL(text, url)
	b.rows = append(b.rows, []tgbotapi.InlineKeyboardButton{button})
	return b
}

// Build создаёт InlineKeyboardMarkup
func (b *Builder) Build() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(b.rows...)
}

// MainMenu создаёт главное меню
func MainMenu() tgbotapi.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("🚴 Зарегистрироваться", "register"),
		).
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("🎁 Добавить подарок", "add_gift"),
		).
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("🏁 Отправить результат", "submit_result"),
		).
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Информация", "info"),
		).
		Build()
}

// BikeTypeMenu создаёт меню выбора типа велосипеда
func BikeTypeMenu() tgbotapi.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("🚵 Гравийник", "bike_gravel"),
			tgbotapi.NewInlineKeyboardButtonData("🏔 МТБ", "bike_mtb"),
		).
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("🚴 Шоссе", "bike_road"),
			tgbotapi.NewInlineKeyboardButtonData("🔧 Фикс", "bike_single_speed"),
		).
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Тандем", "bike_tandem"),
		).
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		).
		Build()
}

// GenderMenu создаёт меню выбора пола
func GenderMenu() tgbotapi.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("👨 Мужской", "gender_male"),
			tgbotapi.NewInlineKeyboardButtonData("👩 Женский", "gender_female"),
		).
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		).
		Build()
}

// CancelMenu создаёт меню с кнопкой отмены
func CancelMenu() tgbotapi.InlineKeyboardMarkup {
	return NewBuilder().
		AddButton("❌ Отмена", "cancel").
		Build()
}

// GiftPhotoMenu создаёт меню для добавления фото подарка
func GiftPhotoMenu() tgbotapi.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Завершить", "finish_gift"),
			tgbotapi.NewInlineKeyboardButtonData("⏭ Пропустить", "skip_photos"),
		).
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		).
		Build()
}

// ConfirmMenu создаёт меню подтверждения
func ConfirmMenu(confirmData, cancelData string) tgbotapi.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Да", confirmData),
			tgbotapi.NewInlineKeyboardButtonData("❌ Нет", cancelData),
		).
		Build()
}

// BackToMainMenu создаёт кнопку возврата в главное меню
func BackToMainMenu() tgbotapi.InlineKeyboardMarkup {
	return NewBuilder().
		AddButton("🏠 Главное меню", "start").
		Build()
}
