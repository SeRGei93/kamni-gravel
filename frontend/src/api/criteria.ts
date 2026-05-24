import { get, post, put, del } from './client';
import type {
  Criteria,
  CriteriaListResponse,
  CreateCriteriaRequest,
  UpdateCriteriaRequest,
} from '@/types';

const CRITERIA_PREFIX = '/api/criteria';

export const criteriaApi = {
  async getAll(type?: string): Promise<CriteriaListResponse> {
    const params = new URLSearchParams();
    if (type) params.append('type', type);
    const query = params.toString();
    return get<CriteriaListResponse>(`${CRITERIA_PREFIX}${query ? `?${query}` : ''}`);
  },

  async getById(id: number): Promise<Criteria> {
    return get<Criteria>(`${CRITERIA_PREFIX}/${id}`);
  },

  async create(data: CreateCriteriaRequest): Promise<Criteria> {
    return post<Criteria>(CRITERIA_PREFIX, data);
  },

  async update(id: number, data: UpdateCriteriaRequest): Promise<Criteria> {
    return put<Criteria>(`${CRITERIA_PREFIX}/${id}`, data);
  },

  async delete(id: number): Promise<void> {
    return del<void>(`${CRITERIA_PREFIX}/${id}`);
  },
};
