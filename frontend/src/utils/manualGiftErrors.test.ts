import { describe, expect, it } from 'vitest';
import { ApiError } from '../api/client';
import { getManualGiftErrorMessage } from './manualGiftErrors';

describe('manual gift error messages', () => {
  it('explains when a Telegram user does not exist in the database', () => {
    const message = getManualGiftErrorMessage(
      new ApiError(404, 'Not Found', { message: 'user not found' }),
      999999
    );

    expect(message).toBe(
      'Пользователь с Telegram ID 999999 не найден. Сначала пользователь должен написать боту или быть зарегистрирован в базе.'
    );
  });

  it('points to the active event instead of manual event selection when the event is missing', () => {
    const message = getManualGiftErrorMessage(
      new ApiError(404, 'Not Found', { message: 'event not found' }),
      1234
    );

    expect(message).toBe(
      'Активное событие не найдено. Обновите страницу — приз добавляется к активному событию.'
    );
  });
});
