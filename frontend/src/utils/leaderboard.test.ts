import { describe, expect, it } from 'vitest';
import type { MiniappLeaderboardEntry } from '@/types';
import {
  filterRankedLeaderboardBySearch,
  rankAndFilterLeaderboard,
} from './leaderboard';

function leaderboardEntry(overrides: Partial<MiniappLeaderboardEntry>): MiniappLeaderboardEntry {
  return {
    id: 1,
    name: 'Участник',
    gender: 'male',
    bike_type: 'gravel',
    status: 'active',
    is_finished: true,
    place: 0,
    elapsed_time_sec: 10_000,
    ...overrides,
  };
}

describe('rankAndFilterLeaderboard', () => {
  it('assigns places in the gender standings without filtering by bike type', () => {
    const entries = [
      leaderboardEntry({ id: 1, gender: 'male', bike_type: 'gravel', elapsed_time_sec: 10_000 }),
      leaderboardEntry({ id: 2, gender: 'female', bike_type: 'road', elapsed_time_sec: 9_000 }),
      leaderboardEntry({ id: 3, gender: 'female', bike_type: 'mtb', elapsed_time_sec: 11_000 }),
    ];

    const rows = rankAndFilterLeaderboard(entries, 'female', 'all');

    expect(rows.map(({ entry, place }) => ({ id: entry.id, place }))).toEqual([
      { id: 2, place: 1 },
      { id: 3, place: 2 },
    ]);
  });

  it('assigns places in the bike standings without filtering by gender', () => {
    const entries = [
      leaderboardEntry({ id: 1, gender: 'male', bike_type: 'mtb', elapsed_time_sec: 10_000 }),
      leaderboardEntry({ id: 2, gender: 'female', bike_type: 'mtb', elapsed_time_sec: 9_000 }),
      leaderboardEntry({ id: 3, gender: 'male', bike_type: 'gravel', elapsed_time_sec: 8_000 }),
    ];

    const rows = rankAndFilterLeaderboard(entries, 'all', 'mtb');

    expect(rows.map(({ entry, place }) => ({ id: entry.id, place }))).toEqual([
      { id: 2, place: 1 },
      { id: 1, place: 2 },
    ]);
  });

  it('filters already ranked rows case-insensitively without changing places', () => {
    const entries = [
      leaderboardEntry({ id: 1, name: 'Иван Петров', elapsed_time_sec: 10_000 }),
      leaderboardEntry({ id: 2, name: 'Анна Сидорова', elapsed_time_sec: 9_000 }),
    ];

    const rankedRows = rankAndFilterLeaderboard(entries, 'all', 'all');
    const rows = filterRankedLeaderboardBySearch(rankedRows, 'ПЕТРОВ');

    expect(rows.map(({ entry, place }) => ({ id: entry.id, place }))).toEqual([
      { id: 1, place: 2 },
    ]);
  });

  it('keeps all ranked rows for an empty or whitespace-only query', () => {
    const rows = rankAndFilterLeaderboard(
      [
        leaderboardEntry({ id: 1, name: 'Иван Петров' }),
        leaderboardEntry({ id: 2, name: 'Анна Сидорова' }),
      ],
      'all',
      'all'
    );

    expect(filterRankedLeaderboardBySearch(rows, '')).toEqual(rows);
    expect(filterRankedLeaderboardBySearch(rows, '  \t')).toEqual(rows);
  });

  it('returns no rows when the query has no matches', () => {
    const rows = rankAndFilterLeaderboard(
      [leaderboardEntry({ name: 'Иван Петров' })],
      'all',
      'all'
    );

    expect(filterRankedLeaderboardBySearch(rows, 'Несуществующий')).toEqual([]);
  });

  it('searches within the selected gender and bike category standings', () => {
    const entries = [
      leaderboardEntry({ id: 1, name: 'Иван Гравий', gender: 'male', bike_type: 'gravel', elapsed_time_sec: 10_000 }),
      leaderboardEntry({ id: 2, name: 'Пётр Гравий', gender: 'male', bike_type: 'gravel', elapsed_time_sec: 9_000 }),
      leaderboardEntry({ id: 3, name: 'Иван Шоссе', gender: 'male', bike_type: 'road', elapsed_time_sec: 8_000 }),
      leaderboardEntry({ id: 4, name: 'Иван MTB', gender: 'female', bike_type: 'gravel', elapsed_time_sec: 7_000 }),
    ];

    const rankedRows = rankAndFilterLeaderboard(entries, 'male', 'gravel');
    const rows = filterRankedLeaderboardBySearch(rankedRows, 'Иван');

    expect(rows.map(({ entry, place }) => ({ id: entry.id, place }))).toEqual([
      { id: 1, place: 2 },
    ]);
  });
});
