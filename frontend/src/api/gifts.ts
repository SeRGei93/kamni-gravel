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

export const giftsApi = {
  // getByEvent возвращает ВСЕ подарки события (без пагинации) — для модалок выбора
  // приза и т.п. Не передавайте page/page_size.
  async getByEvent(
    eventId: number,
    reviewStatus?: GiftReviewStatus
  ): Promise<GiftListResponse> {
    const params = new URLSearchParams();
    if (reviewStatus) {
      params.set('review_status', reviewStatus);
    }
    const query = params.toString();
    return get<GiftListResponse>(
      `${EVENTS_PREFIX}/${eventId}/gifts${query ? `?${query}` : ''}`
    );
  },

  // listByEvent возвращает страницу подарков (серверная пагинация) + status_counts.
  async listByEvent(params: {
    eventId: number;
    review_status?: GiftReviewStatus;
    page: number;
    page_size: PageSize;
  }): Promise<GiftListResponse> {
    const search = new URLSearchParams();
    if (params.review_status) {
      search.set('review_status', params.review_status);
    }
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

  async delete(id: number): Promise<void> {
    return del<void>(`${GIFTS_PREFIX}/${id}`);
  },
};
