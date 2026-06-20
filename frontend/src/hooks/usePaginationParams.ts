'use client';

import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useCallback } from 'react';

// Размер страницы настраивается пользователем, но ограничен диапазоном [50, 100]
// (то же ограничение, что и на бэкенде).
export const PAGE_SIZE_OPTIONS = [50, 100] as const;
export const DEFAULT_PAGE_SIZE = 50;
const MIN_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 100;

function clampPageSize(value: number): number {
  if (!Number.isFinite(value)) return DEFAULT_PAGE_SIZE;
  if (value < MIN_PAGE_SIZE) return MIN_PAGE_SIZE;
  if (value > MAX_PAGE_SIZE) return MAX_PAGE_SIZE;
  return value;
}

export type PaginationParams = {
  page: number;
  pageSize: number;
  setPage: (page: number) => void;
  setPageSize: (size: number) => void;
};

/**
 * Читает page/page_size из URL и даёт сеттеры, которые обновляют URL,
 * сохраняя остальные query-параметры. Смена размера страницы сбрасывает на стр. 1.
 */
export function usePaginationParams(): PaginationParams {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const pageRaw = parseInt(searchParams.get('page') ?? '', 10);
  const page = Number.isFinite(pageRaw) && pageRaw > 0 ? pageRaw : 1;

  const sizeRaw = parseInt(searchParams.get('page_size') ?? '', 10);
  const pageSize = clampPageSize(Number.isFinite(sizeRaw) ? sizeRaw : DEFAULT_PAGE_SIZE);

  const setParams = useCallback(
    (next: { page?: number; pageSize?: number }) => {
      const params = new URLSearchParams(searchParams.toString());
      if (next.pageSize !== undefined) {
        params.set('page_size', String(clampPageSize(next.pageSize)));
      }
      if (next.page !== undefined) {
        params.set('page', String(Math.max(1, next.page)));
      }
      const query = params.toString();
      router.replace(`${pathname}${query ? `?${query}` : ''}`, { scroll: false });
    },
    [pathname, router, searchParams]
  );

  const setPage = useCallback((p: number) => setParams({ page: p }), [setParams]);
  const setPageSize = useCallback(
    (s: number) => setParams({ page: 1, pageSize: s }),
    [setParams]
  );

  return { page, pageSize, setPage, setPageSize };
}
