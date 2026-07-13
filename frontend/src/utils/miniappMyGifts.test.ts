import { describe, expect, it } from 'vitest';
import {
  isRecipientSelectionChanged,
  miniappGiftModeLabel,
  miniappGiftMutationErrorMessage,
  miniappGiftRecipientLabel,
  miniappGiftReviewLabel,
} from './miniappMyGifts';
import { MiniappApiError } from '@/api/miniapp';
import type { ManualGift } from '@/types';

const manualGift: ManualGift = {
  id: 1,
  event_id: 7,
  description: 'Bottle',
  review_status: 'pending_review',
  manual_distribution: true,
  created_at: '2026-07-11T00:00:00Z',
};

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
  });
});
