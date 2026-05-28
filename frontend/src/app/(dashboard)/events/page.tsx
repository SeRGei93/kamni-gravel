'use client';

import { useEffect, useState } from 'react';
import { eventsApi } from '@/api/events';
import type { Event, CreateEventRequest, UpdateEventRequest } from '@/types';
import EventsTable from '@/components/events/EventsTable';
import EventForm from '@/components/events/EventForm';
import Button from '@/components/ui/button/Button';
import { Modal } from '@/components/ui/modal';
import { useModal } from '@/hooks/useModal';
import { PlusIcon } from '@/icons';

export default function EventsPage() {
  const [events, setEvents] = useState<Event[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const { isOpen: isCreateOpen, openModal: openCreateModal, closeModal: closeCreateModal } = useModal();

  useEffect(() => {
    loadEvents();
  }, []);

  const loadEvents = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await eventsApi.getAll();
      setEvents(response.events);
    } catch (err) {
      setError('Ошибка загрузки событий');
      console.error('Failed to load events:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreate = async (
    data: CreateEventRequest | UpdateEventRequest,
    gpxFile?: File
  ) => {
    try {
      // При создании все поля обязательны
      const createData: CreateEventRequest = {
        name: data.name || '',
        description: data.description || '',
        participation_conditions: data.participation_conditions || '',
        active: data.active ?? true,
        start_date: data.start_date,
        end_date: data.end_date,
      };
      const event = await eventsApi.create(createData);
      if (gpxFile) {
        await eventsApi.uploadGpxFile(event.id, gpxFile);
      }
      closeCreateModal();
      await loadEvents();
    } catch (err) {
      setError('Ошибка создания события');
      console.error('Failed to create event:', err);
      throw err;
    }
  };

  const handleDelete = async (eventId: number) => {
    try {
      await eventsApi.delete(eventId);
      await loadEvents();
    } catch (err) {
      setError('Ошибка удаления события');
      console.error('Failed to delete event:', err);
      throw err;
    }
  };

  const handleCancelCreate = () => {
    closeCreateModal();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
            События
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Управление событиями велогонки
          </p>
        </div>
        <Button startIcon={<PlusIcon />} onClick={openCreateModal} size="sm">
          Создать событие
        </Button>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {/* Таблица событий */}
      <EventsTable
        events={events}
        isLoading={isLoading}
        onDelete={handleDelete}
      />

      {/* Модальное окно создания */}
      <Modal isOpen={isCreateOpen} onClose={closeCreateModal} className="max-w-4xl m-4">
        <div className="no-scrollbar relative w-full max-w-4xl overflow-y-auto rounded-3xl bg-white p-4 dark:bg-gray-900 lg:p-11">
          <div className="px-2 pr-14">
            <h4 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white/90">
              Создать событие
            </h4>
            <p className="mb-6 text-sm text-gray-500 dark:text-gray-400 lg:mb-7">
              Заполните форму для создания нового события
            </p>
          </div>
          <div className="px-2">
            <EventForm
              onSubmit={handleCreate}
              onCancel={handleCancelCreate}
            />
          </div>
        </div>
      </Modal>
    </div>
  );
}
