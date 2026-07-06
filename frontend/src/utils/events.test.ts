import { describe, expect, it } from 'vitest';
import type { Event, EventListResponse } from '@/types';
import { extractActiveEvent } from './events';

function makeEvent(overrides: Partial<Event>): Event {
  return {
    id: 1,
    name: 'Event',
    description: '',
    participation_conditions: '',
    active: false,
    stop_results: false,
    stop_gifts: false,
    telegram_texts: {} as Event['telegram_texts'],
    created_at: '2026-06-15T00:00:00Z',
    updated_at: '2026-06-15T00:00:00Z',
    ...overrides,
  };
}

function makeResponse(events: Event[]): EventListResponse {
  return { events, total: events.length };
}

describe('extractActiveEvent', () => {
  it('returns the active event from an active-only response', () => {
    const active = makeEvent({ id: 7, name: 'Active', active: true });
    const event = extractActiveEvent(makeResponse([active]));

    expect(event?.id).toBe(7);
  });

  it('returns null when there is no active event', () => {
    expect(extractActiveEvent(makeResponse([]))).toBeNull();
  });

  it('does not fall back to an inactive event', () => {
    const inactive = makeEvent({ id: 3, name: 'Old', active: false });

    expect(extractActiveEvent(makeResponse([inactive]))).toBeNull();
  });
});
