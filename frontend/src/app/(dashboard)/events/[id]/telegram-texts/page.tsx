'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { eventsApi } from '@/api/events';
import EventTelegramTextsForm from '@/components/events/EventTelegramTextsForm';
import type { Event, EventTelegramTexts } from '@/types';

export default function EventTelegramTextsPage() {
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

  const handleSubmit = async (telegramTexts: EventTelegramTexts) => {
    try {
      setIsSaving(true);
      setError(null);
      const updated = await eventsApi.update(eventId, {
        telegram_texts: telegramTexts,
      });
      setEvent(updated);
      router.push('/events');
    } catch (err) {
      setError('Ошибка сохранения текстов');
      console.error('Failed to update event telegram texts:', err);
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
          Тексты Telegram
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
        <EventTelegramTextsForm
          texts={event.telegram_texts}
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          isLoading={isSaving}
        />
      )}
    </div>
  );
}
