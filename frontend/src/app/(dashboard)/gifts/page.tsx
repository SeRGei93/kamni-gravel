'use client';

import { useCallback, useEffect, useState } from 'react';
import { giftsApi } from '@/api/gifts';
import { eventsApi } from '@/api/events';
import { prizeDistributionApi } from '@/api/prizeDistribution';
import type { Gift, Event, GiftReviewStatus } from '@/types';
import GiftsTable from '@/components/gifts/GiftsTable';
import EditGiftModal from '@/components/gifts/EditGiftModal';
import Select from '@/components/form/Select';
import Label from '@/components/form/Label';
import { GIFT_REVIEW_STATUS_FILTER_OPTIONS } from '@/constants';

type GiftReviewStatusFilter = 'all' | GiftReviewStatus;

export default function GiftsPage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [selectedEventId, setSelectedEventId] = useState<number | null>(null);
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [allGifts, setAllGifts] = useState<Gift[]>([]);
  const [reviewStatusFilter, setReviewStatusFilter] =
    useState<GiftReviewStatusFilter>('all');
  const [assignedGiftIds, setAssignedGiftIds] = useState<Set<number>>(new Set());
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [editingGift, setEditingGift] = useState<Gift | null>(null);

  // Загрузка событий
  useEffect(() => {
    loadEvents();
  }, []);

  const loadEvents = async () => {
    try {
      const response = await eventsApi.getAll();
      setEvents(response.events);
      // Автоматически выбираем первое активное событие
      const activeEvent = response.events.find((e) => e.active);
      if (activeEvent) {
        setSelectedEventId(activeEvent.id);
      } else if (response.events.length > 0) {
        setSelectedEventId(response.events[0].id);
      }
    } catch (err) {
      setError('Ошибка загрузки событий');
      console.error('Failed to load events:', err);
    }
  };

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
        console.error('Failed to load prize distribution:', err);
        setAssignedGiftIds(new Set());
      }
    } catch (err) {
      setError('Ошибка загрузки подарков');
      console.error('Failed to load gifts:', err);
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
      console.error('Failed to delete gift:', err);
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
      console.error('Failed to approve gift:', err);
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
              defaultValue={selectedEventId ? String(selectedEventId) : ''}
              onChange={(value) =>
                setSelectedEventId(value ? Number(value) : null)
              }
            />
          </div>
          <div>
            <Label>Статус проверки</Label>
            <Select
              options={GIFT_REVIEW_STATUS_FILTER_OPTIONS}
              defaultValue={reviewStatusFilter}
              onChange={(value) =>
                setReviewStatusFilter(value as GiftReviewStatusFilter)
              }
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
        onEdit={(gift) => setEditingGift(gift)}
        onApprove={handleApprove}
        onDelete={handleDelete}
      />

      {/* Модальное окно редактирования подарка */}
      {editingGift && (
        <EditGiftModal
          isOpen={!!editingGift}
          onClose={() => setEditingGift(null)}
          gift={editingGift}
          onSuccess={loadGifts}
        />
      )}
    </div>
  );
}
