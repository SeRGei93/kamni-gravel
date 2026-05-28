'use client';

import React, { useState } from 'react';
import Label from '../form/Label';
import TextArea from '../form/input/TextArea';
import Button from '../ui/button/Button';
import type { EventTelegramTexts } from '@/types';

interface EventTelegramTextsFormProps {
  texts?: Partial<EventTelegramTexts>;
  onSubmit: (texts: EventTelegramTexts) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
}

const DEFAULT_TELEGRAM_TEXTS: EventTelegramTexts = {
  gift_gender_step: `🎁 Добавление приза

Шаг 1/4: Выберите пол участника`,
  gift_bike_step: `🎁 Добавление приза

Шаг 2/4: Выберите тип велосипеда`,
  gift_description_step: `🎁 Добавление приза

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
  gift_photo_step: `🎁 Добавление приза

Шаг 4/4: Описание сохранено. Отправьте фото приза следующим сообщением в поле ввода ниже (можно несколько) или нажмите "Готово" в последнем сообщении снизу.

Фото можно отправлять до и после описания. Подписи к фото на этом шаге не заменяют сохранённое описание.

После каждого сообщения используйте кнопки в последнем сообщении снизу.`,
  gift_photo_added:
    'Фото добавлено! Всего фото: {photo_count}. Отправьте ещё фото в поле ввода ниже или нажмите "Готово" в последнем сообщении снизу.',
  gift_draft: `{step_text}

Черновик приза:
• Пол участника: {gender}
• Тип велосипеда: {bike_type}
• Описание: {description_status}
• Фото: {photo_count}

{action_hint}
После каждого сообщения используйте кнопки в последнем сообщении снизу.`,
  gift_draft_value_missing: 'не выбран',
  gift_draft_description_missing: 'нужно отправить',
  gift_draft_description_added: 'добавлено',
  gift_draft_action_description:
    'Отправьте описание текстом или фото с подписью. Фото без подписи прикрепится к черновику, но описание всё равно нужно будет отправить.',
  gift_draft_action_photo:
    'Можно отправить ещё фото. Когда всё готово, нажмите «Готово» в последнем сообщении снизу.',
  gift_preview: `🎁 Проверьте приз перед отправкой

📋 Детали приза:
• Пол участника: {gender}
• Тип велосипеда: {bike_type}
• Описание: {description}
• Фото: {photo_count}

Если всё верно, подтвердите отправку. После подтверждения приз попадёт на проверку администратору.`,
  gift_confirmation_prompt:
    'Приз уже заполнен. Подтвердите отправку кнопками ниже или отмените добавление.',
  gift_success: `✅ Приз успешно добавлен в призовой фонд!

📋 Детали приза:
• Пол участника: {gender}
• Тип велосипеда: {bike_type}
• Описание: {description}{photo_line}

🙏 Огромное спасибо за ваш вклад!
Вы делаете наше мероприятие ещё лучше! 🎁✨

После проверки администратором приз сможет участвовать в распределении призов.`,
  gift_cancelled: 'Добавление приза отменено.',
  gift_session_error:
    'Ошибка: данные приза не найдены или повреждены. Начните добавление приза заново.',
  gift_callback_continue: 'Продолжите добавление',
  gift_callback_add_description: 'Добавьте описание',
  gift_callback_review_draft: 'Сначала проверьте приз',
  gift_callback_confirm: 'Подтвердите приз',
  gift_callback_open_menu: 'Сначала откройте меню',
  result_prompt: `🏁 Отправка результата

Отправьте ссылку на вашу активность Strava.

Пример:
https://www.strava.com/activities/123456789`,
  result_invalid_link:
    'Отправьте ссылку на активность Strava.\nПример:\nhttps://www.strava.com/activities/123456789',
  result_success:
    '✅ Результат принят!\n\nСсылка: {result_link}\n\nВаше время будет обработано администратором. Следите за обновлениями! 🏆',
  result_already_sent: 'Вы уже отправили результат!',
  result_not_registered:
    'Вы не зарегистрированы на это событие. Сначала пройдите регистрацию.',
  result_start_missing:
    'Подача результата пока недоступна: время старта события не настроено. Обратитесь к организатору.',
  result_not_started:
    'Подача результата откроется после старта события: {start_time} (Минск UTC+3).',
};

