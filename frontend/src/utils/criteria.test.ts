import { describe, expect, it } from 'vitest';
import { getCriteriaColor, getCriteriaTypeLabel } from './criteria';

describe('criteria helpers', () => {
  it('formats random criteria with a distinct badge color', () => {
    expect(getCriteriaTypeLabel('random')).toBe('Рандом');
    expect(getCriteriaColor('random')).toBe('primary');
  });
});
