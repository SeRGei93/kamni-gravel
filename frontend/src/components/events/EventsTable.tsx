'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '../ui/table';
import Badge from '../ui/badge/Badge';
import Button from '../ui/button/Button';
import { TrashBinIcon } from '@/icons';
import type { Event } from '@/types';
import { formatMinskDateTime } from '@/utils/minskTime';

interface EventsTableProps {
  events: Event[];
  isLoading?: boolean;
  onDelete?: (eventId: number) => void;
}

export default function EventsTable({
  events,
  isLoading,
  onDelete,
}: EventsTableProps) {
  const [deletingId, setDeletingId] = useState<number | null>(null);

  const handleDelete = async (eventId: number) => {
    if (!confirm('Вы уверены, что хотите удалить это событие? Все связанные данные будут сохранены.')) {
      return;
    }

    setDeletingId(eventId);
    try {
      if (onDelete) {
        await onDelete(eventId);
      }
    } finally {
      setDeletingId(null);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">Загрузка...</div>
      </div>
    );
  }

  if (events.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">События не найдены</div>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-white/[0.05] dark:bg-white/[0.03]">
      <div className="max-w-full overflow-x-auto">
        <Table>
          <TableHeader className="border-b border-gray-100 dark:border-white/[0.05]">
            <TableRow>
              <TableCell
                isHeader
                className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
              >
                Название
              </TableCell>
              <TableCell
                isHeader
                className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
              >
                Описание
              </TableCell>
              <TableCell
                isHeader
                className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
              >
                Старт
              </TableCell>
              <TableCell
                isHeader
                className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
              >
                Финиш
              </TableCell>
              <TableCell
                isHeader
                className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
              >
                Статус
              </TableCell>
              <TableCell
                isHeader
                className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
              >
                Telegram
              </TableCell>
              <TableCell
                isHeader
                className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
              >
                Действия
              </TableCell>
            </TableRow>
          </TableHeader>

          <TableBody className="divide-y divide-gray-100 dark:divide-white/[0.05]">
            {events.map((event) => (
              <TableRow
                key={event.id}
                className="hover:bg-gray-50 dark:hover:bg-white/5"
              >
                <TableCell className="px-5 py-4 text-start">
                  <Link
                    href={`/events/${event.id}`}
                    className="font-medium text-brand-500 hover:text-brand-600 dark:text-brand-400 dark:hover:text-brand-300"
                  >
                    {event.name}
                  </Link>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <p className="max-w-md text-sm text-gray-800 dark:text-white/90 line-clamp-2">
                    {event.description || '-'}
                  </p>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <div className="text-sm text-gray-800 dark:text-white/90">
                    {formatMinskDateTime(event.start_date)}
                  </div>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <div className="text-sm text-gray-800 dark:text-white/90">
                    {formatMinskDateTime(event.end_date)}
                  </div>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <Badge color={event.active ? 'success' : 'light'} size="sm">
                    {event.active ? 'Активно' : 'Неактивно'}
                  </Badge>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge
                      color={
                        Object.values(event.telegram_texts || {}).some(Boolean)
                          ? 'info'
                          : 'light'
                      }
                      size="sm"
                    >
                      {Object.values(event.telegram_texts || {}).some(Boolean)
                        ? 'Тексты'
                        : 'По умолчанию'}
                    </Badge>
                    <Link
                      href={`/events/${event.id}/telegram-texts`}
                      className="inline-flex items-center justify-center rounded-lg bg-white px-3 py-2 text-xs font-medium text-gray-700 ring-1 ring-inset ring-gray-300 transition hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-400 dark:ring-gray-700 dark:hover:bg-white/[0.03] dark:hover:text-gray-300"
                    >
                      Тексты
                    </Link>
                  </div>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <div className="flex items-center gap-2">
                    <Link
                      href={`/events/${event.id}`}
                      className="inline-flex items-center justify-center rounded-lg bg-white px-3 py-2 text-sm font-medium text-gray-700 ring-1 ring-inset ring-gray-300 transition hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-400 dark:ring-gray-700 dark:hover:bg-white/[0.03] dark:hover:text-gray-300"
                    >
                      Редактировать
                    </Link>
                    {onDelete && (
                      <Button
                        size="sm"
                        variant="outline"
                        startIcon={<TrashBinIcon />}
                        onClick={() => handleDelete(event.id)}
                        disabled={deletingId === event.id}
                      >
                        {deletingId === event.id ? '...' : 'Удалить'}
                      </Button>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
