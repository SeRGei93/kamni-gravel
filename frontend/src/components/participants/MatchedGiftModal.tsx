'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { giftsApi } from '@/api/gifts';
import GiftPhotoPreviewGrid from '@/components/gifts/GiftPhotoPreviewGrid';
import Badge from '@/components/ui/badge/Badge';
import { Modal } from '@/components/ui/modal';
import { getCriteriaColor } from '@/utils/criteria';
import type { Gift } from '@/types';

const GENDER_FILTER_LABELS: Record<string, string> = {
  male: 'Мужской зачёт',
  female: 'Женский зачёт',
};

const BIKE_TYPE_FILTER_LABELS: Record<string, string> = {
  gravel: 'Гравийник',
  mtb: 'МТБ',
  road: 'Шоссе',
  single_speed: 'Фикс',
  tandem: 'Тандем',
};

interface MatchedGiftModalProps {
  /** ID приза; null — модалка закрыта. */
  giftId: number | null;
  /** Подпись о назначении («выдано месту N» и т.п.) из карточки-источника. */
  note?: string;
  onClose: () => void;
}

// Модалка с деталями подобранного приза. Данные в matched-призах неполные
// (без фото и дарителя), поэтому приз догружается по id при открытии.
export default function MatchedGiftModal({ giftId, note, onClose }: MatchedGiftModalProps) {
  return (
    <Modal isOpen={giftId !== null} onClose={onClose} className="max-w-2xl m-4">
      {giftId !== null && (
        // key: при смене приза контент размонтируется — состояние загрузки сбрасывается
        <MatchedGiftModalContent key={giftId} giftId={giftId} note={note} />
      )}
    </Modal>
  );
}

function MatchedGiftModalContent({ giftId, note }: { giftId: number; note?: string }) {
  const [gift, setGift] = useState<Gift | null>(null);
  const [error, setError] = useState<string | null>(null);
  const isLoading = !gift && !error;

  useEffect(() => {
    let cancelled = false;
    giftsApi
      .getById(giftId)
      .then((loaded) => {
        if (!cancelled) setGift(loaded);
      })
      .catch((err) => {
        if (!cancelled) setError('Не удалось загрузить приз');
        console.error('Failed to load matched gift details:', {
          gift_id: giftId,
          operation: 'load_matched_gift_modal',
          error: err,
        });
      });
    return () => {
      cancelled = true;
    };
  }, [giftId]);

  const donorName = gift
    ? [gift.first_name, gift.last_name].filter(Boolean).join(' ') ||
      gift.username ||
      `user${gift.user_id}`
    : '';

  return (
    <div className="max-h-[85vh] overflow-y-auto p-5 lg:p-8">
      <h3 className="mb-4 pr-10 text-lg font-semibold text-gray-800 dark:text-white">
        Приз
      </h3>

      {isLoading && (
        <p className="py-8 text-center text-sm text-gray-500 dark:text-gray-400">
          Загрузка...
        </p>
      )}

      {error && (
        <p className="py-8 text-center text-sm text-error-600 dark:text-error-400">
          {error}
        </p>
      )}

      {gift && (
        <div className="space-y-4">
          <p className="text-sm font-medium text-gray-800 dark:text-white/90">
            {gift.description}
          </p>

          {((gift.criteria && gift.criteria.length > 0) ||
            (gift.gender_filter && gift.gender_filter !== 'all') ||
            (gift.bike_type_filter && gift.bike_type_filter !== 'all')) && (
            <div className="flex flex-wrap gap-1.5">
              {gift.criteria?.map((c) => (
                <Badge key={c.id} color={getCriteriaColor(c.criteria_type)} size="sm">
                  {c.name}
                </Badge>
              ))}
              {gift.gender_filter && gift.gender_filter !== 'all' && (
                <Badge color="info" size="sm">
                  {GENDER_FILTER_LABELS[gift.gender_filter] || gift.gender_filter}
                </Badge>
              )}
              {gift.bike_type_filter && gift.bike_type_filter !== 'all' && (
                <Badge color="info" size="sm">
                  {BIKE_TYPE_FILTER_LABELS[gift.bike_type_filter] || gift.bike_type_filter}
                </Badge>
              )}
            </div>
          )}

          {note && <p className="text-xs text-gray-500 dark:text-gray-400">{note}</p>}

          <GiftPhotoPreviewGrid gift={gift} />

          <div className="flex items-center justify-between gap-3 border-t border-gray-200 pt-4 dark:border-gray-700">
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Даритель:{' '}
              <span className="font-medium text-gray-800 dark:text-white/90">
                {donorName}
              </span>
              {gift.username ? ` (@${gift.username})` : ''}
            </p>
            <Link
              href={`/gifts/${gift.id}`}
              className="shrink-0 text-sm font-medium text-brand-500 hover:text-brand-600 dark:text-brand-400 dark:hover:text-brand-300"
            >
              Страница приза
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
