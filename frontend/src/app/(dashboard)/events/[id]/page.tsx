'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { eventsApi } from '@/api/events';
import EventForm from '@/components/events/EventForm';
import type { CreateEventRequest, Event, UpdateEventRequest } from '@/types';

export default function EventEditPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const eventId = useMemo(() => Number(params.id), [params.id]);
  const [event, setEvent] = useState<Event | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadEvent = useCallback(async () => {
    if (!Number.isFinite(eventId) || eventId <= 0) {
      setError('Некорректный ID события');
      setIsLoading(false);
      return;
    }

    try {
      setIsLoading(true);
      setError(null);
      const response = await eventsApi.getById(eventId);
      setEvent(response);
    } catch (err) {
      setError('Ошибка загрузки события');
      console.error('Failed to load event:', err);
    } finally {
      setIsLoading(false);
    }
  }, [eventId]);

  useEffect(() => {
    loadEvent();
  }, [loadEvent]);

  const handleSubmit = async (
    data: CreateEventRequest | UpdateEventRequest,
    gpxFile?: File
  ) => {
    try {
      setIsSaving(true);
      setError(null);
      await eventsApi.update(eventId, data);
      if (gpxFile) {
        await eventsApi.uploadGpxFile(eventId, gpxFile);
      }
      router.push('/events');
    } catch (err) {
      setError('Ошибка сохранения события');
      console.error('Failed to update event:', err);
      throw err;
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancel = () => {
    router.push('/events');
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">Загрузка...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
          Редактировать событие
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          {event ? event.name : `Событие #${eventId}`}
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {event && (
        <div className="max-w-4xl rounded-xl border border-gray-200 bg-white p-5 dark:border-white/[0.05] dark:bg-white/[0.03] lg:p-6">
          <EventForm
            event={event}
            onSubmit={handleSubmit}
            onCancel={handleCancel}
            isLoading={isSaving}
          />
        </div>
      )}
    </div>
  );
}
