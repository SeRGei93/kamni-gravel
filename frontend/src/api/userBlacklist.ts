import { del, get, post, put } from './client';
import type {
  CreateUserBlacklistRequest,
  UpdateUserBlacklistRequest,
  UserBlacklistEntry,
  UserBlacklistListResponse,
} from '@/types';

const USER_BLACKLIST_PREFIX = '/api/user-blacklist';

export const userBlacklistApi = {
  async getAll(): Promise<UserBlacklistListResponse> {
    return get<UserBlacklistListResponse>(USER_BLACKLIST_PREFIX);
  },

  async add(data: CreateUserBlacklistRequest): Promise<UserBlacklistEntry> {
    return post<UserBlacklistEntry>(USER_BLACKLIST_PREFIX, data);
  },

  async updateReason(
    telegramUserId: number,
    data: UpdateUserBlacklistRequest
  ): Promise<UserBlacklistEntry> {
    return put<UserBlacklistEntry>(
      `${USER_BLACKLIST_PREFIX}/${telegramUserId}`,
      data
    );
  },

  async remove(telegramUserId: number): Promise<void> {
    return del<void>(`${USER_BLACKLIST_PREFIX}/${telegramUserId}`);
  },
};
