import { describe, expect, it } from 'vitest';
import type { Criteria } from '@/types';
import {
  addSelectedCriterionId,
  getCriteriaColor,
  getCriteriaTypeLabel,
  mergeCriterion,
} from './criteria';

function makeCriterion(overrides: Partial<Criteria>): Criteria {
  return {
    id: 1,
    name: 'Speed',
    description: '',
    criteria_type: 'custom',
    created_at: '2026-06-16T00:00:00Z',
    ...overrides,
  };
}

describe('criteria helpers', () => {
  it('formats random criteria with a distinct badge color', () => {
    expect(getCriteriaTypeLabel('random')).toBe('Рандом');
    expect(getCriteriaColor('random')).toBe('primary');
  });
});

describe('mergeCriterion', () => {
  it('appends a criterion that is not yet in the list', () => {
    const list = [makeCriterion({ id: 1 })];
    const result = mergeCriterion(list, makeCriterion({ id: 2, name: 'New' }));

    expect(result.map((c) => c.id)).toEqual([1, 2]);
    expect(list).toHaveLength(1); // не мутирует исходный массив
  });

  it('replaces an existing criterion with the same id', () => {
    const list = [
      makeCriterion({ id: 1, name: 'Old' }),
      makeCriterion({ id: 2 }),
    ];
    const result = mergeCriterion(list, makeCriterion({ id: 1, name: 'Updated' }));

    expect(result).toHaveLength(2);
    expect(result.find((c) => c.id === 1)?.name).toBe('Updated');
  });
});

describe('addSelectedCriterionId', () => {
  it('adds an id when it is absent', () => {
    expect(addSelectedCriterionId([1, 2], 3)).toEqual([1, 2, 3]);
  });

  it('does not duplicate an id that is already selected', () => {
    const ids = [1, 2];
    expect(addSelectedCriterionId(ids, 2)).toBe(ids);
  });
});
