import { describe, expect, it } from 'vitest';
import type { Gift } from '@/types';
import {
  filterMiniappGiftsBySearch,
  shouldShowGiftPlaceGaps,
} from './miniappGiftSearch';

function gift(overrides: Partial<Gift>): Gift {
  return {
    id: 1,
    user_id: 101,
    event_id: 1,
    description: 'Приз',
    review_status: 'approved',
    created_at: '2026-07-14T12:00:00Z',
    ...overrides,
  };
}

describe('filterMiniappGiftsBySearch', () => {
  const gifts = [
    gift({ id: 1, description: 'Шлем для гравия', username: 'velo_owner' }),
    gift({ id: 2, description: 'Фляга', first_name: 'Анна', last_name: 'Петрова' }),
    gift({ id: 3, description: 'Носки', first_name: 'Иван' }),
  ];

  it('finds gifts by description, full author name, and username with @', () => {
    expect(filterMiniappGiftsBySearch(gifts, 'ГРАВИЯ').map(({ id }) => id)).toEqual([1]);
    expect(filterMiniappGiftsBySearch(gifts, 'Анна Петрова').map(({ id }) => id)).toEqual([2]);
    expect(filterMiniappGiftsBySearch(gifts, '@velo_owner').map(({ id }) => id)).toEqual([1]);
  });

  it('keeps all gifts for an empty or whitespace-only query', () => {
    expect(filterMiniappGiftsBySearch(gifts, '')).toEqual(gifts);
    expect(filterMiniappGiftsBySearch(gifts, '  \n')).toEqual(gifts);
  });

  it('returns no gifts when the query has no matches', () => {
    expect(filterMiniappGiftsBySearch(gifts, 'Несуществующий')).toEqual([]);
  });

  it('preserves the original order without mutating the input array', () => {
    const originalIds = gifts.map(({ id }) => id);
    const filtered = filterMiniappGiftsBySearch(gifts, 'а');

    expect(filtered.map(({ id }) => id)).toEqual([1, 2, 3]);
    expect(gifts.map(({ id }) => id)).toEqual(originalIds);
    expect(filtered).not.toBe(gifts);
  });
});

describe('shouldShowGiftPlaceGaps', () => {
  it('shows place gaps only when search is inactive', () => {
    expect(shouldShowGiftPlaceGaps('')).toBe(true);
    expect(shouldShowGiftPlaceGaps('  \t')).toBe(true);
    expect(shouldShowGiftPlaceGaps('шлем')).toBe(false);
  });
});
