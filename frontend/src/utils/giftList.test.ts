import { describe, expect, it } from 'vitest';
import type { Gift, ManualGift } from '@/types';
import {
  buildFilteredGiftList,
  isCurrentGiftListRequest,
  paginateGifts,
  shouldSettleGiftListRequest,
} from './giftList';

const automaticGift: Gift = {
  id: 1,
  user_id: 11,
  event_id: 7,
  description: 'Automatic gift',
  review_status: 'approved',
  created_at: '2026-07-16T00:00:00Z',
};

const pendingManualGift: Gift = {
  ...automaticGift,
  id: 2,
  description: 'Pending manual gift',
  review_status: 'pending_review',
  manual_distribution: true,
};

const manualGifts: ManualGift[] = [
  {
    id: 3,
    event_id: 7,
    description: 'Assigned manual gift',
    review_status: 'approved',
    manual_distribution: true,
    recipient: { id: 22, display_name: 'Alex', status: 'active' },
    created_at: '2026-07-16T00:00:00Z',
  },
];

describe('gift list helpers', () => {
  it('enriches gifts before applying the manual-unassigned filter', () => {
    const assignedManualGift: Gift = {
      ...automaticGift,
      id: 3,
      description: 'Assigned manual gift',
      manual_distribution: true,
    };

    const gifts = buildFilteredGiftList({
      gifts: [automaticGift, pendingManualGift, assignedManualGift],
      manualGifts,
      distributionFilter: 'manual_unassigned',
      assignedGiftIds: new Set([automaticGift.id]),
    });

    expect(gifts.map((gift) => gift.id)).toEqual([2]);
  });

  it('paginates a client-filtered list and preserves the full-list mode', () => {
    const gifts = [automaticGift, pendingManualGift, { ...automaticGift, id: 3 }];

    expect(paginateGifts(gifts, 2, 2).map((gift) => gift.id)).toEqual([3]);
    expect(paginateGifts(gifts, 1, 'all')).toBe(gifts);
  });

  it('rejects stale or unmounted list requests', () => {
    expect(isCurrentGiftListRequest(2, 3)).toBe(false);
    expect(isCurrentGiftListRequest(3, 3)).toBe(true);
    expect(shouldSettleGiftListRequest(3, 3, true)).toBe(true);
    expect(shouldSettleGiftListRequest(3, 4, true)).toBe(false);
    expect(shouldSettleGiftListRequest(3, 3, false)).toBe(false);
  });
});
