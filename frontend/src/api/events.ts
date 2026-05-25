import { get, post, postForm, put, del } from './client';
import type {
  Event,
  EventListResponse,
  CreateEventRequest,
  UpdateEventRequest,
} from '@/types';

const EVENTS_PREFIX = '/api/events';

export const eventsApi = {
  async getAll(): Promise<EventListResponse> {
    return get<EventListResponse>(EVENTS_PREFIX);
  },

  async getById(id: number): Promise<Event> {
    return get<Event>(`${EVENTS_PREFIX}/${id}`);
  },

  async create(data: CreateEventRequest): Promise<Event> {
    return post<Event>(EVENTS_PREFIX, data);
  },

  async update(id: number, data: UpdateEventRequest): Promise<Event> {
    return put<Event>(`${EVENTS_PREFIX}/${id}`, data);
  },

  async uploadGpxFile(id: number, file: File): Promise<Event> {
    const formData = new FormData();
    formData.append('file', file);
    return postForm<Event>(`${EVENTS_PREFIX}/${id}/gpx-file`, formData);
  },

  async delete(id: number): Promise<void> {
    return del<void>(`${EVENTS_PREFIX}/${id}`);
  },
};
