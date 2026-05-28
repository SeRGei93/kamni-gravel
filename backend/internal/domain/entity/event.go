package entity

import (
	"strings"
	"time"

	"gravel_bot/internal/domain/valueobject"
)

// Event представляет велогонку/мероприятие
type Event struct {
	ID                      uint
	Name                    string
	Description             string
	ParticipationConditions string
	Active                  bool
	StartDate               *time.Time
	EndDate                 *time.Time
	GPXFilePath             string
	TelegramTexts           EventTelegramTexts
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

const defaultEventParticipationConditions = `УСЛОВИЯ УЧАСТИЯ (ДИСКЛЕЙМЕР) ‼️

Для обычного человека гравийная поездка на 200-300 км не является лёгкой прогулкой и требует хорошей физической и моральной подготовки, планирования питания и питья, а также наличие всего необходимого для ремонта велосипеда, оказания медпомощи и эвакуации себя.

Участие в КАМНЯХ означает полное принятие следующих условий:

1. Участниками КАМНЕЙ 200 признаются люди старше 18 лет, кто прошёл регистрацию, сделал взнос в призовой фонд и проехал хотя бы часть маршрута, что подтверждается Страва-ссылкой. Таковые попадают в лидерборд, при этом приоритет имеют участники, проехавшие маршрут целиком.

2. Самостоятельное обеспечение питанием и питьём 🍼🍔: участник обязан самостоятельно обеспечивать себя питанием и питьём на протяжении всей поездки. Рекомендуется употреблять 50-100 г углеводов каждый час (батончики, гели, фрукты, еда, изотон), не дожидаясь чувства голода, а также небольшое кол-во белков (орехи, сыр, мясо и т.п.). Питьё небольшими глотками каждые 15-20 минут, не дожидаясь жажды. Общий литраж индивидуален, от 0,5 до 1 литра в час, в идеале изотонические напитки (водный раствор с полезными минералами, углеводами: соль, мёд, сок и т.п.). Всегда иметь запас воды и питания, пополнять его при первой возможности.

3. Безопасность 🤌: участник несёт полную ответственность за свою безопасность. Обязателен шлем, исправный велосипед, фонарики, запас денег, внимание к самочувствию. Выход на маршрут только в здоровом состоянии. Рекомендуется иметь при себе аптечку первой помощи: бинты, пластырь, дезинфицирующий раствор, обезболивающие препараты. При плохом самочувствии — остановить прохождение дистанции, сойти с дороги в безопасное место, привести себя в чувство, а при невозможности – сойти с дистанции. При жаре не допускать перегрева, охлаждаться водой, тенью.

4. Выезд на дороги, соблюдение ПДД 🚗 и законов РБ: участник несёт полную ответственность за соблюдение правил дорожного движения, законов Республики Беларусь, и безопасность при выезде на дороги общего пользования.

5. Проблемы на маршруте 🧚‍♀️: при возникновении любого рода проблем на маршруте участник надеется на себя. Помощь другим участникам приветствуется, однако, надеяться на неё не стоит. Обязательно иметь при себе заряженный телефон. Телефоны экстренных служб: 103 скорая медпомощь, 112 МЧС, 102 Милиция. При возникновении непреодолимых трудностей пишите в данный чат. Возможно, кто-то сможет вам помочь.

6. Сход с дистанции ⛔️: в случае схода с дистанции участник самостоятельно добирается до дома. Транспорт не предоставляется. Если вы осознали, что неспособны доехать маршрут до конца, вызывайте эвакуацию (друзья, такси, попутки) либо двигайтесь в направлении ближайшей ЖД-станции.

7. Ремонт и техобслуживание 🚴‍♀️: участник выходит на маршрут на полностью исправном велосипеде с работающей тормозной системой, имеет при себе необходимые инструменты и запчасти для ремонта. Особенно рекомендуется иметь несколько запасных камер, латки, червяки для бескамерки, т.к. вероятность проколов высока.

8. Навигация и маршрут 🏞: участники обязаны самостоятельно ориентироваться на маршруте и иметь при себе навигационные средства с достаточным зарядом до завершения маршрута. Разметка на треке отсутствует.

9. Неопытным велосипедистам 🍬: тем, кто ещё не ездил в длительные поездки, рекомендуется прохождение трека в компании. Так веселее, безопаснее и проще преодолевать трудности.

10. Риски и ответственность 🤌: участие в гонке сопряжено с определенными рисками, включая травмы и аварии. Участники принимают участие на свой страх и риск, создатели маршрута не несут ответственности за возможные инциденты, любые происшествия, связанные с участием в заезде. Принимая участие в заезде, каждый участник подтверждает свое согласие с данными условиями и принимает на себя все связанные риски.`

// DefaultEventParticipationConditions возвращает стандартные условия участия.
func DefaultEventParticipationConditions() string {
	return defaultEventParticipationConditions
}

// NormalizeEventParticipationConditions заполняет пустые условия участия значением по умолчанию.
func NormalizeEventParticipationConditions(conditions string) string {
	if strings.TrimSpace(conditions) == "" {
		return DefaultEventParticipationConditions()
	}
	return conditions
}

// EventTelegramTexts содержит редактируемые тексты Telegram-сценариев события.
type EventTelegramTexts struct {
	GiftGenderStep              string `json:"gift_gender_step"`
	GiftBikeStep                string `json:"gift_bike_step"`
	GiftDescriptionStep         string `json:"gift_description_step"`
	GiftPhotoStep               string `json:"gift_photo_step"`
	GiftPhotoAdded              string `json:"gift_photo_added"`
	GiftDraft                   string `json:"gift_draft"`
	GiftDraftValueMissing       string `json:"gift_draft_value_missing"`
	GiftDraftDescriptionMissing string `json:"gift_draft_description_missing"`
	GiftDraftDescriptionAdded   string `json:"gift_draft_description_added"`
	GiftDraftActionDescription  string `json:"gift_draft_action_description"`
	GiftDraftActionPhoto        string `json:"gift_draft_action_photo"`
	GiftPreview                 string `json:"gift_preview"`
	GiftConfirmationPrompt      string `json:"gift_confirmation_prompt"`
	GiftSuccess                 string `json:"gift_success"`
	GiftCancelled               string `json:"gift_cancelled"`
	GiftSessionError            string `json:"gift_session_error"`
	GiftCallbackContinue        string `json:"gift_callback_continue"`
	GiftCallbackAddDescription  string `json:"gift_callback_add_description"`
	GiftCallbackReviewDraft     string `json:"gift_callback_review_draft"`
	GiftCallbackConfirm         string `json:"gift_callback_confirm"`
	GiftCallbackOpenMenu        string `json:"gift_callback_open_menu"`
	ResultPrompt                string `json:"result_prompt"`
	ResultInvalidLink           string `json:"result_invalid_link"`
	ResultSuccess               string `json:"result_success"`
	ResultAlreadySent           string `json:"result_already_sent"`
	ResultNotRegistered         string `json:"result_not_registered"`
	ResultStartMissing          string `json:"result_start_missing"`
	ResultNotStarted            string `json:"result_not_started"`
}

// DefaultEventTelegramTexts возвращает стандартные тексты Telegram для события.
func DefaultEventTelegramTexts() EventTelegramTexts {
	return EventTelegramTexts{
		GiftGenderStep: `🎁 Добавление приза

Шаг 1/4: Выберите пол участника`,
		GiftBikeStep: `🎁 Добавление приза

Шаг 2/4: Выберите тип велосипеда`,
		GiftDescriptionStep: `🎁 Добавление приза

Шаг 3/4: Отправьте описание приза текстом или пришлите фото с подписью.

Фото можно отправить уже сейчас: оно прикрепится к черновику. Если фото без подписи, описание всё равно нужно будет отправить отдельным сообщением.

Укажите за что этот приз (номинацию) и что именно вы дарите.

После каждого сообщения используйте кнопки в последнем сообщении снизу.

Примеры:
• Самый быстрый на гревеле - Парафиновая смазка Мамкина забота
• Выпито больше всего пива на маршруте - Упаковка кислых червячков
• Лучшее фото у камней - Топкеп Спаси и сохрани
• Последнее место в общем зачете - Проездной на общественный транспорт
• Бутылка водки "Налибоки" за первое место МТБ
• Первое место абсолют - Кирпич`,
		GiftPhotoStep: `🎁 Добавление приза

Шаг 4/4: Описание сохранено. Отправьте фото приза следующим сообщением в поле ввода ниже (можно несколько) или нажмите "Готово" в последнем сообщении снизу.

Фото можно отправлять до и после описания. Подписи к фото на этом шаге не заменяют сохранённое описание.

После каждого сообщения используйте кнопки в последнем сообщении снизу.`,
		GiftPhotoAdded: `Фото добавлено! Всего фото: {photo_count}. Отправьте ещё фото в поле ввода ниже или нажмите "Готово" в последнем сообщении снизу.`,
		GiftDraft: `{step_text}

Черновик приза:
• Пол участника: {gender}
• Тип велосипеда: {bike_type}
• Описание: {description_status}
• Фото: {photo_count}

{action_hint}
После каждого сообщения используйте кнопки в последнем сообщении снизу.`,
		GiftDraftValueMissing:       "не выбран",
		GiftDraftDescriptionMissing: "нужно отправить",
		GiftDraftDescriptionAdded:   "добавлено",
		GiftDraftActionDescription:  "Отправьте описание текстом или фото с подписью. Фото без подписи прикрепится к черновику, но описание всё равно нужно будет отправить.",
		GiftDraftActionPhoto:        "Можно отправить ещё фото. Когда всё готово, нажмите «Готово» в последнем сообщении снизу.",
		GiftPreview: `🎁 Проверьте приз перед отправкой

📋 Детали приза:
• Пол участника: {gender}
• Тип велосипеда: {bike_type}
• Описание: {description}
• Фото: {photo_count}

Если всё верно, подтвердите отправку. После подтверждения приз попадёт на проверку администратору.`,
		GiftConfirmationPrompt: "Приз уже заполнен. Подтвердите отправку кнопками ниже или отмените добавление.",
		GiftSuccess: `✅ Приз успешно добавлен в призовой фонд!

📋 Детали приза:
• Пол участника: {gender}
• Тип велосипеда: {bike_type}
• Описание: {description}{photo_line}

🙏 Огромное спасибо за ваш вклад!
Вы делаете наше мероприятие ещё лучше! 🎁✨

После проверки администратором приз сможет участвовать в распределении призов.`,
		GiftCancelled:              "Добавление приза отменено.",
		GiftSessionError:           "Ошибка: данные приза не найдены или повреждены. Начните добавление приза заново.",
		GiftCallbackContinue:       "Продолжите добавление",
		GiftCallbackAddDescription: "Добавьте описание",
		GiftCallbackReviewDraft:    "Сначала проверьте приз",
		GiftCallbackConfirm:        "Подтвердите приз",
		GiftCallbackOpenMenu:       "Сначала откройте меню",
		ResultPrompt: `🏁 Отправка результата

Отправьте ссылку на вашу активность Strava.

Пример:
https://www.strava.com/activities/123456789`,
		ResultInvalidLink:   "Отправьте ссылку на активность Strava.\nПример:\nhttps://www.strava.com/activities/123456789",
		ResultSuccess:       "✅ Результат принят!\n\nСсылка: {result_link}\n\nВаше время будет обработано администратором. Следите за обновлениями! 🏆",
		ResultAlreadySent:   "Вы уже отправили результат!",
		ResultNotRegistered: "Вы не зарегистрированы на это событие. Сначала пройдите регистрацию.",
		ResultStartMissing:  "Подача результата пока недоступна: время старта события не настроено. Обратитесь к организатору.",
		ResultNotStarted:    "Подача результата откроется после старта события: {start_time} (Минск UTC+3).",
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
	if texts.GiftDraft == "" {
		texts.GiftDraft = defaults.GiftDraft
	}
	if texts.GiftDraftValueMissing == "" {
		texts.GiftDraftValueMissing = defaults.GiftDraftValueMissing
	}
	if texts.GiftDraftDescriptionMissing == "" {
		texts.GiftDraftDescriptionMissing = defaults.GiftDraftDescriptionMissing
	}
	if texts.GiftDraftDescriptionAdded == "" {
		texts.GiftDraftDescriptionAdded = defaults.GiftDraftDescriptionAdded
	}
	if texts.GiftDraftActionDescription == "" {
		texts.GiftDraftActionDescription = defaults.GiftDraftActionDescription
	}
	if texts.GiftDraftActionPhoto == "" {
		texts.GiftDraftActionPhoto = defaults.GiftDraftActionPhoto
	}
	if texts.GiftPreview == "" {
		texts.GiftPreview = defaults.GiftPreview
	}
	if texts.GiftConfirmationPrompt == "" {
		texts.GiftConfirmationPrompt = defaults.GiftConfirmationPrompt
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
	if texts.GiftCallbackContinue == "" {
		texts.GiftCallbackContinue = defaults.GiftCallbackContinue
	}
	if texts.GiftCallbackAddDescription == "" {
		texts.GiftCallbackAddDescription = defaults.GiftCallbackAddDescription
	}
	if texts.GiftCallbackReviewDraft == "" {
		texts.GiftCallbackReviewDraft = defaults.GiftCallbackReviewDraft
	}
	if texts.GiftCallbackConfirm == "" {
		texts.GiftCallbackConfirm = defaults.GiftCallbackConfirm
	}
	if texts.GiftCallbackOpenMenu == "" {
		texts.GiftCallbackOpenMenu = defaults.GiftCallbackOpenMenu
	}
	if texts.ResultPrompt == "" {
		texts.ResultPrompt = defaults.ResultPrompt
	}
	if texts.ResultInvalidLink == "" {
		texts.ResultInvalidLink = defaults.ResultInvalidLink
	}
	if texts.ResultSuccess == "" {
		texts.ResultSuccess = defaults.ResultSuccess
	}
	if texts.ResultAlreadySent == "" {
		texts.ResultAlreadySent = defaults.ResultAlreadySent
	}
	if texts.ResultNotRegistered == "" {
		texts.ResultNotRegistered = defaults.ResultNotRegistered
	}
	if texts.ResultStartMissing == "" {
		texts.ResultStartMissing = defaults.ResultStartMissing
	}
	if texts.ResultNotStarted == "" {
		texts.ResultNotStarted = defaults.ResultNotStarted
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

// HasStartedAt проверяет, наступило ли время старта события для подачи результата.
func (e *Event) HasStartedAt(now time.Time) bool {
	if e == nil || e.StartDate == nil {
		return false
	}

	start := e.StartDate.In(valueobject.MinskLocation())
	current := now.In(valueobject.MinskLocation())
	return !current.Before(start)
}

// SubmissionStartTimeInMinsk возвращает время старта подачи результата в Минске UTC+3.
func (e *Event) SubmissionStartTimeInMinsk() (time.Time, bool) {
	if e == nil || e.StartDate == nil {
		return time.Time{}, false
	}

	return e.StartDate.In(valueobject.MinskLocation()), true
}
