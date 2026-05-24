'use client';

import { useEffect, useState } from 'react';
import { giftsApi } from '@/api/gifts';
import { eventsApi } from '@/api/events';
import { prizeDistributionApi } from '@/api/prizeDistribution';
import type { Gift, Event } from '@/types';
import GiftsTable from '@/components/gifts/GiftsTable';
import CreateGiftModal from '@/components/gifts/CreateGiftModal';
import EditGiftModal from '@/components/gifts/EditGiftModal';
import Select from '@/components/form/Select';
import Label from '@/components/form/Label';
import Button from '@/components/ui/button/Button';

export default function GiftsPage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [selectedEventId, setSelectedEventId] = useState<number | null>(null);
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [assignedGiftIds, setAssignedGiftIds] = useState<Set<number>>(new Set());
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingGift, setEditingGift] = useState<Gift | null>(null);

  // Загрузка событий
  useEffect(() => {
    loadEvents();
  }, []);

  // Загрузка подарков при изменении фильтров
  useEffect(() => {
    if (selectedEventId) {
      loadGifts();
    } else {
      setGifts([]);
    }
  }, [selectedEventId]);

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

  const loadGifts = async () => {
    if (!selectedEventId) return;

    try {
      setIsLoading(true);
      setError(null);

      // Загружаем подарки
      const response = await giftsApi.getByEvent(selectedEventId);
      setGifts(response.gifts);

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
  };

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

  const eventOptions = events.map((e) => ({
    value: String(e.id),
    label: e.name,
  }));

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
            Подарки
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Список подарков в призовом фонде
          </p>
        </div>
        {selectedEventId && (
          <Button onClick={() => setIsCreateModalOpen(true)}>
            Добавить подарок
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
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
        </div>
      </div>

      {/* Информация о количестве */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Найдено подарков: {gifts.length}
        </p>
      </div>

      {/* Таблица */}
      <GiftsTable
        gifts={gifts}
        assignedGiftIds={assignedGiftIds}
        isLoading={isLoading}
        onEdit={(gift) => setEditingGift(gift)}
        onDelete={handleDelete}
      />

      {/* Модальное окно создания подарка */}
      {selectedEventId && (
        <CreateGiftModal
          isOpen={isCreateModalOpen}
          onClose={() => setIsCreateModalOpen(false)}
          eventId={selectedEventId}
          onSuccess={loadGifts}
        />
      )}

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
