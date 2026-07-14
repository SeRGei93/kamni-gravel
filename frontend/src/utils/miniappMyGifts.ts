import { MiniappApiError } from '@/api/miniapp';
import type { ManualGift, ManualGiftRecipient, MiniappParticipantOption } from '@/types';

export class MiniappMyGiftsRefreshError extends Error {
  constructor() {
    super('Recipient was assigned, but the My Prizes list could not be refreshed');
    this.name = 'MiniappMyGiftsRefreshError';
  }
}

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
  if (error instanceof MiniappMyGiftsRefreshError) {
    return 'Получатель назначен, но список не удалось обновить. Обновите страницу.';
  }
  if (!(error instanceof MiniappApiError)) {
    return 'Не удалось сохранить получателя. Проверьте соединение и повторите попытку.';
  }

  if (error.status === 404) {
    return 'Приз или участник больше недоступен. Обновите список и выберите получателя снова.';
  }
  if (error.status === 409) {
    return 'Этот приз нельзя назначить вручную: получатель должен завершить заезд или иметь статус «сошёл с дистанции». Также проверьте событие и наличие участников без награды.';
  }
  if (error.status === 400) {
    return 'Проверьте выбранного получателя и повторите попытку.';
  }
  return 'Не удалось сохранить получателя. Повторите попытку.';
}

export function updateManualGiftRecipient(
  gifts: ManualGift[],
  participants: MiniappParticipantOption[],
  giftID: number,
  recipientID: number | null
): ManualGift[] {
  const giftIndex = gifts.findIndex((gift) => gift.id === giftID);
  if (giftIndex < 0) {
    return gifts;
  }

  const participant = recipientID === null
    ? null
    : participants.find((option) => option.id === recipientID);
  if (recipientID !== null && !participant) {
    return gifts;
  }

  const recipient: ManualGiftRecipient | undefined = participant
    ? {
        id: participant.id,
        display_name: participant.display_name,
        username: participant.username,
        status: participant.status,
      }
    : undefined;
  const updatedGift = { ...gifts[giftIndex], recipient };

  return [
    ...gifts.slice(0, giftIndex),
    updatedGift,
    ...gifts.slice(giftIndex + 1),
  ];
}

export function isRecipientSelectionChanged(
  gift: ManualGift,
  nextRecipientID: number | null
): boolean {
  return (gift.recipient?.id ?? null) !== nextRecipientID;
}
