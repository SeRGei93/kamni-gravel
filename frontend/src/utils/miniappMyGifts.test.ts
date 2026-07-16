import { describe, expect, it } from 'vitest';
import {
  isRecipientSelectionChanged,
  miniappGiftModeLabel,
  miniappGiftMutationErrorMessage,
  miniappGiftRecipientLabel,
  miniappGiftReviewLabel,
  MiniappMyGiftsRefreshError,
  updateManualGiftRecipient,
} from './miniappMyGifts';
import { MiniappApiError } from '@/api/miniapp';
import type { ManualGift, MiniappParticipantOption } from '@/types';

const manualGift: ManualGift = {
  id: 1,
  event_id: 7,
  description: 'Bottle',
  review_status: 'pending_review',
  manual_distribution: true,
  created_at: '2026-07-11T00:00:00Z',
};

const participants: MiniappParticipantOption[] = [
  { id: 12, display_name: 'Alex', username: 'alex', status: 'active', has_prize: false },
];

describe('Mini App My Prizes helpers', () => {
  it('formats review, mode, and unassigned recipient states', () => {
    expect(miniappGiftReviewLabel(manualGift)).toBe('На проверке');
    expect(miniappGiftModeLabel(manualGift)).toBe('Ручное распределение');
    expect(miniappGiftRecipientLabel(manualGift)).toBe('Получатель пока не выбран');
    expect(miniappGiftModeLabel({ ...manualGift, manual_distribution: false })).toBe(
      'Автоматическое распределение'
    );
  });

  it('detects selection changes including an explicit clear', () => {
    const assignedGift: ManualGift = {
      ...manualGift,
      recipient: { id: 12, display_name: 'Alex', status: 'active' },
    };
    expect(isRecipientSelectionChanged(assignedGift, 12)).toBe(false);
    expect(isRecipientSelectionChanged(assignedGift, null)).toBe(true);
  });

  it('maps stale and conflicting mutations to actionable messages', () => {
    expect(miniappGiftMutationErrorMessage(new MiniappApiError(404, 'Not Found'))).toContain('недоступен');
    expect(miniappGiftMutationErrorMessage(new MiniappApiError(409, 'Conflict'))).toContain('завершить заезд');
    expect(
      miniappGiftMutationErrorMessage(
        new MiniappApiError(409, 'Conflict'),
        'random_unawarded'
      )
    ).toContain('без награды');
    expect(
      miniappGiftMutationErrorMessage(
        new MiniappApiError(409, 'Conflict'),
        'random_including_awarded'
      )
    ).not.toContain('без награды');
    expect(miniappGiftMutationErrorMessage(new MiniappMyGiftsRefreshError())).toContain('назначен');
  });

  it('updates the cached recipient immediately after a successful manual assignment', () => {
    const gifts = [{ ...manualGift, recipient: { id: 7, display_name: 'Maria', status: 'dnf' as const } }];

    const updated = updateManualGiftRecipient(gifts, participants, 1, 12);

    expect(updated).toEqual([
      {
        ...manualGift,
        recipient: { id: 12, display_name: 'Alex', username: 'alex', status: 'active' },
      },
    ]);
    expect(updated).not.toBe(gifts);
    expect(gifts[0].recipient?.id).toBe(7);
  });

  it('clears the cached recipient and leaves it unchanged for an unknown participant', () => {
    const assignedGifts = [{ ...manualGift, recipient: { id: 12, display_name: 'Alex', status: 'active' as const } }];

    expect(updateManualGiftRecipient(assignedGifts, participants, 1, null)[0].recipient).toBeUndefined();
    expect(updateManualGiftRecipient(assignedGifts, participants, 1, 999)).toBe(assignedGifts);
  });
});
