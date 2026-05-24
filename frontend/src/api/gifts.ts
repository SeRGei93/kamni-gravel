import { get, post, put, del } from './client';
import type { Gift, GiftListResponse, CreateGiftRequest, UpdateGiftRequest } from '@/types';

const GIFTS_PREFIX = '/api/gifts';
const EVENTS_PREFIX = '/api/events';

export const giftsApi = {
  async getByEvent(eventId: number): Promise<GiftListResponse> {
    return get<GiftListResponse>(`${EVENTS_PREFIX}/${eventId}/gifts`);
  },

  async getById(id: number): Promise<Gift> {
    return get<Gift>(`${GIFTS_PREFIX}/${id}`);
  },

  async create(eventId: number, data: CreateGiftRequest): Promise<Gift> {
    return post<Gift>(`${EVENTS_PREFIX}/${eventId}/gifts`, data);
  },

  async update(id: number, data: UpdateGiftRequest): Promise<Gift> {
    return put<Gift>(`${GIFTS_PREFIX}/${id}`, data);
  },

  async delete(id: number): Promise<void> {
    return del<void>(`${GIFTS_PREFIX}/${id}`);
  },
};
