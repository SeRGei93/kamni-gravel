import { describe, expect, it } from 'vitest';
import { hasSearchQuery, matchesSearchQuery } from './search';

describe('hasSearchQuery', () => {
  it('treats an empty or whitespace-only query as inactive', () => {
    expect(hasSearchQuery('')).toBe(false);
    expect(hasSearchQuery('   \n\t')).toBe(false);
  });

  it('treats a query with content as active', () => {
    expect(hasSearchQuery('Гравий')).toBe(true);
  });
});

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
