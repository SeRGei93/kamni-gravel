import { describe, expect, it } from 'vitest';
import {
  attachManualGiftAssignments,
  buildManualGiftUpdate,
  canAssignRandomRecipient,
  canAssignRandomRecipientIncludingAwarded,
  filterGiftsByDistribution,
  formatManualRecipientSearchLabel,
  getManualGiftsForRecipient,
  getManualGiftStatus,
  isCurrentManualGiftsRequest,
  isGiftDistributed,
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

  it('returns only manually assigned gifts for the requested recipient in API order', () => {
    const manualGifts: ManualGift[] = [
      {
        id: 10,
        event_id: 7,
        description: 'First assigned manual gift',
        review_status: 'approved',
        manual_distribution: true,
        recipient: { id: 12, display_name: 'Participant 12', status: 'active' },
        created_at: '2026-07-15T00:00:00Z',
      },
      {
        id: 11,
        event_id: 7,
        description: 'Unassigned manual gift',
        review_status: 'approved',
        manual_distribution: true,
        created_at: '2026-07-15T00:00:00Z',
      },
      {
        id: 12,
        event_id: 7,
        description: 'Another participant manual gift',
        review_status: 'approved',
        manual_distribution: true,
        recipient: { id: 24, display_name: 'Participant 24', status: 'active' },
        created_at: '2026-07-15T00:00:00Z',
      },
      {
        id: 13,
        event_id: 7,
        description: 'Unexpected automatic gift',
        review_status: 'approved',
        manual_distribution: false,
        recipient: { id: 12, display_name: 'Participant 12', status: 'active' },
        created_at: '2026-07-15T00:00:00Z',
      },
      {
        id: 14,
        event_id: 7,
        description: 'Second assigned manual gift',
        review_status: 'approved',
        manual_distribution: true,
        recipient: { id: 12, display_name: 'Participant 12', status: 'active' },
        created_at: '2026-07-15T00:00:00Z',
      },
    ];

    expect(getManualGiftsForRecipient(manualGifts, 12).map((gift) => gift.id)).toEqual([10, 14]);
  });

  it('ignores a manual-gifts response after a newer request starts', () => {
    const firstRequestVersion = 1;
    const latestRequestVersion = 2;

    expect(
      isCurrentManualGiftsRequest(firstRequestVersion, latestRequestVersion)
    ).toBe(false);
    expect(
      isCurrentManualGiftsRequest(latestRequestVersion, latestRequestVersion)
    ).toBe(true);
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

  it('filters manual gifts independently of their review presentation status', () => {
    const manualAssigned: Gift = {
      ...automaticGift,
      id: 2,
      manual_distribution: true,
      manual_assignment: {
        id: 2,
        event_id: 7,
        description: 'Assigned manual gift',
        review_status: 'approved',
        manual_distribution: true,
        recipient: { id: 12, display_name: 'Alex', status: 'active' },
        created_at: automaticGift.created_at,
      },
    };
    const manualUnassigned: Gift = {
      ...automaticGift,
      id: 3,
      manual_distribution: true,
    };
    const pendingManualUnassigned: Gift = {
      ...manualUnassigned,
      id: 4,
      review_status: 'pending_review',
    };
    const pendingManualAssigned: Gift = {
      ...manualAssigned,
      id: 5,
      review_status: 'pending_review',
    };
    const gifts = [
      automaticGift,
      manualAssigned,
      manualUnassigned,
      pendingManualUnassigned,
      pendingManualAssigned,
    ];

    expect(
      filterGiftsByDistribution(gifts, 'manual', new Set([automaticGift.id])).map(
        (gift) => gift.id
      )
    ).toEqual([2, 3, 4, 5]);
    expect(
      filterGiftsByDistribution(gifts, 'manual_unassigned', new Set()).map(
        (gift) => gift.id
      )
    ).toEqual([3, 4]);
    expect(
      filterGiftsByDistribution(gifts, 'assigned', new Set([automaticGift.id])).map(
        (gift) => gift.id
      )
    ).toEqual([1, 2]);
  });

  it('formats a searchable recipient label without duplicating at-signs', () => {
    expect(formatManualRecipientSearchLabel('Alex', '@alex')).toBe('Alex (@alex)');
    expect(formatManualRecipientSearchLabel('Alex')).toBe('Alex');
  });

  it('treats manual and automatic assignments as distributed', () => {
    const manualGift: Gift = {
      ...automaticGift,
      id: 2,
      manual_distribution: true,
      manual_assignment: {
        id: 2,
        event_id: 7,
        description: 'Cap',
        review_status: 'approved',
        manual_distribution: true,
        recipient: { id: 12, display_name: 'Alex', status: 'active' },
        created_at: '2026-07-11T00:00:00Z',
      },
    };

    expect(isGiftDistributed(automaticGift, new Set([1]))).toBe(true);
    expect(isGiftDistributed(manualGift, new Set())).toBe(true);
    expect(isGiftDistributed(automaticGift, new Set())).toBe(false);
    expect(isGiftDistributed({ ...automaticGift, review_status: 'pending_review' }, new Set([1]))).toBe(false);
  });

  it('allows random distribution only for approved unassigned gifts', () => {
    expect(canAssignRandomRecipient(automaticGift, new Set())).toBe(true);
    expect(canAssignRandomRecipient({ ...automaticGift, manual_distribution: true }, new Set())).toBe(true);
    expect(canAssignRandomRecipient(automaticGift, new Set([automaticGift.id]))).toBe(false);
    expect(canAssignRandomRecipient({ ...automaticGift, review_status: 'pending_review' }, new Set())).toBe(false);
    expect(canAssignRandomRecipient({
      ...automaticGift,
      manual_distribution: true,
      manual_assignment: {
        id: automaticGift.id,
        event_id: automaticGift.event_id,
        description: automaticGift.description,
        review_status: 'approved',
        manual_distribution: true,
        recipient: { id: 12, display_name: 'Alex', status: 'active' },
        created_at: automaticGift.created_at,
      },
    }, new Set())).toBe(false);
  });

  it('allows the including-awarded random action only for an approved unassigned manual gift', () => {
    expect(canAssignRandomRecipientIncludingAwarded(automaticGift, new Set())).toBe(false);
    expect(
      canAssignRandomRecipientIncludingAwarded(
        { ...automaticGift, manual_distribution: true },
        new Set()
      )
    ).toBe(true);
    expect(
      canAssignRandomRecipientIncludingAwarded(
        { ...automaticGift, manual_distribution: true, review_status: 'pending_review' },
        new Set()
      )
    ).toBe(false);
    expect(
      canAssignRandomRecipientIncludingAwarded(
        {
          ...automaticGift,
          manual_distribution: true,
          manual_assignment: {
            id: automaticGift.id,
            event_id: automaticGift.event_id,
            description: automaticGift.description,
            review_status: 'approved',
            manual_distribution: true,
            recipient: { id: 12, display_name: 'Alex', status: 'active' },
            created_at: automaticGift.created_at,
          },
        },
        new Set()
      )
    ).toBe(false);
  });
});
