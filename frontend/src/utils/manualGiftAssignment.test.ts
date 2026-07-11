import { describe, expect, it } from 'vitest';
import {
  attachManualGiftAssignments,
  buildManualGiftUpdate,
  formatManualRecipientSearchLabel,
  getManualGiftStatus,
} from './manualGiftAssignment';
import type { Gift, ManualGift } from '@/types';

const automaticGift: Gift = {
  id: 1,
  user_id: 10,
  event_id: 7,
  description: 'Bottle',
  review_status: 'approved',
  created_at: '2026-07-11T00:00:00Z',
};

describe('manual gift assignment helpers', () => {
  it('sends an explicit clear when manual distribution is disabled', () => {
    expect(buildManualGiftUpdate(false, 12)).toEqual({
      manual_distribution: false,
      manual_recipient_participant_id: null,
    });
    expect(buildManualGiftUpdate(true, null)).toEqual({
      manual_distribution: true,
      manual_recipient_participant_id: null,
    });
  });

  it('keeps protected recipient data associated only with its gift', () => {
    const manualGift: ManualGift = {
      id: 2,
      event_id: 7,
      description: 'Cap',
      review_status: 'approved',
      manual_distribution: true,
      recipient: { id: 12, display_name: 'Alex', status: 'active' },
      created_at: '2026-07-11T00:00:00Z',
    };
    const result = attachManualGiftAssignments(
      [automaticGift, { ...automaticGift, id: 2, manual_distribution: true }],
      [manualGift]
    );

    expect(result[0].manual_assignment).toBeUndefined();
    expect(result[1].manual_assignment?.recipient?.id).toBe(12);
  });

  it('derives distinct pending, manual, and automatic statuses', () => {
    expect(getManualGiftStatus({ ...automaticGift, review_status: 'pending_review' }, new Set())).toMatchObject({
      status: 'pending_review',
    });
    expect(getManualGiftStatus({ ...automaticGift, manual_distribution: true }, new Set())).toMatchObject({
      status: 'manual_unassigned',
    });
    expect(getManualGiftStatus({ ...automaticGift, id: 2, manual_distribution: true, manual_assignment: {
      id: 2, event_id: 7, description: 'Cap', review_status: 'approved', manual_distribution: true,
      recipient: { id: 12, display_name: 'Alex', status: 'active' }, created_at: '2026-07-11T00:00:00Z',
    } }, new Set())).toMatchObject({ status: 'manual_assigned' });
    expect(getManualGiftStatus(automaticGift, new Set([1]))).toMatchObject({ status: 'automatic_assigned' });
    expect(getManualGiftStatus(automaticGift, new Set())).toMatchObject({ status: 'automatic_unassigned' });
  });

  it('formats a searchable recipient label without duplicating at-signs', () => {
    expect(formatManualRecipientSearchLabel('Alex', '@alex')).toBe('Alex (@alex)');
    expect(formatManualRecipientSearchLabel('Alex')).toBe('Alex');
  });
});
