import { describe, expect, it } from 'vitest';
import {
  formatHeartRateTimeProduct,
  isSortableColumn,
  PARTICIPANT_COLUMNS,
} from './participantColumns';

describe('participantColumns', () => {
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
});
