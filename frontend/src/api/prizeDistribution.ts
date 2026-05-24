import { get } from './client';
import type {
  PrizeDistributionListResponse,
  ResultListResponse,
} from '@/types';

const EVENTS_PREFIX = '/api/events';

export const prizeDistributionApi = {
  async getPrizeDistribution(
    eventId: number
  ): Promise<PrizeDistributionListResponse> {
    return get<PrizeDistributionListResponse>(
      `${EVENTS_PREFIX}/${eventId}/prize-distribution`
    );
  },

  async getResultsWithPlaces(eventId: number): Promise<ResultListResponse> {
    return get<ResultListResponse>(`${EVENTS_PREFIX}/${eventId}/results`);
  },
};
