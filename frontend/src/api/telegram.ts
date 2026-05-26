import { get } from './client';

const TELEGRAM_PREFIX = '/api/telegram';

export interface TelegramFileInfo {
  file_id: string;
  file_path: string;
  file_size?: number;
  url: string;
}

export interface TelegramFileURLResponse {
  url: string;
}

export const telegramApi = {
  async getFileURL(fileId: string): Promise<string> {
    const response = await get<TelegramFileURLResponse>(
      `${TELEGRAM_PREFIX}/files/${encodeURIComponent(fileId)}`
    );
    return response.url;
  },

  async getFileInfo(fileId: string): Promise<TelegramFileInfo> {
    return get<TelegramFileInfo>(
      `${TELEGRAM_PREFIX}/files/${encodeURIComponent(fileId)}/info`
    );
  },
};
