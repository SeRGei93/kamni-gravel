package entity

import "time"

// Event представляет велогонку/мероприятие
type Event struct {
	ID            uint
	Name          string
	Description   string
	Active        bool
	StartDate     *time.Time
	EndDate       *time.Time
	GPXFilePath   string
	TelegramTexts EventTelegramTexts
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// EventTelegramTexts содержит редактируемые тексты Telegram-сценариев события.
type EventTelegramTexts struct {
	GiftGenderStep      string `json:"gift_gender_step"`
	GiftBikeStep        string `json:"gift_bike_step"`
	GiftDescriptionStep string `json:"gift_description_step"`
	GiftPhotoStep       string `json:"gift_photo_step"`
	GiftPhotoAdded      string `json:"gift_photo_added"`
	GiftPreview         string `json:"gift_preview"`
	GiftSuccess         string `json:"gift_success"`
	GiftCancelled       string `json:"gift_cancelled"`
	GiftSessionError    string `json:"gift_session_error"`
}

// DefaultEventTelegramTexts возвращает стандартные тексты Telegram для события.
func DefaultEventTelegramTexts() EventTelegramTexts {
	return EventTelegramTexts{
		GiftGenderStep: `🎁 Добавление приза

Шаг 1/4: Выберите пол участника`,
		GiftBikeStep: `🎁 Добавление приза

Шаг 2/4: Выберите тип велосипеда`,
		GiftDescriptionStep: `🎁 Добавление приза

Шаг 3/4: Напишите описание приза следующим сообщением в поле ввода ниже.

Укажите за что этот приз (номинацию) и что именно вы дарите.

Примеры:
• Самый быстрый на гревеле - Парафиновая смазка Мамкина забота
• Выпито больше всего пива на маршруте - Упаковка кислых червячков
• Лучшее фото у камней - Топкеп Спаси и сохрани
• Последнее место в общем зачете - Проездной на общественный транспорт
• Бутылка водки "Налибоки" за первое место МТБ
• Первое место абсолют - Кирпич`,
		GiftPhotoStep: `🎁 Добавление приза

Шаг 4/4: Отправьте фото приза следующим сообщением в поле ввода ниже (можно несколько).

Когда закончите, нажмите "Завершить" или "Пропустить", если фото нет.`,
		GiftPhotoAdded: `Фото добавлено! Всего фото: {photo_count}. Отправьте ещё фото в поле ввода ниже или нажмите "Завершить".`,
		GiftPreview: `🎁 Проверьте приз перед отправкой

📋 Детали приза:
• Пол участника: {gender}
• Тип велосипеда: {bike_type}
• Описание: {description}
• Фото: {photo_count}

Если всё верно, подтвердите отправку. После подтверждения приз попадёт на проверку администратору.`,
		GiftSuccess: `✅ Приз успешно добавлен в призовой фонд!

📋 Детали приза:
• Пол участника: {gender}
• Тип велосипеда: {bike_type}
• Описание: {description}{photo_line}

🙏 Огромное спасибо за ваш вклад!
Вы делаете наше мероприятие ещё лучше! 🎁✨

После проверки администратором приз сможет участвовать в распределении призов.`,
		GiftCancelled:    "Добавление приза отменено.",
		GiftSessionError: "Ошибка: данные приза не найдены или повреждены. Начните добавление приза заново.",
	}
}

// NormalizeEventTelegramTexts заполняет пустые тексты значениями по умолчанию.
func NormalizeEventTelegramTexts(texts EventTelegramTexts) EventTelegramTexts {
	defaults := DefaultEventTelegramTexts()
	if texts.GiftGenderStep == "" {
		texts.GiftGenderStep = defaults.GiftGenderStep
	}
	if texts.GiftBikeStep == "" {
		texts.GiftBikeStep = defaults.GiftBikeStep
	}
	if texts.GiftDescriptionStep == "" {
		texts.GiftDescriptionStep = defaults.GiftDescriptionStep
	}
	if texts.GiftPhotoStep == "" {
		texts.GiftPhotoStep = defaults.GiftPhotoStep
	}
	if texts.GiftPhotoAdded == "" {
		texts.GiftPhotoAdded = defaults.GiftPhotoAdded
	}
	if texts.GiftPreview == "" {
		texts.GiftPreview = defaults.GiftPreview
	}
	if texts.GiftSuccess == "" {
		texts.GiftSuccess = defaults.GiftSuccess
	}
	if texts.GiftCancelled == "" {
		texts.GiftCancelled = defaults.GiftCancelled
	}
	if texts.GiftSessionError == "" {
		texts.GiftSessionError = defaults.GiftSessionError
	}
	return texts
}

// IsActive проверяет, активно ли событие
func (e *Event) IsActive() bool {
	return e.Active
}

// IsOngoing проверяет, идёт ли событие сейчас
func (e *Event) IsOngoing() bool {
	now := time.Now()
	if e.StartDate != nil && now.Before(*e.StartDate) {
		return false
	}
	if e.EndDate != nil && now.After(*e.EndDate) {
		return false
	}
	return e.Active
}
