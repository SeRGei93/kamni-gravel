import { get, post } from './client';
import type {
  AdminUser,
  AdminUserListResponse,
  CreateAdminRequest,
} from '@/types';

const ADMIN_USERS_PREFIX = '/api/admin-users';

export const adminUsersApi = {
  async getAll(): Promise<AdminUserListResponse> {
    return get<AdminUserListResponse>(ADMIN_USERS_PREFIX);
  },

  async create(data: CreateAdminRequest): Promise<AdminUser> {
    return post<AdminUser>(ADMIN_USERS_PREFIX, data);
  },
};
