import { describe, expect, it } from 'vitest';
import { matchesSearchQuery } from './search';

describe('matchesSearchQuery', () => {
  it('matches a username with or without the @ prefix', () => {
    expect(matchesSearchQuery('rider', ['rider'])).toBe(true);
    expect(matchesSearchQuery('@rider', ['rider'])).toBe(true);
    expect(matchesSearchQuery('@@RIDER', ['rider'])).toBe(true);
  });

  it('matches names and numeric identifiers without throwing on missing usernames', () => {
    expect(matchesSearchQuery('Иван', ['Иван Петров', undefined])).toBe(true);
    expect(matchesSearchQuery('1001', [1001, undefined])).toBe(true);
  });

  it('does not match unrelated values', () => {
    expect(matchesSearchQuery('@rider', ['another-user'])).toBe(false);
  });
});
