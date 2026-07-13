import { afterEach, describe, expect, it, vi } from 'vitest';
import { miniappApi } from './miniapp';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('miniappApi manual gift endpoints', () => {
  it('uses the owner endpoint and sends an explicit null to clear a recipient', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(null, { status: 204 })
    );
    vi.stubGlobal('fetch', fetchMock);

    await miniappApi.updateMyGiftRecipient(15, null);

    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:8080/api/miniapp/my-gifts/15/recipient',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ participant_id: null }),
      })
    );
    const request = fetchMock.mock.calls[0][1] as RequestInit;
    expect(new Headers(request.headers).get('Content-Type')).toBe('application/json');
    expect(new Headers(request.headers).get('X-Telegram-Init-Data')).toBeNull();
  });

  it('uses protected list endpoints', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ gifts: [] }), { status: 200 })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ participants: [], total: 0 }), { status: 200 })
      );
    vi.stubGlobal('fetch', fetchMock);

    await miniappApi.getMyGifts();
    await miniappApi.getParticipants();

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      'http://localhost:8080/api/miniapp/my-gifts',
      'http://localhost:8080/api/miniapp/participants',
    ]);
  });

	it('assigns a random recipient through the owner endpoint', async () => {
	  const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
	  vi.stubGlobal('fetch', fetchMock);

	  await miniappApi.assignRandomMyGiftRecipient(15);

	  expect(fetchMock).toHaveBeenCalledWith(
		'http://localhost:8080/api/miniapp/my-gifts/15/random-recipient',
		expect.objectContaining({ method: 'POST' })
	  );
	});
});
