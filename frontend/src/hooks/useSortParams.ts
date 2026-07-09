'use client';

import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useCallback } from 'react';
import type { SortOrder } from '@/api/participants';

export type SortParams = {
  sortKey: string | null;
  sortOrder: SortOrder;
  /**
   * Задаёт сортировку. key === null сбрасывает её (порядок по умолчанию).
   * Смена сортировки сбрасывает пагинацию на первую страницу.
   */
  setSort: (key: string | null, order?: SortOrder) => void;
};

/**
 * Хранит состояние сортировки списка в URL (?sort=&order=), чтобы оно
 * переживало перезагрузку страницы и было шарящимся ссылкой. Остальные
 * query-параметры сохраняются; смена сортировки сбрасывает страницу на 1.
 */
export function useSortParams(): SortParams {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const sortKey = searchParams.get('sort') || null;
  const sortOrder: SortOrder = searchParams.get('order') === 'desc' ? 'desc' : 'asc';

  const setSort = useCallback(
    (key: string | null, order: SortOrder = 'asc') => {
      const params = new URLSearchParams(searchParams.toString());
      if (key) {
        params.set('sort', key);
        params.set('order', order === 'desc' ? 'desc' : 'asc');
      } else {
        params.delete('sort');
        params.delete('order');
      }
      // Смена сортировки — возвращаемся на первую страницу.
      params.set('page', '1');
      const query = params.toString();
      router.replace(`${pathname}${query ? `?${query}` : ''}`, { scroll: false });
    },
    [pathname, router, searchParams]
  );

  return { sortKey, sortOrder, setSort };
}
