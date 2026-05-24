import { get, post, put, del } from './client';
import type {
  Result,
  ResultListResponse,
  CreateResultRequest,
  UpdateResultRequest,
} from '@/types';

const RESULTS_PREFIX = '/api/results';
const PARTICIPANTS_PREFIX = '/api/participants';

export const resultsApi = {
  async getByParticipant(participantId: number): Promise<ResultListResponse> {
    return get<ResultListResponse>(
      `${PARTICIPANTS_PREFIX}/${participantId}/results`
    );
  },

  async getById(id: number): Promise<Result> {
    return get<Result>(`${RESULTS_PREFIX}/${id}`);
  },

  async create(
    participantId: number,
    data: CreateResultRequest
  ): Promise<Result> {
    return post<Result>(
      `${PARTICIPANTS_PREFIX}/${participantId}/results`,
      data
    );
  },

  async update(id: number, data: UpdateResultRequest): Promise<Result> {
    return put<Result>(`${RESULTS_PREFIX}/${id}`, data);
  },

  async delete(id: number): Promise<void> {
    return del<void>(`${RESULTS_PREFIX}/${id}`);
  },

  async addCriteria(resultId: number, criteriaId: number): Promise<Result> {
    return post<Result>(`${RESULTS_PREFIX}/${resultId}/criteria`, {
      criteria_id: criteriaId,
    });
  },

  async removeCriteria(
    resultId: number,
    criteriaId: number
  ): Promise<Result> {
    return del<Result>(
      `${RESULTS_PREFIX}/${resultId}/criteria/${criteriaId}`
    );
  },
};
