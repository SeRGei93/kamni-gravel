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

// ButtonWebApp создаёт inline кнопку для запуска Telegram WebApp.
func ButtonWebApp(text, webAppURL string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:   text,
		WebApp: &models.WebAppInfo{URL: webAppURL},
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

// AddButtonWebApp добавляет кнопку запуска Telegram WebApp в новый ряд.
func (b *Builder) AddButtonWebApp(text, webAppURL string) *Builder {
	button := ButtonWebApp(text, webAppURL)
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
type MainMenuDeepLinks struct {
	Register   string
	Conditions string
}

func MainMenu(hasActiveEvent bool, isRegistered bool, miniappURL string, deepLinks *MainMenuDeepLinks) models.InlineKeyboardMarkup {
	if !hasActiveEvent {
		return models.InlineKeyboardMarkup{}
	}

	builder := NewBuilder()

	if isRegistered {
		builder.AddRow(Button("😢 Отказаться от участия", "withdraw_participation"))
	} else {
		if deepLinks != nil && deepLinks.Register != "" {
			builder.AddRow(ButtonURL("✅ Принять участие", deepLinks.Register))
		} else {
			builder.AddRow(Button("✅ Принять участие", "register"))
		}
	}

	builder.AddRow(Button("🎁 Добавить приз", "add_gift"))

	if isRegistered {
		builder.AddRow(Button("🏁 Я уже проехал", "submit_result"))
	}

	if deepLinks != nil && deepLinks.Conditions != "" {
		builder.AddRow(ButtonURL("‼️ Условия участия", deepLinks.Conditions))
	} else {
		builder.AddRow(Button("‼️ Условия участия", "event_conditions"))
	}

	if miniappURL != "" {
		builder.AddRow(ButtonWebApp("🏆 Призовой фонд", miniappURL))
	}

	return builder.Build()
}

// PublicMenu создаёт меню для публичного чата.
func PublicMenu(miniappURL, registerLink, conditionsLink string) models.InlineKeyboardMarkup {
	builder := NewBuilder()

	if registerLink != "" {
		builder.AddRow(ButtonURL("✅ Принять участие", registerLink))
	} else {
		builder.AddRow(Button("✅ Принять участие", "register"))
	}

	if miniappURL != "" {
		builder.AddRow(ButtonWebApp("🏆 Призовой фонд", miniappURL))
	}

	if conditionsLink != "" {
		builder.AddRow(ButtonURL("‼️ Условия участия", conditionsLink))
	} else {
		builder.AddRow(Button("‼️ Условия участия", "event_conditions"))
	}

	return builder.Build()
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

// RegistrationConsentMenu создаёт меню согласия с условиями участия.
func RegistrationConsentMenu() models.InlineKeyboardMarkup {
	return NewBuilder().
		AddRow(
			Button("✅ Согласен", "registration_accept_conditions"),
			Button("❌ Отказываюсь", "registration_decline_conditions"),
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

// GiftDraftMenu создаёт актуальную клавиатуру черновика подарка.
func GiftDraftMenu(hasDescription bool) models.InlineKeyboardMarkup {
	builder := NewBuilder()
	if hasDescription {
		builder.AddRow(Button("✅ Готово", "finish_gift"))
	}

	builder.AddRow(Button("🔄 Заполнить заново", "restart_gift"))
	builder.AddRow(Button("❌ Отмена", "cancel"))

	return builder.Build()
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
