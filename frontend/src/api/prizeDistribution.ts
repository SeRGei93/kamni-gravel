import { get } from './client';
import type {
  BikeTypeFilter,
  GenderFilter,
  PrizeDistributionListResponse,
  PageSize,
  ResultListResponse,
} from '@/types';

const EVENTS_PREFIX = '/api/events';

export const prizeDistributionApi = {
  // Без params возвращает всё распределение (для страницы призов).
  // С page/page_size — серверная пагинация; остальные параметры фильтруют
  // уже рассчитанное автоматическое распределение для его отображения.
  async getPrizeDistribution(
    eventId: number,
    params?: {
      gender?: GenderFilter;
      bike_type?: BikeTypeFilter;
      match_reason?: string;
      page?: number;
      page_size?: PageSize;
    }
  ): Promise<PrizeDistributionListResponse> {
    const search = new URLSearchParams();
    if (params?.gender && params.gender !== 'all') {
      search.append('gender', params.gender);
    }
    if (params?.bike_type && params.bike_type !== 'all') {
      search.append('bike_type', params.bike_type);
    }
    if (params?.match_reason && params.match_reason !== 'all') {
      search.append('match_reason', params.match_reason);
    }
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
