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
});
