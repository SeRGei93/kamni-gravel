import { get, post, put } from './client';
import type {
  ChangePasswordRequest,
  LoginRequest,
  LoginResponse,
  User,
  TokenPair,
} from '@/types';

const AUTH_PREFIX = '/api/auth';

export const authApi = {
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    return post<LoginResponse>(`${AUTH_PREFIX}/login`, credentials);
  },

  async refresh(refreshToken: string): Promise<TokenPair> {
    return post<TokenPair>(`${AUTH_PREFIX}/refresh`, {
      refresh_token: refreshToken,
    });
  },

  async me(): Promise<User> {
    return get<User>(`${AUTH_PREFIX}/me`);
  },

  async changePassword(data: ChangePasswordRequest): Promise<void> {
    return put<void>(`${AUTH_PREFIX}/me/password`, data);
  },
};