const TEXT_FIELDS: Array<{
  key: keyof EventTelegramTexts;
  label: string;
  rows: number;
  wide?: boolean;
}> = [
  { key: 'gift_gender_step', label: 'Шаг 1: выбор пола', rows: 3 },
  { key: 'gift_bike_step', label: 'Шаг 2: выбор велосипеда', rows: 3 },
  { key: 'gift_description_step', label: 'Шаг 3: описание приза', rows: 12, wide: true },
  { key: 'gift_photo_step', label: 'Шаг 4: фото приза', rows: 5 },
  { key: 'gift_photo_added', label: 'После добавления фото', rows: 3 },
  { key: 'gift_draft', label: 'Черновик приза', rows: 11, wide: true },
  { key: 'gift_draft_value_missing', label: 'Черновик: значение не выбрано', rows: 2 },
  { key: 'gift_draft_description_missing', label: 'Черновик: описание не заполнено', rows: 2 },
  { key: 'gift_draft_description_added', label: 'Черновик: описание заполнено', rows: 2 },
  { key: 'gift_draft_action_description', label: 'Черновик: подсказка описания', rows: 3 },
  { key: 'gift_draft_action_photo', label: 'Черновик: подсказка фото', rows: 3 },
  { key: 'gift_preview', label: 'Предпросмотр перед отправкой', rows: 9, wide: true },
  { key: 'gift_confirmation_prompt', label: 'Повторное подтверждение', rows: 3 },
  { key: 'gift_success', label: 'Успешное добавление', rows: 11, wide: true },
  { key: 'gift_cancelled', label: 'Отмена', rows: 2 },
  { key: 'gift_session_error', label: 'Ошибка сессии', rows: 2 },
  { key: 'gift_callback_continue', label: 'Callback: продолжить', rows: 2 },
  { key: 'gift_callback_add_description', label: 'Callback: добавить описание', rows: 2 },
  { key: 'gift_callback_review_draft', label: 'Callback: проверить приз', rows: 2 },
  { key: 'gift_callback_confirm', label: 'Callback: подтвердить приз', rows: 2 },
  { key: 'gift_callback_open_menu', label: 'Callback: открыть меню', rows: 2 },
  { key: 'result_prompt', label: 'Результат: запрос ссылки', rows: 6, wide: true },
  { key: 'result_invalid_link', label: 'Результат: неверная ссылка', rows: 3 },
  { key: 'result_success', label: 'Результат: принято', rows: 6, wide: true },
  { key: 'result_already_sent', label: 'Результат: уже отправлен', rows: 2 },
  { key: 'result_not_registered', label: 'Результат: участник не зарегистрирован', rows: 3 },
  { key: 'result_start_missing', label: 'Результат: старт не настроен', rows: 3 },
  { key: 'result_not_started', label: 'Результат: старт ещё не наступил', rows: 3 },
];

function normalizeTexts(texts?: Partial<EventTelegramTexts>): EventTelegramTexts {
  return {
    ...DEFAULT_TELEGRAM_TEXTS,
    ...texts,
  };
}

export default function EventTelegramTextsForm({
  texts,
  onSubmit,
  onCancel,
  isLoading = false,
}: EventTelegramTextsFormProps) {
  const [formTexts, setFormTexts] = useState<EventTelegramTexts>(normalizeTexts(texts));

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    await onSubmit(formTexts);
  };

  const updateText = (key: keyof EventTelegramTexts, value: string) => {
    setFormTexts((current) => ({
      ...current,
      [key]: value,
    }));
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="rounded-lg border border-gray-200 bg-white p-4 dark:border-white/[0.08] dark:bg-white/[0.03]">
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          {TEXT_FIELDS.map((field) => (
            <div key={field.key} className={field.wide ? 'xl:col-span-2' : undefined}>
              <Label>{field.label}</Label>
              <TextArea
                value={formTexts[field.key]}
                onChange={(value) => updateText(field.key, value)}
                rows={field.rows}
                disabled={isLoading}
              />
            </div>
          ))}
        </div>
      </div>

      <div className="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-white/[0.08] dark:bg-white/[0.03]">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Плейсхолдеры для призов и результатов:{' '}
          <span className="font-mono">{'{gender}'}</span>,{' '}
          <span className="font-mono">{'{bike_type}'}</span>,{' '}
          <span className="font-mono">{'{description}'}</span>,{' '}
          <span className="font-mono">{'{photo_count}'}</span>,{' '}
          <span className="font-mono">{'{photo_line}'}</span>,{' '}
          <span className="font-mono">{'{result_link}'}</span>,{' '}
          <span className="font-mono">{'{start_time}'}</span>.
        </p>
      </div>

      <div className="flex flex-wrap items-center justify-end gap-3">
        <Button
          type="button"
          variant="outline"
          onClick={() => setFormTexts(DEFAULT_TELEGRAM_TEXTS)}
          disabled={isLoading}
        >
          Вернуть тексты по умолчанию
        </Button>
        <Button type="button" variant="outline" onClick={onCancel} disabled={isLoading}>
          Назад
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading ? 'Сохранение...' : 'Сохранить тексты'}
        </Button>
      </div>
    </form>
  );
}
