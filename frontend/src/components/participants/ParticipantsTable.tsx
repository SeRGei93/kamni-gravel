'use client';

import React from 'react';
import Link from 'next/link';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '../ui/table';
import Badge from '../ui/badge/Badge';
import type { Participant } from '@/types';

interface ParticipantsTableProps {
  participants: Participant[];
  isLoading?: boolean;
}

const GENDER_LABELS: Record<string, string> = {
  male: 'М',
  female: 'Ж',
};

const BIKE_TYPE_LABELS: Record<string, string> = {
  gravel: 'Гравийник',
  mtb: 'МТБ',
  road: 'Шоссе',
  single_speed: 'Фикс',
  tandem: 'Тандем',
};

export default function ParticipantsTable({
  participants,
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
      <div className="max-w-full overflow-x-auto">
        <div className="min-w-[1200px]">
          <Table>
            <TableHeader className="border-b border-gray-100 dark:border-white/[0.05]">
              <TableRow>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Место
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Username
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Имя
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Пол
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Велосипед
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Общее время
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Время в пути
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Приз
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Призы
                </TableCell>
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
                  <TableCell className="px-5 py-4 text-start">
                    <span className="font-medium text-gray-800 text-theme-sm dark:text-white/90">
                      {participant.place && participant.place > 0
                        ? participant.place
                        : '-'}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-4 text-start">
                    <Link
                      href={`/participants/${participant.id}`}
                      className="font-medium text-brand-500 hover:text-brand-600 dark:text-brand-400 dark:hover:text-brand-300"
                    >
                      {participant.username || `@user${participant.user_id}`}
                    </Link>
                  </TableCell>
                  <TableCell className="px-5 py-4 text-start">
                    <span className="text-gray-800 text-theme-sm dark:text-white/90">
                      {participant.first_name} {participant.last_name}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-4 text-start">
                    <Badge
                      color={participant.gender === 'male' ? 'info' : 'warning'}
                      size="sm"
                    >
                      {GENDER_LABELS[participant.gender] || participant.gender}
                    </Badge>
                  </TableCell>
                  <TableCell className="px-5 py-4 text-start">
                    <Badge color="light" size="sm">
                      {BIKE_TYPE_LABELS[participant.bike_type] ||
                        participant.bike_type}
                    </Badge>
                  </TableCell>
                  <TableCell className="px-5 py-4 text-start">
                    <span className="text-gray-800 text-theme-sm dark:text-white/90">
                      {participant.elapsed_time || '-'}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-4 text-start">
                    <span className="text-gray-800 text-theme-sm dark:text-white/90">
                      {participant.moving_time || '-'}
                    </span>
                  </TableCell>
                  <TableCell className="px-5 py-4 text-start">
                    {participant.has_gift ? (
                      <Badge color="success" size="sm">
                        Да
                      </Badge>
                    ) : (
                      <Badge color="light" size="sm">
                        Нет
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="px-5 py-4 text-start">
                    {participant.prizes_count > 0 ? (
                      <Badge color="warning" size="sm">
                        {participant.prizes_count}
                      </Badge>
                    ) : (
                      <span className="text-gray-500 text-theme-sm dark:text-gray-400">
                        -
                      </span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}
