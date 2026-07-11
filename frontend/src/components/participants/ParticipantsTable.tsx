'use client';

import React from 'react';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '../ui/table';
import { ArrowUpIcon, ArrowDownIcon } from '@/icons';
import type { SortOrder } from '@/api/participants';
import type { Participant } from '@/types';
import { isSortableColumn, type ParticipantColumn } from './participantColumns';

interface ParticipantsTableProps {
  participants: Participant[];
  /** Разрешённые видимые колонки (в порядке отображения). */
  columns: ParticipantColumn[];
  isLoading?: boolean;
  /** Ключ активной колонки сортировки (null — сортировки нет). */
  sortKey?: string | null;
  /** Направление активной сортировки. */
  sortOrder?: SortOrder;
  /** Клик по управлению сортировкой в шапке (page реализует тристейт). */
  onSortChange?: (key: string) => void;
}

function alignClass(align?: ParticipantColumn['align']): string {
  if (align === 'end') return 'text-end';
  if (align === 'center') return 'text-center';
  return 'text-start';
}

export default function ParticipantsTable({
  participants,
  columns,
  isLoading,
  sortKey = null,
  sortOrder = 'asc',
  onSortChange,
}: ParticipantsTableProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">Загрузка...</div>
      </div>
    );
  }

  if (participants.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">
          Участники не найдены
        </div>
      </div>
    );
  }

  return (
    <div className="w-full max-w-full overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-white/[0.05] dark:bg-white/[0.03]">
      {/* Натуральная ширина + горизонтальный скролл: число колонок переменное. */}
      <div className="w-full max-w-full overflow-x-auto">
        <Table>
          <TableHeader className="border-b border-gray-100 dark:border-white/[0.05]">
            <TableRow>
              {columns.map((column) => (
                <TableCell
                  key={column.key}
                  isHeader
                  className={`whitespace-nowrap px-5 py-3 font-medium text-gray-500 ${alignClass(
                    column.align,
                  )} text-theme-xs dark:text-gray-400`}
                >
                  <ColumnHeader
                    column={column}
                    isActive={sortKey === column.key}
                    sortOrder={sortOrder}
                    onSortChange={onSortChange}
                  />
                </TableCell>
              ))}
            </TableRow>
          </TableHeader>

          <TableBody className="divide-y divide-gray-100 dark:divide-white/[0.05]">
            {participants.map((participant) => (
              <TableRow
                key={participant.id}
                className={`${
                  participant.is_finished
                    ? 'bg-green-50/50 dark:bg-green-900/10'
                    : ''
                } hover:bg-gray-50 dark:hover:bg-white/5`}
              >
                {columns.map((column) => (
                  <TableCell
                    key={column.key}
                    className={`whitespace-nowrap px-5 py-4 ${alignClass(
                      column.align,
                    )}`}
                  >
                    {column.render(participant)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

/** Заголовок колонки: обычный текст или кнопка сортировки с иконкой. */
function ColumnHeader({
  column,
  isActive,
  sortOrder,
  onSortChange,
}: {
  column: ParticipantColumn;
  isActive: boolean;
  sortOrder: SortOrder;
  onSortChange?: (key: string) => void;
}) {
  if (!onSortChange || !isSortableColumn(column.key)) {
    return <>{column.label}</>;
  }

  const stateLabel = isActive
    ? sortOrder === 'asc'
      ? 'по возрастанию'
      : 'по убыванию'
    : 'не отсортировано';

  return (
    <button
      type="button"
      onClick={() => onSortChange(column.key)}
      title={`Сортировать по «${column.label}»`}
      aria-label={`Сортировать по «${column.label}» (${stateLabel})`}
      className={`inline-flex select-none items-center gap-1 transition-colors hover:text-gray-700 dark:hover:text-gray-200 ${
        isActive ? 'text-gray-700 dark:text-gray-200' : ''
      }`}
    >
      <span>{column.label}</span>
      <SortIndicator isActive={isActive} sortOrder={sortOrder} />
    </button>
  );
}

function SortIndicator({
  isActive,
  sortOrder,
}: {
  isActive: boolean;
  sortOrder: SortOrder;
}) {
  if (!isActive) {
    return <ArrowDownIcon className="size-3.5 opacity-30" aria-hidden="true" />;
  }
  return sortOrder === 'asc' ? (
    <ArrowUpIcon className="size-3.5" aria-hidden="true" />
  ) : (
    <ArrowDownIcon className="size-3.5" aria-hidden="true" />
  );
}
