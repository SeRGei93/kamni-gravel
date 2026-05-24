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
