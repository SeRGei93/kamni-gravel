import { describe, expect, it } from 'vitest';
import {
  buildGiftPlaceRuleFromForm,
  formatGiftPlaceRule,
  formatPlaceList,
  formatPrizeAssignment,
  getGiftPlaceRuleFormState,
  parseGiftPlaceRulePlaces,
} from './giftPlaceRule';
import type { Gift } from '@/types';

describe('parseGiftPlaceRulePlaces', () => {
  it('parses comma-separated places and ranges into sorted unique places', () => {
    expect(parseGiftPlaceRulePlaces('1, 3, 10-15, 3')).toEqual([
      1, 3, 10, 11, 12, 13, 14, 15,
    ]);
  });

  it('rejects malformed input', () => {
    for (const input of ['', ' ', '0', '-1', '1,', '1,,3', 'abc', '10-5', '1-']) {
      expect(() => parseGiftPlaceRulePlaces(input), input).toThrow();
    }
  });
});

describe('formatGiftPlaceRule', () => {
  it('formats empty rule', () => {
    expect(formatGiftPlaceRule(null)).toBe('Без привязки');
  });

  it('formats ranges and sparse places', () => {
    expect(formatGiftPlaceRule({ type: 'places', places: [10, 11, 12, 13, 14, 15] })).toBe('Места 10-15');
    expect(formatGiftPlaceRule({ type: 'places', places: [1, 3, 5] })).toBe('Места 1, 3, 5');
  });

  it('formats last_n', () => {
    expect(formatGiftPlaceRule({ type: 'last_n', last_count: 5 })).toBe('5 последних');
  });
});

describe('getGiftPlaceRuleFormState', () => {
  const baseGift: Gift = {
    id: 1,
    user_id: 10,
    event_id: 20,
    description: 'Prize',
    gender_filter: 'all',
    bike_type_filter: 'all',
    review_status: 'pending_review',
    created_at: '2026-05-28T00:00:00Z',
  };

  it('prefills every supported gift rule variant', () => {
    expect(
      getGiftPlaceRuleFormState({
        ...baseGift,
        place_rule: { type: 'places', places: [10, 11, 12, 15] },
      })
    ).toEqual({ mode: 'places', placesInput: '10-12, 15', lastCount: '' });

    expect(
      getGiftPlaceRuleFormState({
        ...baseGift,
        place_rule: { type: 'last_n', last_count: 5 },
      })
    ).toEqual({ mode: 'last_n', placesInput: '', lastCount: '5' });

    expect(getGiftPlaceRuleFormState({ ...baseGift, place: 3 })).toEqual({
      mode: 'places',
      placesInput: '3',
      lastCount: '',
    });

    expect(getGiftPlaceRuleFormState(baseGift)).toEqual({
      mode: 'none',
      placesInput: '',
      lastCount: '',
    });
  });
});

describe('buildGiftPlaceRuleFromForm', () => {
  it('builds API payloads for every editor mode', () => {
    expect(buildGiftPlaceRuleFromForm('none', '1, 2', '3')).toBeNull();
    expect(buildGiftPlaceRuleFromForm('places', '1, 3-5, 3', '')).toEqual({
      type: 'places',
      places: [1, 3, 4, 5],
    });
    expect(buildGiftPlaceRuleFromForm('last_n', '', '2')).toEqual({
      type: 'last_n',
      last_count: 2,
    });
  });

  it('rejects invalid editor payloads before submit', () => {
    expect(() => buildGiftPlaceRuleFromForm('places', '1,,3', '')).toThrow();
    expect(() => buildGiftPlaceRuleFromForm('last_n', '', '0')).toThrow();
    expect(() => buildGiftPlaceRuleFromForm('last_n', '', '1.5')).toThrow();
  });
});

describe('formatPlaceList', () => {
  it('compacts adjacent places into ranges', () => {
    expect(formatPlaceList([5, 1, 2, 3, 8, 9])).toBe('1-3, 5, 8-9');
  });
});

describe('formatPrizeAssignment', () => {
  it('formats exact and fallback assignment summaries for rule variants', () => {
    expect(
      formatPrizeAssignment({
        gift: {} as never,
        gift_id: 1,
        rule_type: 'places',
        target_rank: 15,
        assigned_rank: 15,
        is_fallback: false,
        match_reason: 'place',
      })
    ).toBe('место 15');

    expect(
      formatPrizeAssignment({
        gift: {} as never,
        gift_id: 1,
        rule_type: 'places',
        target_rank: 15,
        assigned_rank: 14,
        is_fallback: true,
        fallback_reason: 'target_unavailable',
        match_reason: 'place',
      })
    ).toBe('место 15 -> выдано месту 14');

    expect(
      formatPrizeAssignment({
        gift: {} as never,
        gift_id: 2,
        rule_type: 'last_n',
        target_rank: 3,
        assigned_rank: 3,
        is_fallback: false,
        match_reason: 'place',
      })
    ).toBe('место 3');

    expect(
      formatPrizeAssignment({
        gift: {} as never,
        gift_id: 3,
        rule_type: 'none',
        assigned_rank: 1,
        is_fallback: false,
        match_reason: 'match',
      })
    ).toBe('выдано месту 1');
  });
});
