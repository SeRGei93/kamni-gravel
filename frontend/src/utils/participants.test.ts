import { describe, expect, it } from 'vitest';
import type { Participant } from '@/types';
import { filterParticipants } from './participants';

function makeParticipant(overrides: Partial<Participant>): Participant {
  return {
    id: 1,
    user_id: 1000,
    username: 'rider',
    first_name: 'Ivan',
    last_name: 'Petrov',
    event_id: 1,
    bike_type: 'gravel',
    gender: 'male',
    is_finished: false,
    registered_at: '2026-06-15T00:00:00Z',
    has_gift: false,
    prizes_count: 0,
    ...overrides,
  };
}

const withGift = makeParticipant({
  id: 1,
  user_id: 1001,
  username: 'giver',
  first_name: 'Anna',
  last_name: 'Sidorova',
  has_gift: true,
});
const withoutGift = makeParticipant({
  id: 2,
  user_id: 1002,
  username: 'racer',
  first_name: 'Boris',
  last_name: 'Ivanov',
  has_gift: false,
});

describe('filterParticipants', () => {
  it('returns everyone when no filters are applied', () => {
    const result = filterParticipants([withGift, withoutGift]);
    expect(result).toHaveLength(2);
  });

  it('keeps only participants who added a gift when has_gift filter is "yes"', () => {
    const result = filterParticipants([withGift, withoutGift], {
      hasGiftFilter: 'yes',
    });
    expect(result.map((p) => p.id)).toEqual([1]);
  });

  it('keeps only participants without a gift when has_gift filter is "no"', () => {
    const result = filterParticipants([withGift, withoutGift], {
      hasGiftFilter: 'no',
    });
    expect(result.map((p) => p.id)).toEqual([2]);
  });

  it('does not change the set for has_gift filter "all"', () => {
    const result = filterParticipants([withGift, withoutGift], {
      hasGiftFilter: 'all',
    });
    expect(result).toHaveLength(2);
  });

  it('matches the search query against name, username, and user id', () => {
    expect(filterParticipants([withGift, withoutGift], { searchQuery: 'anna' })).toEqual([
      withGift,
    ]);
    expect(filterParticipants([withGift, withoutGift], { searchQuery: 'RACER' })).toEqual([
      withoutGift,
    ]);
    expect(filterParticipants([withGift, withoutGift], { searchQuery: '1001' })).toEqual([
      withGift,
    ]);
  });

  it('combines search and has_gift filters', () => {
    const result = filterParticipants([withGift, withoutGift], {
      searchQuery: 'i',
      hasGiftFilter: 'no',
    });
    expect(result.map((p) => p.id)).toEqual([2]);
  });
});
