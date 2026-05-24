'use client';

import { useEffect, useState } from 'react';
import { nominationsApi } from '@/api/nominations';
import { eventsApi } from '@/api/events';
import { participantsApi } from '@/api/participants';
import type {
  Nomination,
  Event,
  CreateNominationRequest,
  UpdateNominationRequest,
  GenderFilter,
  BikeTypeFilter,
} from '@/types';
import NominationsTable from '@/components/nominations/NominationsTable';
import NominationForm from '@/components/nominations/NominationForm';
import Select from '@/components/form/Select';
import Label from '@/components/form/Label';
import Button from '@/components/ui/button/Button';
import { Modal } from '@/components/ui/modal';
import { useModal } from '@/hooks/useModal';
import { PlusIcon } from '@/icons';

export default function NominationsPage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [selectedEventId, setSelectedEventId] = useState<number | null>(null);
  const [nominations, setNominations] = useState<Nomination[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [editingNomination, setEditingNomination] = useState<Nomination | null>(null);
  const [participantsCount, setParticipantsCount] = useState<number | undefined>(undefined);

  const { isOpen: isCreateOpen, openModal: openCreateModal, closeModal: closeCreateModal } =
    useModal();
  const { isOpen: isEditOpen, openModal: openEditModal, closeModal: closeEditModal } = useModal();

  // Загружаем количество участников при открытии формы создания
  useEffect(() => {
    if (isCreateOpen && selectedEventId) {
      loadParticipantsCount(selectedEventId, 'all', 'all');
    }
  }, [isCreateOpen, selectedEventId]);

  // Загрузка событий
  useEffect(() => {
    loadEvents();
  }, []);

  // Загрузка номинаций при изменении события
  useEffect(() => {
    if (selectedEventId) {
      loadNominations();
    } else {
      setNominations([]);
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

  const loadNominations = async () => {
    if (!selectedEventId) return;

    try {
      setIsLoading(true);
      setError(null);
      const response = await nominationsApi.getByEvent(selectedEventId);
      setNominations(response.nominations);
    } catch (err) {
      setError('Ошибка загрузки номинаций');
      console.error('Failed to load nominations:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const loadParticipantsCount = async (
    eventId: number,
    genderFilter: GenderFilter,
    bikeTypeFilter: BikeTypeFilter
  ) => {
    try {
      const filters: {
        bike_type?: string;
        gender?: string;
      } = {};

      // Применяем фильтры только если они не "all"
      if (genderFilter !== 'all') {
        filters.gender = genderFilter;
      }
      if (bikeTypeFilter !== 'all') {
        filters.bike_type = bikeTypeFilter;
      }

      const response = await participantsApi.getByEvent(eventId, filters);
      setParticipantsCount(response.total);
    } catch (err) {
      console.error('Failed to load participants count:', err);
      setParticipantsCount(undefined);
    }
  };

  const handleCreate = async (data: CreateNominationRequest | UpdateNominationRequest) => {
    try {
      if (editingNomination) {
        await nominationsApi.update(editingNomination.id, data as UpdateNominationRequest);
      } else {
        await nominationsApi.create(data as CreateNominationRequest);
      }
      closeCreateModal();
      closeEditModal();
      setEditingNomination(null);
      setParticipantsCount(undefined);
      await loadNominations();
    } catch (err) {
      setError(editingNomination ? 'Ошибка обновления номинации' : 'Ошибка создания номинации');
      console.error('Failed to save nomination:', err);
      throw err;
    }
  };

  const handleEdit = (nomination: Nomination) => {
    setEditingNomination(nomination);
    // Загружаем количество подходящих участников для редактируемой номинации
    if (selectedEventId) {
      loadParticipantsCount(
        selectedEventId,
        nomination.gender_filter,
        nomination.bike_type_filter
      );
    }
    openEditModal();
  };

  const handleDelete = async (nominationId: number) => {
    try {
      await nominationsApi.delete(nominationId);
      await loadNominations();
    } catch (err) {
      setError('Ошибка удаления номинации');
      console.error('Failed to delete nomination:', err);
    }
  };

  const handleMoveUp = async (nominationId: number) => {
    const nomination = nominations.find((n) => n.id === nominationId);
    if (!nomination || !selectedEventId) return;

    const sorted = [...nominations].sort((a, b) => a.sort_order - b.sort_order);
    const index = sorted.findIndex((n) => n.id === nominationId);
    if (index <= 0) return;

    const prevNomination = sorted[index - 1];
    const newSortOrder = prevNomination.sort_order;
    const prevNewSortOrder = nomination.sort_order;

    try {
      // Обновляем обе номинации
      await nominationsApi.update(nominationId, { sort_order: newSortOrder });
      await nominationsApi.update(prevNomination.id, { sort_order: prevNewSortOrder });
      await loadNominations();
    } catch (err) {
      setError('Ошибка изменения порядка');
      console.error('Failed to move nomination:', err);
    }
  };

  const handleMoveDown = async (nominationId: number) => {
    const nomination = nominations.find((n) => n.id === nominationId);
    if (!nomination || !selectedEventId) return;

    const sorted = [...nominations].sort((a, b) => a.sort_order - b.sort_order);
    const index = sorted.findIndex((n) => n.id === nominationId);
    if (index < 0 || index >= sorted.length - 1) return;

    const nextNomination = sorted[index + 1];
    const newSortOrder = nextNomination.sort_order;
    const nextNewSortOrder = nomination.sort_order;

    try {
      // Обновляем обе номинации
      await nominationsApi.update(nominationId, { sort_order: newSortOrder });
      await nominationsApi.update(nextNomination.id, { sort_order: nextNewSortOrder });
      await loadNominations();
    } catch (err) {
      setError('Ошибка изменения порядка');
      console.error('Failed to move nomination:', err);
    }
  };

  const handleCancelCreate = () => {
    closeCreateModal();
    setParticipantsCount(undefined);
  };

  const handleCancelEdit = () => {
    closeEditModal();
    setEditingNomination(null);
    setParticipantsCount(undefined);
  };

  const handleFormFilterChange = (genderFilter: GenderFilter, bikeTypeFilter: BikeTypeFilter) => {
    if (selectedEventId) {
      loadParticipantsCount(selectedEventId, genderFilter, bikeTypeFilter);
    }
  };

  const eventOptions = events.map((e) => ({
    value: String(e.id),
    label: e.name,
  }));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
            Номинации
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Управление номинациями для событий
          </p>
        </div>
        <Button
          startIcon={<PlusIcon />}
          onClick={openCreateModal}
          size="sm"
          disabled={!selectedEventId}
        >
          Создать номинацию
        </Button>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {/* Фильтр по событию */}
      <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <Label>Событие</Label>
            <Select
              options={eventOptions}
              placeholder="Выберите событие"
              defaultValue={selectedEventId ? String(selectedEventId) : ''}
              onChange={(value) => setSelectedEventId(value ? Number(value) : null)}
            />
          </div>
        </div>
      </div>

      {/* Информация о количестве */}
      {selectedEventId && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Найдено номинаций: {nominations.length}
          </p>
        </div>
      )}

      {/* Таблица номинаций */}
      {selectedEventId ? (
        <NominationsTable
          nominations={nominations}
          isLoading={isLoading}
          onEdit={handleEdit}
          onDelete={handleDelete}
          onMoveUp={handleMoveUp}
          onMoveDown={handleMoveDown}
        />
      ) : (
        <div className="flex items-center justify-center py-12">
          <div className="text-gray-500 dark:text-gray-400">
            Выберите событие для просмотра номинаций
          </div>
        </div>
      )}

      {/* Модальное окно создания */}
      <Modal isOpen={isCreateOpen} onClose={closeCreateModal} className="max-w-2xl m-4">
        <div className="no-scrollbar relative w-full max-w-2xl overflow-y-auto rounded-3xl bg-white p-4 dark:bg-gray-900 lg:p-11">
          <div className="px-2 pr-14">
            <h4 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white/90">
              Создать номинацию
            </h4>
            <p className="mb-6 text-sm text-gray-500 dark:text-gray-400 lg:mb-7">
              Заполните форму для создания новой номинации
            </p>
          </div>
          <div className="px-2">
            {selectedEventId && (
              <NominationForm
                eventId={selectedEventId}
                onSubmit={handleCreate}
                onCancel={handleCancelCreate}
                participantsCount={participantsCount}
                onFilterChange={handleFormFilterChange}
              />
            )}
          </div>
        </div>
      </Modal>

      {/* Модальное окно редактирования */}
      <Modal isOpen={isEditOpen} onClose={closeEditModal} className="max-w-2xl m-4">
        <div className="no-scrollbar relative w-full max-w-2xl overflow-y-auto rounded-3xl bg-white p-4 dark:bg-gray-900 lg:p-11">
          <div className="px-2 pr-14">
            <h4 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white/90">
              Редактировать номинацию
            </h4>
            <p className="mb-6 text-sm text-gray-500 dark:text-gray-400 lg:mb-7">
              Измените данные номинации
            </p>
          </div>
          <div className="px-2">
            {editingNomination && selectedEventId && (
              <NominationForm
                eventId={selectedEventId}
                nomination={editingNomination}
                onSubmit={handleCreate}
                onCancel={handleCancelEdit}
                participantsCount={participantsCount}
                onFilterChange={handleFormFilterChange}
              />
            )}
          </div>
        </div>
      </Modal>
    </div>
  );
}
