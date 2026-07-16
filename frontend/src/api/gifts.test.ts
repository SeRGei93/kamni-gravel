import { afterEach, describe, expect, it, vi } from 'vitest';
import { giftsApi } from './gifts';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('giftsApi', () => {
  it('gets the full filtered list without pagination parameters', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ gifts: [], total: 0 }), { status: 200 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await giftsApi.getByEvent(77, {
      review_status: 'approved',
      owner_user_id: 42,
      q: 'helmet',
    });

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/events/77/gifts?review_status=approved&owner_user_id=42&q=helmet',
    );
  });

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

  it('posts an empty request to assign a random recipient including awarded participants', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(null, { status: 204 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await giftsApi.assignRandomRecipientIncludingAwarded(77);

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/gifts/77/random-recipient-including-awarded',
    );
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'POST' });
    expect(fetchMock.mock.calls[0][1].body).toBeUndefined();
  });

  it('gets protected manual gifts for an event', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ gifts: [] }), { status: 200 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await giftsApi.getManualByEvent(77);

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/events/77/manual-gifts',
    );
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'GET' });
  });

  it('posts the requested number of copies to the protected copy endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ created_count: 3 }), { status: 201 }),
    );
    vi.stubGlobal('fetch', fetchMock);

    const result = await giftsApi.copy(77, 3);

    expect(result).toEqual({ created_count: 3 });
    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:8080/api/gifts/77/copies',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ copies_count: 3 }),
      }),
    );
  });
});
