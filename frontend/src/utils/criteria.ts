import type { CriteriaType } from '@/types';

type BadgeColor = 'primary' | 'success' | 'error' | 'warning' | 'info' | 'light' | 'dark';

export const getCriteriaTypeLabel = (type: CriteriaType): string => {
  switch (type) {
    case 'speed':
      return 'Скорость';
    case 'photo':
      return 'Фото';
    case 'beer':
      return 'Пиво';
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
    case 'custom':
      return 'light';
    default:
      return 'light';
  }
};
