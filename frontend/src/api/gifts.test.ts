import { afterEach, describe, expect, it, vi } from 'vitest';
import { giftsApi } from './gifts';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('giftsApi', () => {
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

  it('posts an empty request to assign a random recipient', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(null, { status: 204 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await giftsApi.assignRandomRecipient(77);

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/gifts/77/random-recipient',
    );
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'POST' });
    expect(fetchMock.mock.calls[0][1].body).toBeUndefined();
  });
});
