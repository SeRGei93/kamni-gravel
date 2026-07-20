import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';

import type { PrizeDistribution } from '@/types';

import {
  PRIZE_DISTRIBUTION_COLUMNS,
  PRIZE_DISTRIBUTION_COLUMNS_STORAGE_KEY,
  PRIZE_DISTRIBUTION_DEFAULT_VISIBLE_KEYS,
  PRIZE_DISTRIBUTION_TOGGLEABLE_COLUMN_KEYS,
} from './prizeDistributionColumns';

const row: PrizeDistribution = {
  participant_id: 7,
  participant_name: 'Иван Петров',
  gender: 'male',
  bike_type: 'gravel',
  status: 'active',
  display_place: 2,
  place_absolute: 5,
  place_by_gender: 2,
  place_by_gender_bike: 1,
  result_criteria: [],
  match_reason: 'place',
  matched_gift_assignments: [
    {
      gift_id: 10,
      gift: {
        id: 10,
        user_id: 42,
        first_name: 'Анна',
        last_name: 'Иванова',
        event_id: 77,
        description: 'Шлем',
        review_status: 'approved',
        created_at: '2026-07-15T00:00:00Z',
      },
      rule_type: 'places',
      target_rank: 1,
      assigned_rank: 2,
      is_fallback: true,
      fallback_reason: 'target_unavailable',
      match_reason: 'place',
    },
  ],
};

describe('prize distribution column registry', () => {
  it('keeps the award workflow columns in a fixed order', () => {
    expect(PRIZE_DISTRIBUTION_COLUMNS.map((column) => column.key)).toEqual([
      'display_place',
      'participant',
      'gender',
      'bike_type',
      'status',
      'place_absolute',
      'place_by_gender',
      'place_by_gender_bike',
      'criteria',
      'prizes',
      'match_reason',
    ]);
    expect(PRIZE_DISTRIBUTION_COLUMNS_STORAGE_KEY).toBe('prize-distribution:visible-columns');
  });

  it('does not allow hiding place, participant, or prizes', () => {
    const alwaysVisible = PRIZE_DISTRIBUTION_COLUMNS
      .filter((column) => column.alwaysVisible)
      .map((column) => column.key);

    expect(alwaysVisible).toEqual(['display_place', 'participant', 'prizes']);
    expect(PRIZE_DISTRIBUTION_TOGGLEABLE_COLUMN_KEYS).not.toContain('prizes');
    expect(PRIZE_DISTRIBUTION_DEFAULT_VISIBLE_KEYS).toContain('status');
  });

  it('renders each prize separately with its donor and fallback context', () => {
    const prizesColumn = PRIZE_DISTRIBUTION_COLUMNS.find((column) => column.key === 'prizes');
    expect(prizesColumn).toBeDefined();

    const markup = renderToStaticMarkup(prizesColumn!.render(row));
    expect(markup).toContain('Шлем');
    expect(markup).toContain('от Анна Иванова');
    expect(markup).toContain('место 1 -&gt; выдано месту 2');
    expect(markup).toContain('целевое место недоступно');
  });

  it('shows manually assigned prizes alongside automatic prizes', () => {
    const prizesColumn = PRIZE_DISTRIBUTION_COLUMNS.find((column) => column.key === 'prizes');
    expect(prizesColumn).toBeDefined();

    const markup = renderToStaticMarkup(prizesColumn!.render(row, [
      {
        id: 11,
        event_id: 77,
        description: 'Термокружка',
        review_status: 'approved',
        manual_distribution: true,
        recipient: { id: row.participant_id, display_name: row.participant_name, status: 'active' },
        created_at: '2026-07-15T00:00:00Z',
      },
    ]));

    expect(markup).toContain('Шлем');
    expect(markup).toContain('Термокружка');
    expect(markup).toContain('Назначен вручную');
  });
});
