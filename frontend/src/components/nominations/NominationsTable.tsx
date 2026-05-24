'use client';

import React, { useState } from 'react';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '../ui/table';
import Badge from '../ui/badge/Badge';
import Button from '../ui/button/Button';
import { PencilIcon, TrashBinIcon, ArrowUpIcon, ArrowDownIcon } from '@/icons';
import type { Nomination } from '@/types';

interface NominationsTableProps {
  nominations: Nomination[];
  isLoading?: boolean;
  onEdit?: (nomination: Nomination) => void;
  onDelete?: (nominationId: number) => void;
  onMoveUp?: (nominationId: number) => void;
  onMoveDown?: (nominationId: number) => void;
}

const getGenderFilterLabel = (filter: string): string => {
  switch (filter) {
    case 'all':
      return 'Любой';
    case 'male':
      return 'Мужской';
    case 'female':
      return 'Женский';
    default:
      return filter;
  }
};

const getBikeTypeFilterLabel = (filter: string): string => {
  switch (filter) {
    case 'all':
      return 'Любой';
    case 'gravel':
      return 'Гравийник';
    case 'mtb':
      return 'МТБ';
    case 'road':
      return 'Шоссе';
    case 'single_speed':
      return 'Фикс';
    case 'tandem':
      return 'Тандем';
    default:
      return filter;
  }
};

export default function NominationsTable({
  nominations,
  isLoading,
  onEdit,
  onDelete,
  onMoveUp,
  onMoveDown,
}: NominationsTableProps) {
  const [deletingId, setDeletingId] = useState<number | null>(null);

  // Сортируем номинации по sort_order
  const sortedNominations = [...nominations].sort(
    (a, b) => a.sort_order - b.sort_order
  );

  const handleDelete = async (nominationId: number) => {
    if (
      !confirm(
        'Вы уверены, что хотите удалить эту номинацию? Все связанные данные будут сохранены.'
      )
    ) {
      return;
    }

    setDeletingId(nominationId);
    try {
      if (onDelete) {
        await onDelete(nominationId);
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

  if (nominations.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">
          Номинации не найдены
        </div>
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
                Порядок
              </TableCell>
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
                Фильтры
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
                Действия
              </TableCell>
            </TableRow>
          </TableHeader>

          <TableBody className="divide-y divide-gray-100 dark:divide-white/[0.05]">
            {sortedNominations.map((nomination, index) => (
              <TableRow
                key={nomination.id}
                className="hover:bg-gray-50 dark:hover:bg-white/5"
              >
                <TableCell className="px-5 py-4 text-start">
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-gray-800 dark:text-white/90">
                      {nomination.sort_order}
                    </span>
                    <div className="flex flex-col gap-1">
                      {onMoveUp && index > 0 && (
                        <Button
                          size="xs"
                          variant="outline"
                          onClick={() => onMoveUp(nomination.id)}
                          className="h-6 w-6 p-0"
                        >
                          <ArrowUpIcon className="h-3 w-3" />
                        </Button>
                      )}
                      {onMoveDown && index < sortedNominations.length - 1 && (
                        <Button
                          size="xs"
                          variant="outline"
                          onClick={() => onMoveDown(nomination.id)}
                          className="h-6 w-6 p-0"
                        >
                          <ArrowDownIcon className="h-3 w-3" />
                        </Button>
                      )}
                    </div>
                  </div>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <div className="font-medium text-gray-800 dark:text-white/90">
                    {nomination.name}
                  </div>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <p className="max-w-md text-sm text-gray-800 dark:text-white/90 line-clamp-2">
                    {nomination.description || '-'}
                  </p>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <div className="flex flex-wrap gap-2">
                    <Badge color="info" size="sm">
                      {getGenderFilterLabel(nomination.gender_filter)}
                    </Badge>
                    <Badge color="success" size="sm">
                      {getBikeTypeFilterLabel(nomination.bike_type_filter)}
                    </Badge>
                  </div>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <Badge
                    color={nomination.is_active ? 'success' : 'light'}
                    size="sm"
                  >
                    {nomination.is_active ? 'Активна' : 'Неактивна'}
                  </Badge>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <div className="flex items-center gap-2">
                    {onEdit && (
                      <Button
                        size="sm"
                        variant="outline"
                        startIcon={<PencilIcon />}
                        onClick={() => onEdit(nomination)}
                      >
                        Редактировать
                      </Button>
                    )}
                    {onDelete && (
                      <Button
                        size="sm"
                        variant="outline"
                        startIcon={<TrashBinIcon />}
                        onClick={() => handleDelete(nomination.id)}
                        disabled={deletingId === nomination.id}
                      >
                        {deletingId === nomination.id ? '...' : 'Удалить'}
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
