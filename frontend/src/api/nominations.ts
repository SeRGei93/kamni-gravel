import { get, post, put, del } from './client';
import type {
  Nomination,
  NominationListResponse,
  CreateNominationRequest,
  UpdateNominationRequest,
} from '@/types';

const NOMINATIONS_PREFIX = '/api/nominations';
const EVENTS_PREFIX = '/api/events';

export const nominationsApi = {
  async getByEvent(eventId: number): Promise<NominationListResponse> {
    return get<NominationListResponse>(
      `${EVENTS_PREFIX}/${eventId}/nominations`
    );
  },

  async getById(id: number): Promise<Nomination> {
    return get<Nomination>(`${NOMINATIONS_PREFIX}/${id}`);
  },

  async create(data: CreateNominationRequest): Promise<Nomination> {
    return post<Nomination>(
      `${EVENTS_PREFIX}/${data.event_id}/nominations`,
      data
    );
  },

  async update(id: number, data: UpdateNominationRequest): Promise<Nomination> {
    return put<Nomination>(`${NOMINATIONS_PREFIX}/${id}`, data);
  },

  async delete(id: number): Promise<void> {
    return del<void>(`${NOMINATIONS_PREFIX}/${id}`);
  },
};
