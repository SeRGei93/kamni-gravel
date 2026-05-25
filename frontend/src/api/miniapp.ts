import type {
  BikeTypeFilter,
  GenderFilter,
  GiftListResponse,
  MiniappSessionResponse,
} from '@/types';
import { getTelegramInitData } from '@/utils/telegramWebApp';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const MINIAPP_PREFIX = '/api/miniapp';
const TELEGRAM_INIT_DATA_HEADER = 'X-Telegram-Init-Data';

export class MiniappApiError extends Error {
  constructor(
    public status: number,
    public statusText: string,
    public data?: unknown
  ) {
    super(`Miniapp API Error: ${status} ${statusText}`);
    this.name = 'MiniappApiError';
  }
}

export interface MiniappGiftFilters {
  gender?: GenderFilter;
  bike_type?: BikeTypeFilter;
}

async function miniappRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const response = await miniappFetch(endpoint, options);

  if (response.status === 204 || response.headers.get('content-length') === '0') {
    return {} as T;
  }

  return response.json() as Promise<T>;
}

async function miniappBlobRequest(
  endpoint: string,
  options: RequestInit = {}
): Promise<Blob> {
  const response = await miniappFetch(endpoint, options);
  return response.blob();
}

async function miniappFetch(
  endpoint: string,
  options: RequestInit = {}
): Promise<Response> {
  const url = `${API_URL}${endpoint}`;
  const initData = getTelegramInitData();
  const headers = new Headers(options.headers);

  if (!headers.has(TELEGRAM_INIT_DATA_HEADER)) {
    headers.set(TELEGRAM_INIT_DATA_HEADER, initData);
  }
  if (options.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  try {
    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      let errorData: unknown;
      try {
        errorData = await response.json();
      } catch {
        errorData = { error: response.statusText };
      }

      throw new MiniappApiError(response.status, response.statusText, errorData);
    }

    return response;
  } catch (error) {
    if (error instanceof MiniappApiError) {
      console.error('[miniapp] API request failed', {
        endpoint,
        status: error.status,
        statusText: error.statusText,
      });
      throw error;
    }

    console.error('[miniapp] API network failure', {
      endpoint,
      message: error instanceof Error ? error.message : 'Unknown error',
    });
    throw new Error(`Miniapp network error: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
}

function buildGiftQuery(filters: MiniappGiftFilters = {}): string {
  const params = new URLSearchParams();
  if (filters.gender) {
    params.set('gender', filters.gender);
  }
  if (filters.bike_type) {
    params.set('bike_type', filters.bike_type);
  }

  const query = params.toString();
  return query ? `?${query}` : '';
}

export const miniappApi = {
  async getSession(): Promise<MiniappSessionResponse> {
    return miniappRequest<MiniappSessionResponse>(`${MINIAPP_PREFIX}/session`, {
      method: 'GET',
    });
  },

  async getGifts(filters: MiniappGiftFilters = {}): Promise<GiftListResponse> {
    return miniappRequest<GiftListResponse>(
      `${MINIAPP_PREFIX}/gifts${buildGiftQuery(filters)}`,
      { method: 'GET' }
    );
  },

  async getTelegramFile(fileId: string): Promise<Blob> {
    return miniappBlobRequest(
      `${MINIAPP_PREFIX}/telegram/files/${encodeURIComponent(fileId)}`,
      { method: 'GET' }
    );
  },
};
