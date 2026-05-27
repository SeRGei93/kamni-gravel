'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import { criteriaApi } from '@/api/criteria';
import { giftsApi } from '@/api/gifts';
import GiftEditForm from '@/components/gifts/GiftEditForm';
import GiftPhotoPreviewGrid from '@/components/gifts/GiftPhotoPreviewGrid';
import Badge from '@/components/ui/badge/Badge';
import type { Criteria, Gift, UpdateGiftRequest } from '@/types';

function getGiftListHref(searchParams: URLSearchParams): string {
  const params = new URLSearchParams();
  const eventId = searchParams.get('event_id');
  const reviewStatus = searchParams.get('review_status');

  if (eventId && /^\d+$/.test(eventId)) {
    params.set('event_id', eventId);
  }
  if (reviewStatus === 'pending_review' || reviewStatus === 'approved') {
    params.set('review_status', reviewStatus);
  }

  const query = params.toString();
  return `/gifts${query ? `?${query}` : ''}`;
}

export default function GiftEditPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const searchParams = useSearchParams();
  const giftId = useMemo(() => Number(params.id), [params.id]);
  const returnHref = useMemo(
    () => getGiftListHref(new URLSearchParams(searchParams.toString())),
    [searchParams]
  );

  const [gift, setGift] = useState<Gift | null>(null);
  const [criteria, setCriteria] = useState<Criteria[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadGiftEditData = useCallback(async () => {
    if (!Number.isInteger(giftId) || giftId <= 0) {
      setError('Некорректный ID приза');
      setIsLoading(false);
      return;
    }

    try {
      setIsLoading(true);
      setError(null);
      const [giftResponse, criteriaResponse] = await Promise.all([
        giftsApi.getById(giftId),
        criteriaApi.getAll(),
      ]);
      setGift(giftResponse);
      setCriteria(criteriaResponse.criteria);
    } catch (err) {
      setGift(null);
      setError('Приз не найден или недоступен');
      console.error('Failed to load gift edit page:', {
        gift_id: giftId,
        operation: 'load_gift_edit_page',
        error: err,
      });
    } finally {
      setIsLoading(false);
    }
  }, [giftId]);

  useEffect(() => {
    loadGiftEditData();
  }, [loadGiftEditData]);

  const handleSubmit = async (data: UpdateGiftRequest) => {
    await giftsApi.update(giftId, data);
    router.push(returnHref);
  };

  const donorName =
    gift && [gift.first_name, gift.last_name].filter(Boolean).join(' ');
  const donorTitle =
    donorName || gift?.username || (gift ? `user${gift.user_id}` : 'Приз');
  const reviewStatus = gift?.review_status || 'pending_review';

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">Загрузка...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <Link
            href={returnHref}
            className="mb-3 inline-flex text-sm font-medium text-brand-500 hover:text-brand-600 dark:text-brand-400 dark:hover:text-brand-300"
          >
            Назад к призам
          </Link>
          <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
            Редактировать приз
          </h1>
          <p className="text-gray-600 dark:text-gray-400">{donorTitle}</p>
        </div>

        {gift && (
          <div className="flex flex-wrap items-center gap-2">
            <Badge
              color={reviewStatus === 'pending_review' ? 'warning' : 'success'}
              size="sm"
            >
              {reviewStatus === 'pending_review'
                ? 'Новый / на проверке'
                : 'Проверен'}
            </Badge>
            <span className="text-sm text-gray-500 dark:text-gray-400">
              ID: {gift.id}
            </span>
          </div>
        )}
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
          <Link
            href={returnHref}
            className="mt-3 inline-flex text-sm font-medium text-brand-500 hover:text-brand-600 dark:text-brand-400 dark:hover:text-brand-300"
          >
            Вернуться к списку призов
          </Link>
        </div>
      )}

      {gift && (
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
          <div className="rounded-xl border border-gray-200 bg-white p-5 dark:border-white/[0.05] dark:bg-white/[0.03] lg:p-6">
            <GiftEditForm
              gift={gift}
              criteria={criteria}
              onSubmit={handleSubmit}
              onCancel={() => router.push(returnHref)}
            />
          </div>

          <aside className="space-y-4">
            <GiftPhotoPreviewGrid gift={gift} />

            <div className="rounded-xl border border-gray-200 bg-white p-5 dark:border-white/[0.05] dark:bg-white/[0.03]">
              <h2 className="mb-3 text-base font-semibold text-gray-800 dark:text-white/90">
                Даритель
              </h2>
              <div className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
                <p className="font-medium text-gray-800 dark:text-white/90">
                  {donorTitle}
                </p>
                <p>@{gift.username || `user${gift.user_id}`}</p>
                <p>ID пользователя: {gift.user_id}</p>
              </div>
            </div>
          </aside>
        </div>
      )}
    </div>
  );
}
