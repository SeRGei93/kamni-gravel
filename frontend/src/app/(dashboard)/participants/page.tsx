'use client';

import { useCallback, useEffect, useState } from 'react';
import { participantsApi } from '@/api/participants';
import { eventsApi } from '@/api/events';
import { extractActiveEvent } from '@/utils/events';
import { filterParticipants, type HasGiftFilter } from '@/utils/participants';
import type { Participant } from '@/types';
import ParticipantsTable from '@/components/participants/ParticipantsTable';
import Select from '@/components/form/Select';
import Input from '@/components/form/input/InputField';
import Label from '@/components/form/Label';

export default function ParticipantsPage() {
  const [activeEventId, setActiveEventId] = useState<number | null>(null);
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [deletingParticipantId, setDeletingParticipantId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Фильтры
  const [genderFilter, setGenderFilter] = useState<string>('');
  const [bikeTypeFilter, setBikeTypeFilter] = useState<string>('');
  const [isFinishedFilter, setIsFinishedFilter] = useState<string>('');
  const [hasGiftFilter, setHasGiftFilter] = useState<HasGiftFilter>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const loadActiveEvent = useCallback(async () => {
    try {
      const response = await eventsApi.getActive();
      const activeEvent = extractActiveEvent(response);
      setActiveEventId(activeEvent?.id ?? null);
      if (!activeEvent) {
        setParticipants([]);
        setError('Нет активного события');
      }
    } catch (err) {
      setActiveEventId(null);
      setParticipants([]);
      setError('Ошибка загрузки активного события');
      console.error('Failed to load active event:', {
        operation: 'load_active_event',
        error: err,
      });
    }
  }, []);

  const loadParticipants = useCallback(async () => {
    if (!activeEventId) {
      setParticipants([]);
      return;
    }

    try {
      setIsLoading(true);
      setError(null);

      const filters: {
        bike_type?: string;
        gender?: string;
        is_finished?: boolean;
      } = {};

      if (genderFilter) filters.gender = genderFilter;
      if (bikeTypeFilter) filters.bike_type = bikeTypeFilter;
      if (isFinishedFilter !== '')
        filters.is_finished = isFinishedFilter === 'true';

      const response = await participantsApi.getByEvent(activeEventId, filters);
      setParticipants(response.participants);
    } catch (err) {
      setError('Ошибка загрузки участников');
      console.error('Failed to load participants:', {
        operation: 'load_participants',
        event_id: activeEventId,
        error: err,
      });
    } finally {
      setIsLoading(false);
    }
  }, [bikeTypeFilter, genderFilter, isFinishedFilter, activeEventId]);

  // Загрузка активного события
  useEffect(() => {
    loadActiveEvent();
  }, [loadActiveEvent]);

  // Загрузка участников при изменении фильтров
  useEffect(() => {
    loadParticipants();
  }, [loadParticipants]);

  const handleDeleteParticipant = async (participant: Participant) => {
    if (!activeEventId) return;
    if (
      !window.confirm(
        `Удалить участника ${participant.first_name || participant.username || participant.user_id}?`
      )
    ) {
      return;
    }

    try {
      setDeletingParticipantId(participant.id);
      setError(null);
      await participantsApi.delete(participant.id);
      await loadParticipants();
    } catch (err) {
      setError('Ошибка удаления участника');
      console.error('Failed to delete participant:', {
        operation: 'delete_participant',
        participant_id: participant.id,
        event_id: activeEventId,
        error: err,
      });
    } finally {
      setDeletingParticipantId(null);
    }
  };

  const filteredParticipants = filterParticipants(participants, {
    searchQuery,
    hasGiftFilter,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
          Участники
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Список участников велогонки с фильтрацией
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {/* Фильтры */}
      <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
          <div>
            <Label>Пол</Label>
            <Select
              options={[
                { value: '', label: 'Все' },
                { value: 'male', label: 'Мужской' },
                { value: 'female', label: 'Женский' },
              ]}
              placeholder="Все"
              defaultValue={genderFilter}
              onChange={setGenderFilter}
            />
          </div>

          <div>
            <Label>Тип велосипеда</Label>
            <Select
              options={[
                { value: '', label: 'Все' },
                { value: 'gravel', label: 'Гравийник' },
                { value: 'mtb', label: 'МТБ' },
                { value: 'road', label: 'Шоссе' },
                { value: 'single_speed', label: 'Фикс' },
                { value: 'tandem', label: 'Тандем' },
              ]}
              placeholder="Все"
              defaultValue={bikeTypeFilter}
              onChange={setBikeTypeFilter}
            />
          </div>

          <div>
            <Label>Статус</Label>
            <Select
              options={[
                { value: '', label: 'Все' },
                { value: 'true', label: 'Проехал' },
                { value: 'false', label: 'Не проехал' },
              ]}
              placeholder="Все"
              defaultValue={isFinishedFilter}
              onChange={setIsFinishedFilter}
            />
          </div>

          <div>
            <Label>Добавил приз</Label>
            <Select
              options={[
                { value: 'all', label: 'Все' },
                { value: 'yes', label: 'Да' },
                { value: 'no', label: 'Нет' },
              ]}
              placeholder="Все"
              defaultValue={hasGiftFilter}
              onChange={(value) => setHasGiftFilter(value as HasGiftFilter)}
            />
          </div>

          <div>
            <Label>Поиск</Label>
            <Input
              type="text"
              placeholder="Поиск по имени или username"
              defaultValue={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
        </div>
      </div>

      {/* Информация о количестве */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Найдено участников: {filteredParticipants.length}
        </p>
      </div>

      {/* Таблица */}
      <ParticipantsTable
        participants={filteredParticipants}
        isLoading={isLoading}
        deletingParticipantId={deletingParticipantId}
        onDelete={handleDeleteParticipant}
      />
    </div>
  );
}
