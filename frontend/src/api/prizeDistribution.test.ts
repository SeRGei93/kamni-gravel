import { afterEach, describe, expect, it, vi } from 'vitest';

import { prizeDistributionApi } from './prizeDistribution';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('prizeDistributionApi', () => {
  it('sends gender and bike type filters before pagination', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ distribution: [], total: 0 }), { status: 200 })
    );
    vi.stubGlobal('fetch', fetchMock);

    await prizeDistributionApi.getPrizeDistribution(77, {
      gender: 'male',
      bike_type: 'gravel',
      match_reason: 'place',
      page: 2,
      page_size: 50,
    });

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/events/77/prize-distribution?gender=male&bike_type=gravel&match_reason=place&page=2&page_size=50'
    );
  });

  it('omits all filters from the query string', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ distribution: [], total: 0 }), { status: 200 })
    );
    vi.stubGlobal('fetch', fetchMock);

    await prizeDistributionApi.getPrizeDistribution(77, {
      gender: 'all',
      bike_type: 'all',
      match_reason: 'all',
    });

    expect(fetchMock.mock.calls[0][0]).toBe(
      'http://localhost:8080/api/events/77/prize-distribution'
    );
  });
});
