import { describe, expect, it } from 'vitest';
import {
  formatHeartRateTimeProduct,
  isSortableColumn,
  PARTICIPANT_COLUMNS,
} from './participantColumns';
import type { Participant } from '@/types';

describe('participantColumns', () => {
  const participant: Participant = {
    id: 1,
    user_id: 101,
    username: 'rider',
    first_name: 'Анна',
    last_name: 'Иванова',
    event_id: 77,
    bike_type: 'gravel',
    gender: 'female',
    status: 'dnf',
  is_finished: false,
  has_gift: true,
  prizes_count: 2,
  registered_at: '2026-07-15T08:00:00Z',
    place: 0,
    elapsed_time: '01:23:45',
    heart_rate_time_product: 7200.6,
    result_link: 'https://strava.example/activity/1',
    distance_meters: 12345,
  };

  it('formats the total time and average heart rate product as an integer', () => {
    expect(formatHeartRateTimeProduct(7200)).toMatch(/^7[\s\u00a0\u202f]200$/);
    expect(formatHeartRateTimeProduct(7200.6)).toMatch(/^7[\s\u00a0\u202f]201$/);
  });

  it('omits the product when the result has insufficient metrics', () => {
    expect(formatHeartRateTimeProduct()).toBeUndefined();
    expect(formatHeartRateTimeProduct(null)).toBeUndefined();
    expect(formatHeartRateTimeProduct(Number.NaN)).toBeUndefined();
  });

  it('adds the product as a visible sortable participant column', () => {
    const column = PARTICIPANT_COLUMNS.find(
      (item) => item.key === 'heart_rate_time_product',
    );

    expect(column?.defaultVisible).toBe(true);
    expect(isSortableColumn('heart_rate_time_product')).toBe(true);
  });

  it('returns scalar export values instead of table markup', () => {
    const byKey = (key: string) =>
      PARTICIPANT_COLUMNS.find((column) => column.key === key)!;

    expect(byKey('name').exportValue(participant)).toBe('Анна Иванова');
    expect(byKey('place').exportValue(participant)).toBeNull();
    expect(byKey('gender').exportValue(participant)).toBe('Ж');
    expect(byKey('bike_type').exportValue(participant)).toBe('Гравийник');
    expect(byKey('status').exportValue(participant)).toBe('Сошёл с дистанции');
    expect(byKey('elapsed_time').exportValue(participant)).toBe('01:23:45');
    expect(byKey('heart_rate_time_product').exportValue(participant)).toBe(7201);
    expect(byKey('result_link').exportValue(participant)).toBe(
      'https://strava.example/activity/1',
    );
    expect(byKey('has_gift').exportValue(participant)).toBe('Да');
    expect(byKey('prizes_count').exportValue(participant)).toBe(2);
    expect(byKey('distance_km').exportValue(participant)).toBe('12,3 км');
  });

  it('keeps empty optional values empty in the export', () => {
    const withoutResult: Participant = {
      ...participant,
      first_name: '',
      last_name: '',
      username: '',
      status: 'active',
      result_link: undefined,
      elapsed_time: undefined,
      distance_meters: undefined,
      prizes_count: 0,
    };
    const byKey = (key: string) =>
      PARTICIPANT_COLUMNS.find((column) => column.key === key)!;

    expect(byKey('name').exportValue(withoutResult)).toBeNull();
    expect(byKey('username').exportValue(withoutResult)).toBe('@user101');
    expect(byKey('status').exportValue(withoutResult)).toBeNull();
    expect(byKey('result_link').exportValue(withoutResult)).toBeNull();
    expect(byKey('elapsed_time').exportValue(withoutResult)).toBeNull();
    expect(byKey('distance_km').exportValue(withoutResult)).toBeNull();
    expect(byKey('prizes_count').exportValue(withoutResult)).toBeNull();
  });
});
