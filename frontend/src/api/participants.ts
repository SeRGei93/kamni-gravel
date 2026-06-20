import { get, put, del } from './client';
import type {
  Participant,
  ParticipantDetail,
  ParticipantListResponse,
  UpdateParticipantRequest,
  GiftListResponse,
  PrizeAssignmentListResponse,
} from '@/types';

const PARTICIPANTS_PREFIX = '/api/participants';
const EVENTS_PREFIX = '/api/events';

export const participantsApi = {
  // getByEvent возвращает ВСЕХ участников события (без пагинации) — для номинаций и
  // глобального поиска. Не передавайте page/page_size.
  async getByEvent(
    eventId: number,
    filters?: {
      bike_type?: string;
      gender?: string;
      is_finished?: boolean;
    }
  ): Promise<ParticipantListResponse> {
    const params = new URLSearchParams();
    if (filters?.bike_type) params.append('bike_type', filters.bike_type);
    if (filters?.gender) params.append('gender', filters.gender);
    if (filters?.is_finished !== undefined)
      params.append('is_finished', String(filters.is_finished));

    const query = params.toString();
    return get<ParticipantListResponse>(
      `${EVENTS_PREFIX}/${eventId}/participants${query ? `?${query}` : ''}`
    );
  },

  // listByEvent возвращает страницу участников (серверная пагинация + все фильтры/поиск).
  async listByEvent(
    eventId: number,
    params: {
      bike_type?: string;
      gender?: string;
      is_finished?: boolean;
      has_gift?: boolean;
      q?: string;
      page: number;
      page_size: number;
    }
  ): Promise<ParticipantListResponse> {
    const search = new URLSearchParams();
    if (params.bike_type) search.append('bike_type', params.bike_type);
    if (params.gender) search.append('gender', params.gender);
    if (params.is_finished !== undefined)
      search.append('is_finished', String(params.is_finished));
    if (params.has_gift !== undefined)
      search.append('has_gift', String(params.has_gift));
    if (params.q) search.append('q', params.q);
    search.append('page', String(params.page));
    search.append('page_size', String(params.page_size));
    return get<ParticipantListResponse>(
      `${EVENTS_PREFIX}/${eventId}/participants?${search.toString()}`
    );
  },

  async getById(id: number): Promise<ParticipantDetail> {
    return get<ParticipantDetail>(`${PARTICIPANTS_PREFIX}/${id}`);
  },

  async getGifts(id: number): Promise<GiftListResponse> {
    return get<GiftListResponse>(`${PARTICIPANTS_PREFIX}/${id}/gifts`);
  },

  async update(
    id: number,
    data: UpdateParticipantRequest
  ): Promise<Participant> {
    return put<Participant>(`${PARTICIPANTS_PREFIX}/${id}`, data);
  },

  async delete(id: number): Promise<void> {
    return del<void>(`${PARTICIPANTS_PREFIX}/${id}`);
  },
};
