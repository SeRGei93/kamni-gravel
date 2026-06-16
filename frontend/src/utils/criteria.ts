import type { Criteria, CriteriaType } from '@/types';

type BadgeColor = 'primary' | 'success' | 'error' | 'warning' | 'info' | 'light' | 'dark';

/**
 * Добавляет критерий в список или заменяет существующий с тем же id.
 * Чистая функция — не мутирует входной массив.
 */
export function mergeCriterion(list: Criteria[], criterion: Criteria): Criteria[] {
  const index = list.findIndex((item) => item.id === criterion.id);
  if (index === -1) {
    return [...list, criterion];
  }
  const next = [...list];
  next[index] = criterion;
  return next;
}

/** Добавляет id в выбор, если его там ещё нет (идемпотентно). */
export function addSelectedCriterionId(ids: number[], id: number): number[] {
  return ids.includes(id) ? ids : [...ids, id];
}

export const getCriteriaTypeLabel = (type: CriteriaType): string => {
  switch (type) {
    case 'speed':
      return 'Скорость';
    case 'photo':
      return 'Фото';
    case 'beer':
      return 'Пиво';
    case 'random':
      return 'Рандом';
    case 'custom':
      return 'Кастомный';
    default:
      return type;
  }
};

export const getCriteriaColor = (type: CriteriaType): BadgeColor => {
  switch (type) {
    case 'speed':
      return 'success';
    case 'photo':
      return 'info';
    case 'beer':
      return 'warning';
    case 'random':
      return 'primary';
    case 'custom':
      return 'light';
    default:
      return 'light';
  }
};
