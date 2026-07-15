import { describe, expect, it } from 'vitest';
import type { PrizeDistribution } from '@/types';
import { getAutomaticGiftRecipient } from './giftRecipient';

const distribution: PrizeDistribution[] = [
  {
    participant_id: 11,
    participant_name: 'Alex Rider',
    gender: 'male',
    bike_type: 'gravel',
    status: 'active',
    place_absolute: 1,
    place_by_gender: 1,
    place_by_gender_bike: 1,
    result_criteria: [],
    matched_gift_assignments: [
      {
        gift_id: 77,
        gift: {
          id: 77,
          user_id: 1,
          event_id: 5,
          description: 'Bottle',
          review_status: 'approved',
          created_at: '2026-07-15T00:00:00Z',
        },
        rule_type: 'none',
        assigned_rank: 1,
        is_fallback: false,
        match_reason: 'match',
      },
    ],
    match_reason: 'match',
  },
  {
    participant_id: 12,
    participant_name: 'Legacy Rider',
    gender: 'female',
    bike_type: 'mtb',
    status: 'active',
    place_absolute: 2,
    place_by_gender: 1,
    place_by_gender_bike: 1,
    result_criteria: [],
    matched_gifts: [
      {
        id: 78,
        user_id: 2,
        event_id: 5,
        description: 'Cap',
        review_status: 'approved',
        created_at: '2026-07-15T00:00:00Z',
      },
    ],
    match_reason: 'match',
  },
];

describe('getAutomaticGiftRecipient', () => {
  it('returns the recipient assigned through the current gift assignment contract', () => {
    expect(getAutomaticGiftRecipient(distribution, 77)).toEqual({
      participantID: 11,
      participantName: 'Alex Rider',
    });
  });

  it('supports the legacy matched gifts representation and ignores unmatched gifts', () => {
    expect(getAutomaticGiftRecipient(distribution, 78)).toEqual({
      participantID: 12,
      participantName: 'Legacy Rider',
    });
    expect(getAutomaticGiftRecipient(distribution, 79)).toBeUndefined();
  });
});
