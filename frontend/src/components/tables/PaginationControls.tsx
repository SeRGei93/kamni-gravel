'use client';

import { useEffect } from 'react';
import Pagination from './Pagination';
import { PAGE_SIZE_OPTIONS } from '@/hooks/usePaginationParams';
import type { PageSize } from '@/types';

type PaginationControlsProps = {
  total: number;
  page: number;
  pageSize: PageSize;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: PageSize) => void;
};

/**
 * Полоса управления серверной пагинацией: счётчик «показано X–Y из Z»,
 * селектор размера страницы (50/100/Все) и постраничная навигация.
 */
export default function PaginationControls({
  total,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
}: PaginationControlsProps) {
  const showAll = pageSize === 'all';
  const numericPageSize = showAll ? Math.max(total, 1) : pageSize;
  const hasLegacyPageSize =
    typeof pageSize === 'number' &&
    !PAGE_SIZE_OPTIONS.some((option) => option === pageSize);
  const totalPages = showAll ? 1 : Math.max(1, Math.ceil(total / numericPageSize));

  // Если текущая страница вышла за пределы (напр. устаревший ?page= в URL или
  // сменился размер страницы) — возвращаем на последнюю валидную страницу.
  useEffect(() => {
    if (total > 0 && page > totalPages) {
      onPageChange(totalPages);
    }
  }, [total, totalPages, page, onPageChange]);

  const safePage = Math.min(page, totalPages);
  const from = total === 0 ? 0 : showAll ? 1 : (safePage - 1) * numericPageSize + 1;
  const to = showAll ? total : Math.min(safePage * numericPageSize, total);

  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-4">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          {total === 0 ? 'Ничего не найдено' : `Показано ${from}–${to} из ${total}`}
        </p>
        <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-400">
          На странице:
          <select
            value={pageSize}
            onChange={(e) =>
              onPageSizeChange(e.target.value === 'all' ? 'all' : Number(e.target.value))
            }
            className="rounded-lg border border-gray-300 bg-white px-2 py-1 text-sm text-gray-700 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300"
          >
            {hasLegacyPageSize && <option value={pageSize}>{pageSize}</option>}
            {PAGE_SIZE_OPTIONS.map((opt) => (
              <option key={opt} value={opt}>
                {opt === 'all' ? 'Все' : opt}
              </option>
            ))}
          </select>
        </label>
      </div>
      {totalPages > 1 && (
        <Pagination
          currentPage={safePage}
          totalPages={totalPages}
          onPageChange={onPageChange}
        />
      )}
    </div>
  );
}
