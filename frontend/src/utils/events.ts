import type { Event, EventListResponse } from '@/types';

/**
 * Извлекает единственное активное событие из ответа API.
 * Возвращает null, если активного события нет — без отката на первое
 * неактивное событие.
 */
export function extractActiveEvent(
  response: EventListResponse
): Event | null {
  return response.events.find((event) => event.active) ?? null;
}
