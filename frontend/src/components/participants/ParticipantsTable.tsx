'use client';

import React from 'react';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '../ui/table';
import type { Participant } from '@/types';
import type { ParticipantColumn } from './participantColumns';

interface ParticipantsTableProps {
  participants: Participant[];
  /** Разрешённые видимые колонки (в порядке отображения). */
  columns: ParticipantColumn[];
  isLoading?: boolean;
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
    <div className="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-white/[0.05] dark:bg-white/[0.03]">
      {/* Натуральная ширина + горизонтальный скролл: число колонок переменное. */}
      <div className="max-w-full overflow-x-auto">
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
                  {column.label}
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
