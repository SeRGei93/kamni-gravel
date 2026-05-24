'use client';

import React, { useState } from 'react';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '../ui/table';
import Badge from '../ui/badge/Badge';
import Button from '../ui/button/Button';
import { PencilIcon, TrashBinIcon } from '@/icons';
import { getCriteriaTypeLabel, getCriteriaColor } from '@/utils/criteria';
import type { Criteria } from '@/types';

interface CriteriaTableProps {
  criteria: Criteria[];
  isLoading?: boolean;
  onEdit?: (criteria: Criteria) => void;
  onDelete?: (criteriaId: number) => void;
}

export default function CriteriaTable({
  criteria,
  isLoading,
  onEdit,
  onDelete,
}: CriteriaTableProps) {
  const [deletingId, setDeletingId] = useState<number | null>(null);

  const handleDelete = async (criteriaId: number) => {
    if (
      !confirm(
        'Вы уверены, что хотите удалить этот критерий? Все связанные данные будут сохранены.'
      )
    ) {
      return;
    }

    setDeletingId(criteriaId);
    try {
      if (onDelete) {
        await onDelete(criteriaId);
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

  if (criteria.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">
          Критерии не найдены
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
                Тип
              </TableCell>
              <TableCell
                isHeader
                className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
              >
                Дата создания
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
            {criteria.map((criterion) => (
              <TableRow
                key={criterion.id}
                className="hover:bg-gray-50 dark:hover:bg-white/5"
              >
                <TableCell className="px-5 py-4 text-start">
                  <div className="font-medium text-gray-800 dark:text-white/90">
                    {criterion.name}
                  </div>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <p className="max-w-md text-sm text-gray-800 dark:text-white/90 line-clamp-2">
                    {criterion.description || '-'}
                  </p>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <Badge color={getCriteriaColor(criterion.criteria_type)} size="sm">
                    {getCriteriaTypeLabel(criterion.criteria_type)}
                  </Badge>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <span className="text-sm text-gray-800 dark:text-white/90">
                    {new Date(criterion.created_at).toLocaleDateString('ru-RU')}
                  </span>
                </TableCell>
                <TableCell className="px-5 py-4 text-start">
                  <div className="flex items-center gap-2">
                    {onEdit && (
                      <Button
                        size="sm"
                        variant="outline"
                        startIcon={<PencilIcon />}
                        onClick={() => onEdit(criterion)}
                      >
                        Редактировать
                      </Button>
                    )}
                    {onDelete && (
                      <Button
                        size="sm"
                        variant="outline"
                        startIcon={<TrashBinIcon />}
                        onClick={() => handleDelete(criterion.id)}
                        disabled={deletingId === criterion.id}
                      >
                        {deletingId === criterion.id ? '...' : 'Удалить'}
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
