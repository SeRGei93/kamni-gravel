import { describe, expect, it } from 'vitest';
import { normalizePageSize } from './usePaginationParams';

describe('normalizePageSize', () => {
  it('preserves a valid legacy page size from the URL', () => {
    expect(normalizePageSize('75')).toBe(75);
  });

  it('supports the all page-size option', () => {
    expect(normalizePageSize('all')).toBe('all');
  });

  it('keeps invalid page sizes within the supported range', () => {
    expect(normalizePageSize(null)).toBe(50);
    expect(normalizePageSize('10')).toBe(50);
    expect(normalizePageSize('200')).toBe(100);
  });
});
