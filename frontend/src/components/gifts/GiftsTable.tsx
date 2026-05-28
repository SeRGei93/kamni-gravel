'use client';

import React, { useMemo, useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '../ui/table';
import Badge from '../ui/badge/Badge';
import Button from '../ui/button/Button';
import { TrashBinIcon, CheckLineIcon } from '@/icons';
import { getCriteriaColor } from '@/utils/criteria';
import { formatGiftPlaceRule } from '@/utils/giftPlaceRule';
import type { Gift } from '@/types';
import { useGiftPhotoUrls } from './useGiftPhotoUrls';

interface GiftsTableProps {
  gifts: Gift[];
  assignedGiftIds?: Set<number>;
  isLoading?: boolean;
  onApprove?: (gift: Gift) => Promise<void>;
  onDelete?: (giftId: number) => void;
  editQueryString?: string;
}

export default function GiftsTable({
  gifts,
  assignedGiftIds,
  isLoading,
  onApprove,
  onDelete,
  editQueryString,
}: GiftsTableProps) {
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [approvingId, setApprovingId] = useState<number | null>(null);

  const handleDelete = async (giftId: number) => {
    if (!confirm('Вы уверены, что хотите удалить этот приз?')) {
      return;
    }

    setDeletingId(giftId);
    try {
      if (onDelete) {
        await onDelete(giftId);
      }
    } finally {
      setDeletingId(null);
    }
  };

  const handleApprove = async (gift: Gift) => {
    if (!onApprove) {
      return;
    }

    setApprovingId(gift.id);
    try {
      await onApprove(gift);
    } finally {
      setApprovingId(null);
    }
  };

  const photoUrlTargets = useMemo(
    () =>
      gifts.flatMap((gift) => {
        const photo = gift.attachments?.find(
          (attachment) => attachment.file_type === 'photo'
        );
        return photo ? [{ giftId: gift.id, attachment: photo }] : [];
      }),
    [gifts]
  );
  const photoUrls = useGiftPhotoUrls(photoUrlTargets);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">Загрузка...</div>
      </div>
    );
  }

  if (gifts.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">Призы не найдены</div>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-white/[0.05] dark:bg-white/[0.03]">
      <div className="max-w-full overflow-x-auto">
        <div className="min-w-[1220px]">
          <Table>
            <TableHeader className="border-b border-gray-100 dark:border-white/[0.05]">
              <TableRow>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Фото
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
                  От кого
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
                  Правило
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Критерии
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Распределен
                </TableCell>
                <TableCell
                  isHeader
                  className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400"
                >
                  Дата
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
              {gifts.map((gift) => {
                const firstPhoto = gift.attachments?.find(
                  (a) => a.file_type === 'photo'
                );
                const photoUrl = firstPhoto
                  ? photoUrls[firstPhoto.id]?.url || null
                  : null;
                const isPendingReview = gift.review_status === 'pending_review';

                return (
                  <TableRow
                    key={gift.id}
                    className="hover:bg-gray-50 dark:hover:bg-white/5"
                  >
                    <TableCell className="px-5 py-4">
                      <div className="flex items-center gap-2">
                        {photoUrl ? (
                          <Image
                            src={photoUrl}
                            alt="Приз"
                            width={60}
                            height={60}
                            className="rounded-lg object-cover"
                          />
                        ) : firstPhoto ? (
                          <div className="flex h-[60px] w-[60px] items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-800">
                            <span className="text-xs text-gray-500 dark:text-gray-400">
                              📷
                            </span>
                          </div>
                        ) : (
                          <div className="flex h-[60px] w-[60px] items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-800">
                            <span className="text-xs text-gray-500 dark:text-gray-400">
                              📦
                            </span>
                          </div>
                        )}
                        {gift.attachments && gift.attachments.length > 1 && (
                          <span className="text-xs text-gray-500 dark:text-gray-400">
                            +{gift.attachments.length - 1}
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="px-5 py-4 text-start">
                      <p className="max-w-md text-sm text-gray-800 dark:text-white/90">
                        {gift.description}
                      </p>
                    </TableCell>
                    <TableCell className="px-5 py-4 text-start">
                      <div>
                        <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                          {gift.first_name} {gift.last_name}
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-400">
                          @{gift.username || `user${gift.user_id}`}
                        </p>
                      </div>
                    </TableCell>
                    <TableCell className="px-5 py-4 text-start">
                      <Badge
                        color={isPendingReview ? 'warning' : 'success'}
                        size="sm"
                      >
                        {isPendingReview ? 'Новый / на проверке' : 'Проверен'}
                      </Badge>
                    </TableCell>
                    <TableCell className="px-5 py-4 text-start">
                      <span className="text-sm text-gray-700 dark:text-gray-300">
                        {formatGiftPlaceRule(gift.place_rule ?? (gift.place ? { type: 'places', places: [gift.place] } : null))}
                      </span>
                    </TableCell>
                    <TableCell className="px-5 py-4 text-start">
                      {gift.criteria && gift.criteria.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {gift.criteria.slice(0, 3).map((c) => (
                            <Badge
                              key={c.id}
                              color={getCriteriaColor(c.criteria_type)}
                              size="sm"
                            >
                              {c.name}
                            </Badge>
                          ))}
                          {gift.criteria.length > 3 && (
                            <Badge color="light" size="sm">
                              +{gift.criteria.length - 3}
                            </Badge>
                          )}
                        </div>
                      ) : (
                        <span className="text-sm text-gray-500 dark:text-gray-400">
                          Без критериев
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="px-5 py-4 text-start">
                      <div className="flex items-center gap-2">
                        <div className={`h-2.5 w-2.5 rounded-full ${
                          isPendingReview
                            ? 'bg-warning-500'
                            : assignedGiftIds && assignedGiftIds.has(gift.id)
                            ? 'bg-success-500'
                            : 'bg-error-500'
                        }`} />
                        <span className="text-sm text-gray-800 dark:text-white/90">
                          {isPendingReview
                            ? 'На проверке / Не участвует'
                            : assignedGiftIds && assignedGiftIds.has(gift.id)
                              ? 'Да'
                              : 'Нет'}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="px-5 py-4 text-start">
                      <span className="text-sm text-gray-800 dark:text-white/90">
                        {new Date(gift.created_at).toLocaleDateString('ru-RU')}
                      </span>
                    </TableCell>
                    <TableCell className="px-5 py-4 text-start">
                      <div className="flex items-center gap-2">
                        {isPendingReview && onApprove && (
                          <Button
                            size="sm"
                            startIcon={<CheckLineIcon />}
                            onClick={() => handleApprove(gift)}
                            disabled={approvingId === gift.id}
                          >
                            {approvingId === gift.id ? '...' : 'Проверить'}
                          </Button>
                        )}
                        <Link
                          href={`/gifts/${gift.id}${
                            editQueryString ? `?${editQueryString}` : ''
                          }`}
                          className="inline-flex items-center justify-center rounded-lg bg-white px-3 py-2 text-sm font-medium text-gray-700 ring-1 ring-inset ring-gray-300 transition hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-400 dark:ring-gray-700 dark:hover:bg-white/[0.03] dark:hover:text-gray-300"
                        >
                            Редактировать
                        </Link>
                        {onDelete && (
                          <Button
                            size="sm"
                            variant="outline"
                            startIcon={<TrashBinIcon />}
                            onClick={() => handleDelete(gift.id)}
                            disabled={deletingId === gift.id}
                          >
                            {deletingId === gift.id ? '...' : 'Удалить'}
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}
