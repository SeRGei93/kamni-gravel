import { MiniappApiError } from '@/api/miniapp';
import type { ManualGift } from '@/types';

export function miniappGiftReviewLabel(gift: ManualGift): string {
  return gift.review_status === 'approved' ? 'Проверен' : 'На проверке';
}

export function miniappGiftModeLabel(gift: ManualGift): string {
  return gift.manual_distribution
    ? 'Ручное распределение'
    : 'Автоматическое распределение';
}

export function miniappGiftRecipientLabel(gift: ManualGift): string {
  return gift.recipient?.display_name ?? 'Получатель пока не выбран';
}

export function miniappGiftMutationErrorMessage(error: unknown): string {
  if (!(error instanceof MiniappApiError)) {
    return 'Не удалось сохранить получателя. Проверьте соединение и повторите попытку.';
  }

  if (error.status === 404) {
    return 'Приз или участник больше недоступен. Обновите список и выберите получателя снова.';
  }
  if (error.status === 409) {
	return 'Этот приз нельзя назначить вручную, участник относится к другому событию или все участники уже получили награды.';
  }
  if (error.status === 400) {
    return 'Проверьте выбранного получателя и повторите попытку.';
  }
  return 'Не удалось сохранить получателя. Повторите попытку.';
}

export function isRecipientSelectionChanged(
  gift: ManualGift,
  nextRecipientID: number | null
): boolean {
  return (gift.recipient?.id ?? null) !== nextRecipientID;
}
