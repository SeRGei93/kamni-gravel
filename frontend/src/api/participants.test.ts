import { afterEach, describe, expect, it, vi } from 'vitest';
import { participantsApi } from './participants';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('participantsApi list filters', () => {
  it('sends the result criteria filter before pagination', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ participants: [], total: 0 }), { status: 200 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await participantsApi.listByEvent(77, {
      criteria_id: 12,
      page: 2,
      page_size: 50,
    });

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/events/77/participants?criteria_id=12&page=2&page_size=50',
    );
  });

  it('sends all current list filters and requests the full export set', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ participants: [], total: 0 }), { status: 200 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await participantsApi.listByEvent(77, {
      bike_type: 'gravel',
      gender: 'female',
      is_finished: true,
      has_gift: false,
      criteria_id: 12,
      q: 'Анна',
      sort: 'place',
      order: 'desc',
      page: 1,
      page_size: 'all',
    });

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/events/77/participants?bike_type=gravel&gender=female&is_finished=true&has_gift=false&criteria_id=12&q=%D0%90%D0%BD%D0%BD%D0%B0&sort=place&order=desc&page=1&page_size=all',
    );
  });
});
