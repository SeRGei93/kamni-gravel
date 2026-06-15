import { ApiError } from '../api/client';

export function getManualGiftErrorMessage(
  error: unknown,
  telegramUserId: number
): string {
  if (!(error instanceof ApiError)) {
    return 'Не удалось добавить приз. Проверьте соединение и попробуйте ещё раз.';
  }

  const backendMessage =
    typeof error.data?.message === 'string' ? error.data.message : '';

  if (error.status === 404 && backendMessage.includes('user not found')) {
    return `Пользователь с Telegram ID ${telegramUserId} не найден. Сначала пользователь должен написать боту или быть зарегистрирован в базе.`;
  }

  if (error.status === 404 && backendMessage.includes('event not found')) {
    return 'Событие для добавления приза не найдено. Обновите страницу и выберите событие заново.';
  }

  if (error.status === 403 && backendMessage.includes('blacklisted')) {
    return `Пользователь с Telegram ID ${telegramUserId} находится в blacklist. Приз от его имени добавить нельзя.`;
  }

  if (error.status === 400) {
    return backendMessage || 'Проверьте обязательные поля приза.';
  }

  return backendMessage || 'Не удалось добавить приз. Попробуйте ещё раз.';
}
