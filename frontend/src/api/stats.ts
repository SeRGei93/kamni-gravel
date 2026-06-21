import { get } from './client';
import type { Stats, StatsListResponse, EventDailyStats } from '@/types';

const STATS_PREFIX = '/api/stats';
const EVENTS_PREFIX = '/api/events';

export const statsApi = {
  async getAll(): Promise<StatsListResponse> {
    return get<StatsListResponse>(STATS_PREFIX);
  },

  async getByEvent(eventId: number): Promise<Stats> {
    return get<Stats>(`${EVENTS_PREFIX}/${eventId}/stats`);
  },

  async getDailyByEvent(eventId: number): Promise<EventDailyStats> {
    return get<EventDailyStats>(`${EVENTS_PREFIX}/${eventId}/stats/daily`);
  },
};
