'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { giftsApi } from '@/api/gifts';
import { eventsApi } from '@/api/events';
import { prizeDistributionApi } from '@/api/prizeDistribution';
import type { Gift, Event, GiftReviewStatus } from '@/types';
import GiftsTable from '@/components/gifts/GiftsTable';
import Select from '@/components/form/Select';
import Label from '@/components/form/Label';
import { GIFT_REVIEW_STATUS_FILTER_OPTIONS } from '@/constants';

type GiftReviewStatusFilter = 'all' | GiftReviewStatus;

function parseReviewStatusFilter(value: string | null): GiftReviewStatusFilter {
  return value === 'pending_review' || value === 'approved' ? value : 'all';
}

function parseEventId(value: string | null): number | null {
  if (!value) {
    return null;
  }

  const eventId = Number(value);
  return Number.isInteger(eventId) && eventId > 0 ? eventId : null;
}

export default function GiftsPage() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const eventIdParam = searchParams.get('event_id');
  const reviewStatusParam = searchParams.get('review_status');

  const [events, setEvents] = useState<Event[]>([]);
  const [selectedEventId, setSelectedEventId] = useState<number | null>(null);
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [allGifts, setAllGifts] = useState<Gift[]>([]);
  const [reviewStatusFilter, setReviewStatusFilter] =
    useState<GiftReviewStatusFilter>(
      parseReviewStatusFilter(reviewStatusParam)
    );
  const [assignedGiftIds, setAssignedGiftIds] = useState<Set<number>>(new Set());
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const listQueryString = useMemo(() => {
    const params = new URLSearchParams();

    if (selectedEventId) {
      params.set('event_id', String(selectedEventId));
    }
    if (reviewStatusFilter !== 'all') {
      params.set('review_status', reviewStatusFilter);
    }

    return params.toString();
  }, [selectedEventId, reviewStatusFilter]);

  const updateListQuery = useCallback(
    (next: {
      eventId?: number | null;
      reviewStatus?: GiftReviewStatusFilter;
    }) => {
      const params = new URLSearchParams(searchParams.toString());

      if ('eventId' in next) {
        if (next.eventId) {
          params.set('event_id', String(next.eventId));
        } else {
          params.delete('event_id');
        }
      }

      if ('reviewStatus' in next) {
        if (next.reviewStatus && next.reviewStatus !== 'all') {
          params.set('review_status', next.reviewStatus);
        } else {
          params.delete('review_status');
        }
      }

      const query = params.toString();
      router.replace(`${pathname}${query ? `?${query}` : ''}`, {
        scroll: false,
      });
    },
    [pathname, router, searchParams]
  );

  // Загрузка событий
  useEffect(() => {
    loadEvents();
  }, []);

  const loadEvents = async () => {
    try {
      const response = await eventsApi.getAll();
      setEvents(response.events);
    } catch (err) {
      setError('Ошибка загрузки событий');
      console.error('Failed to load events:', err);
    }
  };

  useEffect(() => {
    const nextReviewStatus = parseReviewStatusFilter(reviewStatusParam);
    setReviewStatusFilter((current) =>
      current === nextReviewStatus ? current : nextReviewStatus
    );
  }, [reviewStatusParam]);

  useEffect(() => {
    if (events.length === 0) {
      setSelectedEventId(null);
      return;
    }

    const requestedEventId = parseEventId(eventIdParam);
    const eventFromUrl = requestedEventId
      ? events.find((event) => event.id === requestedEventId)
      : undefined;
    const activeEvent = events.find((event) => event.active);
    const nextEventId = (eventFromUrl || activeEvent || events[0]).id;

    setSelectedEventId((current) =>
      current === nextEventId ? current : nextEventId
    );
  }, [events, eventIdParam]);

  const loadGifts = useCallback(async () => {
    if (!selectedEventId) return;

    try {
      setIsLoading(true);
      setError(null);

      // Загружаем полный список для счётчиков и выбранный фильтр для таблицы.
      const allResponse = await giftsApi.getByEvent(selectedEventId);
      setAllGifts(allResponse.gifts);

      if (reviewStatusFilter === 'all') {
        setGifts(allResponse.gifts);
      } else {
        const response = await giftsApi.getByEvent(
          selectedEventId,
          reviewStatusFilter
        );
        setGifts(response.gifts);
      }

      // Загружаем распределение призов
      try {
        const distribution = await prizeDistributionApi.getPrizeDistribution(selectedEventId);
        // Собираем ID всех назначенных подарков
        const assignedIds = new Set<number>();
        distribution.distribution.forEach((dist) => {
          if (dist.matched_gifts && dist.matched_gifts.length > 0) {
            dist.matched_gifts.forEach((gift) => assignedIds.add(gift.id));
          }
        });
        setAssignedGiftIds(assignedIds);
      } catch (err) {
        console.error('Failed to load prize distribution:', {
          event_id: selectedEventId,
          operation: 'load_prize_distribution',
          error: err,
        });
        setAssignedGiftIds(new Set());
      }
    } catch (err) {
      setError('Ошибка загрузки подарков');
      console.error('Failed to load gifts:', {
        event_id: selectedEventId,
        review_status: reviewStatusFilter,
        operation: 'load_gifts',
        error: err,
      });
    } finally {
      setIsLoading(false);
    }
  }, [selectedEventId, reviewStatusFilter]);

  // Загрузка подарков при изменении фильтров
  useEffect(() => {
    if (selectedEventId) {
      loadGifts();
    } else {
      setGifts([]);
      setAllGifts([]);
    }
  }, [selectedEventId, loadGifts]);

  const handleDelete = async (giftId: number) => {
    try {
      await giftsApi.delete(giftId);
      // Перезагружаем список подарков
      await loadGifts();
    } catch (err) {
      setError('Ошибка удаления подарка');
      console.error('Failed to delete gift:', {
        gift_id: giftId,
        event_id: selectedEventId,
        operation: 'delete_gift',
        error: err,
      });
      throw err;
    }
  };

  const handleApprove = async (gift: Gift) => {
    try {
      await giftsApi.update(gift.id, {
        description: gift.description,
        gender_filter: gift.gender_filter || 'all',
        bike_type_filter: gift.bike_type_filter || 'all',
        review_status: 'approved',
        place: gift.place ?? null,
        criteria_ids: gift.criteria?.map((criteria) => criteria.id) || [],
      });
      await loadGifts();
    } catch (err) {
      setError('Ошибка проверки подарка');
      console.error('Failed to approve gift:', {
        gift_id: gift.id,
        event_id: selectedEventId,
        operation: 'approve_gift',
        error: err,
      });
      throw err;
    }
  };

  const eventOptions = events.map((e) => ({
    value: String(e.id),
    label: e.name,
  }));
  const pendingReviewCount = gifts.filter(
    (gift) => gift.review_status === 'pending_review'
  ).length;
  const totalPendingReviewCount = allGifts.filter(
    (gift) => gift.review_status === 'pending_review'
  ).length;

  return (
    <div className="space-y-6">
      <div>
        <div>
          <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
            Подарки
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Подарки поступают из Telegram и проходят проверку администратора перед распределением
          </p>
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {selectedEventId && totalPendingReviewCount > 0 && (
        <div className="rounded-lg border border-warning-200 bg-warning-50 p-4 dark:border-warning-800 dark:bg-warning-900/20">
          <p className="text-sm font-medium text-warning-700 dark:text-orange-300">
            На проверке {totalPendingReviewCount} подарков. До проверки они не участвуют в распределении.
          </p>
        </div>
      )}

      {/* Фильтры */}
      <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <Label>Событие</Label>
            <Select
              options={eventOptions}
              placeholder="Выберите событие"
              key={`event-${selectedEventId ?? 'empty'}`}
              defaultValue={selectedEventId ? String(selectedEventId) : ''}
              onChange={(value) => {
                const nextEventId = value ? Number(value) : null;
                setSelectedEventId(nextEventId);
                updateListQuery({ eventId: nextEventId });
              }}
            />
          </div>
          <div>
            <Label>Статус проверки</Label>
            <Select
              options={GIFT_REVIEW_STATUS_FILTER_OPTIONS}
              key={`review-status-${reviewStatusFilter}`}
              defaultValue={reviewStatusFilter}
              onChange={(value) => {
                const nextReviewStatus = parseReviewStatusFilter(value);
                setReviewStatusFilter(nextReviewStatus);
                updateListQuery({ reviewStatus: nextReviewStatus });
              }}
            />
          </div>
        </div>
      </div>

      {/* Информация о количестве */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Показано подарков: {gifts.length}
        </p>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Всего: {allGifts.length} · На проверке: {totalPendingReviewCount}
          {reviewStatusFilter !== 'all' && ` · В фильтре: ${pendingReviewCount}`}
        </p>
      </div>

      {/* Таблица */}
      <GiftsTable
        gifts={gifts}
        assignedGiftIds={assignedGiftIds}
        isLoading={isLoading}
        onApprove={handleApprove}
        onDelete={handleDelete}
        editQueryString={listQueryString}
      />
    </div>
  );
}
