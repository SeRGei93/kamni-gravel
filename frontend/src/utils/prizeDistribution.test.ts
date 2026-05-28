import { describe, expect, it } from 'vitest';
import type { Gift, PrizeDistribution } from '@/types';
import {
  countParticipantsWithPrizes,
  countPrizeAssignmentSlots,
  formatUnassignedPrizeSlot,
  participantHasPrize,
} from './prizeDistribution';

describe('frontend prize distribution summaries', () => {
  const gift: Gift = {
    id: 10,
    user_id: 20,
    event_id: 30,
    description: 'Prize',
    review_status: 'approved',
    created_at: '2026-05-28T00:00:00Z',
  };

  const baseRow: PrizeDistribution = {
    participant_id: 1,
    participant_name: 'Rider',
    gender: 'male',
    bike_type: 'gravel',
    place_absolute: 1,
    place_by_gender: 1,
    place_by_gender_bike: 1,
    result_criteria: [],
    match_reason: 'no_match',
  };

  it('counts participants and concrete assignment slots from the new API shape', () => {
    const distribution: PrizeDistribution[] = [
      {
        ...baseRow,
        participant_id: 1,
        matched_gift_assignments: [
          {
            gift,
            gift_id: gift.id,
            rule_type: 'places',
            target_rank: 1,
            assigned_rank: 1,
            is_fallback: false,
            match_reason: 'place',
          },
          {
            gift: { ...gift, id: 11 },
            gift_id: 11,
            rule_type: 'last_n',
            target_rank: 4,
            assigned_rank: 4,
            is_fallback: false,
            match_reason: 'place',
          },
        ],
      },
      {
        ...baseRow,
        participant_id: 2,
        matched_gift_assignments: [],
        matched_gifts: [{ ...gift, id: 12 }],
      },
      { ...baseRow, participant_id: 3 },
    ];

    expect(participantHasPrize(distribution[0])).toBe(true);
    expect(participantHasPrize(distribution[2])).toBe(false);
    expect(countParticipantsWithPrizes(distribution)).toBe(2);
    expect(countPrizeAssignmentSlots(distribution)).toBe(3);
  });

  it('formats unassigned slot badges for visible diagnostics', () => {
    expect(
      formatUnassignedPrizeSlot({
        gift_id: 15,
        rule_type: 'places',
        target_rank: 9,
        reason: 'target_unavailable',
      })
    ).toBe('Приз 15, место 9');

    expect(
      formatUnassignedPrizeSlot({
        gift_id: 16,
        rule_type: 'last_n',
        reason: 'target_unavailable',
      })
    ).toBe('Приз 16, место -');
  });
});
