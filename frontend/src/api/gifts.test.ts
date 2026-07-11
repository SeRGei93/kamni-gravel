import { afterEach, describe, expect, it, vi } from 'vitest';
import { giftsApi } from './gifts';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('giftsApi list filters', () => {
  it('sends author and text filters before pagination', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ gifts: [], total: 0 }), { status: 200 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await giftsApi.listByEvent({
      eventId: 77,
      review_status: 'approved',
      owner_user_id: 42,
      q: 'helmet',
      page: 2,
      page_size: 50,
    });

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/events/77/gifts?review_status=approved&owner_user_id=42&q=helmet&page=2&page_size=50',
    );
  });
});
