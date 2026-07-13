import { get, post, put, del } from './client';
import type {
  CreateGiftRequest,
  Gift,
  GiftListResponse,
  GiftReviewStatus,
  ManualGiftListResponse,
  PageSize,
  UpdateGiftRequest,
} from '@/types';

const GIFTS_PREFIX = '/api/gifts';
const EVENTS_PREFIX = '/api/events';

export interface GiftListFilters {
  review_status?: GiftReviewStatus;
  owner_user_id?: number;
  q?: string;
}

function appendGiftListFilters(search: URLSearchParams, filters: GiftListFilters): void {
  if (filters.review_status) search.set('review_status', filters.review_status);
  if (filters.owner_user_id) search.set('owner_user_id', String(filters.owner_user_id));
  if (filters.q) search.set('q', filters.q);
}

export const giftsApi = {
  // getByEvent возвращает ВСЕ подарки события (без пагинации) — для модалок выбора
  // приза и т.п. Не передавайте page/page_size.
  async getByEvent(
    eventId: number,
    filters: GiftListFilters = {}
  ): Promise<GiftListResponse> {
    const params = new URLSearchParams();
    appendGiftListFilters(params, filters);
    const query = params.toString();
    return get<GiftListResponse>(
      `${EVENTS_PREFIX}/${eventId}/gifts${query ? `?${query}` : ''}`
    );
  },

  // listByEvent возвращает страницу подарков (серверная пагинация) + status_counts.
  async listByEvent(params: {
    eventId: number;
    review_status?: GiftReviewStatus;
    owner_user_id?: number;
    q?: string;
    page: number;
    page_size: PageSize;
  }): Promise<GiftListResponse> {
    const search = new URLSearchParams();
    appendGiftListFilters(search, params);
    search.set('page', String(params.page));
    search.set('page_size', String(params.page_size));
    return get<GiftListResponse>(
      `${EVENTS_PREFIX}/${params.eventId}/gifts?${search.toString()}`
    );
  },

  async getById(id: number): Promise<Gift> {
    return get<Gift>(`${GIFTS_PREFIX}/${id}`);
  },

  // Protected administrator enrichment. Public gift DTOs deliberately never
  // contain manual recipient identity.
  async getManualByEvent(eventId: number): Promise<ManualGiftListResponse> {
    return get<ManualGiftListResponse>(
      `${EVENTS_PREFIX}/${eventId}/manual-gifts`
    );
  },

  async create(eventId: number, data: CreateGiftRequest): Promise<Gift> {
    return post<Gift>(`${EVENTS_PREFIX}/${eventId}/gifts`, data);
  },

  async update(id: number, data: UpdateGiftRequest): Promise<Gift> {
    return put<Gift>(`${GIFTS_PREFIX}/${id}`, data);
  },

  // The server chooses an eligible participant without a prize. The request
  // deliberately has no body so the client cannot influence the recipient.
  async assignRandomRecipient(id: number): Promise<void> {
    await post<void>(`${GIFTS_PREFIX}/${id}/random-recipient`);
  },

  async delete(id: number): Promise<void> {
    return del<void>(`${GIFTS_PREFIX}/${id}`);
  },
};
