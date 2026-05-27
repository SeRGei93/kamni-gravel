'use client';

import { useCallback, useEffect, useState } from 'react';
import { participantsApi } from '@/api/participants';
import { eventsApi } from '@/api/events';
import type { Participant, Event } from '@/types';
import ParticipantsTable from '@/components/participants/ParticipantsTable';
import Select from '@/components/form/Select';
import Input from '@/components/form/input/InputField';
import Label from '@/components/form/Label';

export default function ParticipantsPage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [selectedEventId, setSelectedEventId] = useState<number | null>(null);
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [deletingParticipantId, setDeletingParticipantId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Фильтры
  const [genderFilter, setGenderFilter] = useState<string>('');
  const [bikeTypeFilter, setBikeTypeFilter] = useState<string>('');
  const [isFinishedFilter, setIsFinishedFilter] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState('');

  const loadEvents = useCallback(async () => {
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
  }, []);

  const loadParticipants = useCallback(async () => {
    if (!selectedEventId) {
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

      const response = await participantsApi.getByEvent(selectedEventId, filters);
      setParticipants(response.participants);
    } catch (err) {
      setError('Ошибка загрузки участников');
      console.error('Failed to load participants:', err);
    } finally {
      setIsLoading(false);
    }
  }, [bikeTypeFilter, genderFilter, isFinishedFilter, selectedEventId]);

  // Загрузка событий
  useEffect(() => {
    loadEvents();
  }, [loadEvents]);

  // Загрузка участников при изменении фильтров
  useEffect(() => {
    loadParticipants();
  }, [loadParticipants]);

  const handleDeleteParticipant = async (participant: Participant) => {
    if (!selectedEventId) return;
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
        event_id: selectedEventId,
        error: err,
      });
    } finally {
      setDeletingParticipantId(null);
    }
  };

  // Фильтрация по поисковому запросу
  const filteredParticipants = participants.filter((p) => {
    if (!searchQuery) return true;
    const query = searchQuery.toLowerCase();
    return (
      p.username?.toLowerCase().includes(query) ||
      p.first_name?.toLowerCase().includes(query) ||
      p.last_name?.toLowerCase().includes(query) ||
      String(p.user_id).includes(query)
    );
  });

  const eventOptions = events.map((e) => ({
    value: String(e.id),
    label: e.name,
  }));

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
            <Label>Событие</Label>
            <Select
              options={eventOptions}
              placeholder="Выберите событие"
              defaultValue={selectedEventId ? String(selectedEventId) : ''}
              onChange={(value) => setSelectedEventId(value ? Number(value) : null)}
            />
          </div>

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
