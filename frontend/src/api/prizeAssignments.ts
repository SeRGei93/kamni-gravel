import { get, post, del } from './client';
import type {
  PrizeAssignment,
  PrizeAssignmentListResponse,
  CreatePrizeAssignmentRequest,
} from '@/types';

const PRIZE_ASSIGNMENTS_PREFIX = '/api/prize-assignments';
const EVENTS_PREFIX = '/api/events';
const PARTICIPANTS_PREFIX = '/api/participants';

export const prizeAssignmentsApi = {
  async getByEvent(eventId: number): Promise<PrizeAssignmentListResponse> {
    return get<PrizeAssignmentListResponse>(
      `${EVENTS_PREFIX}/${eventId}/prize-assignments`
    );
  },

  async getByParticipant(
    participantId: number
  ): Promise<PrizeAssignmentListResponse> {
    return get<PrizeAssignmentListResponse>(
      `${PARTICIPANTS_PREFIX}/${participantId}/prize-assignments`
    );
  },

  async getById(id: number): Promise<PrizeAssignment> {
    return get<PrizeAssignment>(`${PRIZE_ASSIGNMENTS_PREFIX}/${id}`);
  },

  async create(
    data: CreatePrizeAssignmentRequest
  ): Promise<PrizeAssignment> {
    return post<PrizeAssignment>(PRIZE_ASSIGNMENTS_PREFIX, data);
  },

  async delete(id: number): Promise<void> {
    return del<void>(`${PRIZE_ASSIGNMENTS_PREFIX}/${id}`);
  },
};
