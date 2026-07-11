'use client';

import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useCallback } from 'react';
import type { HasGiftFilter } from '@/utils/participants';

export type ParticipantUrlFilters = {
  gender: string; // '' | 'male' | 'female'
  bikeType: string; // '' | 'gravel' | ...
  isFinished: string; // '' | 'true' | 'false'
  hasGift: HasGiftFilter; // 'all' | 'yes' | 'no'
  criteriaId: string; // '' | положительный ID критерия
  q: string; // поисковый запрос
};

export type FilterParams = ParticipantUrlFilters & {
  /** Обновляет часть фильтров в URL. Любое изменение сбрасывает на стр. 1. */
  setFilters: (partial: Partial<ParticipantUrlFilters>) => void;
};

// Соответствие «поле фильтра → имя query-параметра в URL».
const PARAM_KEYS: Record<keyof ParticipantUrlFilters, string> = {
  gender: 'gender',
  bikeType: 'bike_type',
  isFinished: 'is_finished',
  hasGift: 'has_gift',
  criteriaId: 'criteria_id',
  q: 'q',
};

const HAS_GIFT_VALUES: readonly string[] = ['all', 'yes', 'no'];

function normalizeHasGift(value: string | null): HasGiftFilter {
  return value && HAS_GIFT_VALUES.includes(value) ? (value as HasGiftFilter) : 'all';
}

// Значение, при котором параметр считается «пустым» и удаляется из URL,
// чтобы адрес оставался чистым при фильтрах по умолчанию.
function isEmptyValue(field: keyof ParticipantUrlFilters, value: string): boolean {
  if (field === 'hasGift') return value === 'all' || value === '';
  return value === '';
}

/**
 * Хранит фильтры списка участников в URL
 * (gender/bike_type/is_finished/has_gift/criteria_id/q),
 * чтобы они переживали перезагрузку страницы и были шарящимися ссылкой. Остальные
 * query-параметры сохраняются; изменение фильтра сбрасывает страницу на 1.
 */
export function useFilterParams(): FilterParams {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const gender = searchParams.get('gender') ?? '';
  const bikeType = searchParams.get('bike_type') ?? '';
  const isFinished = searchParams.get('is_finished') ?? '';
  const hasGift = normalizeHasGift(searchParams.get('has_gift'));
  const criteriaId = searchParams.get('criteria_id') ?? '';
  const q = searchParams.get('q') ?? '';

  const setFilters = useCallback(
    (partial: Partial<ParticipantUrlFilters>) => {
      const params = new URLSearchParams(searchParams.toString());
      (Object.keys(partial) as (keyof ParticipantUrlFilters)[]).forEach((field) => {
        const value = String(partial[field] ?? '');
        const paramKey = PARAM_KEYS[field];
        if (isEmptyValue(field, value)) {
          params.delete(paramKey);
        } else {
          params.set(paramKey, value);
        }
      });
      // Изменение фильтра — возвращаемся на первую страницу.
      params.set('page', '1');
      const query = params.toString();
      router.replace(`${pathname}${query ? `?${query}` : ''}`, { scroll: false });
    },
    [pathname, router, searchParams]
  );

  return { gender, bikeType, isFinished, hasGift, criteriaId, q, setFilters };
}
