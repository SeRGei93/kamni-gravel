import { get } from './client';
import type {
  PrizeDistributionListResponse,
  PageSize,
  ResultListResponse,
} from '@/types';

const EVENTS_PREFIX = '/api/events';

export const prizeDistributionApi = {
  // Без params возвращает всё распределение (для страницы призов).
  // С page/page_size — серверная пагинация; match_reason — серверный фильтр.
  async getPrizeDistribution(
    eventId: number,
    params?: {
      match_reason?: string;
      page?: number;
      page_size?: PageSize;
    }
  ): Promise<PrizeDistributionListResponse> {
    const search = new URLSearchParams();
    if (params?.match_reason) search.append('match_reason', params.match_reason);
    if (params?.page !== undefined) search.append('page', String(params.page));
    if (params?.page_size !== undefined)
      search.append('page_size', String(params.page_size));
    const query = search.toString();
    return get<PrizeDistributionListResponse>(
      `${EVENTS_PREFIX}/${eventId}/prize-distribution${query ? `?${query}` : ''}`
    );
  },

  async getResultsWithPlaces(eventId: number): Promise<ResultListResponse> {
    return get<ResultListResponse>(`${EVENTS_PREFIX}/${eventId}/results`);
  },
};
