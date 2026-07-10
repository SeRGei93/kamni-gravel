'use client';

import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { useCallback } from 'react';
import type { PageSize } from '@/types';

// Размер страницы настраивается пользователем: 50, 100 или все записи.
export const PAGE_SIZE_OPTIONS = [50, 100, 'all'] as const satisfies readonly PageSize[];
export const DEFAULT_PAGE_SIZE = 50;
const MIN_PAGE_SIZE = 50;
const MAX_PAGE_SIZE = 100;

export function normalizePageSize(value: string | number | null): PageSize {
  if (value === 'all') return 'all';
  const numericValue =
    typeof value === 'number' ? value : Number.parseInt(value ?? '', 10);
  if (!Number.isFinite(numericValue)) return DEFAULT_PAGE_SIZE;
  if (numericValue < MIN_PAGE_SIZE) return MIN_PAGE_SIZE;
  if (numericValue > MAX_PAGE_SIZE) return MAX_PAGE_SIZE;
  return numericValue;
}

export type PaginationParams = {
  page: number;
  pageSize: PageSize;
  setPage: (page: number) => void;
  setPageSize: (size: PageSize) => void;
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

  const pageSize = normalizePageSize(searchParams.get('page_size'));

  const setParams = useCallback(
    (next: { page?: number; pageSize?: PageSize }) => {
      const params = new URLSearchParams(searchParams.toString());
      if (next.pageSize !== undefined) {
        params.set('page_size', String(normalizePageSize(next.pageSize)));
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
    (s: PageSize) => setParams({ page: 1, pageSize: s }),
    [setParams]
  );

  return { page, pageSize, setPage, setPageSize };
}
