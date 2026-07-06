import { get, post, postForm } from './client';

const CHAT_MEMBERS_PREFIX = '/api/chat-members';
const CHAT_PURGE_PREFIX = '/api/chat-purge';

export interface ChatMembersImportResult {
  imported: number;
  skipped_rows: number;
  total_in_table: number;
}

export interface ChatPurgeCandidate {
  user_id: number;
  label: string;
  username?: string;
  reason: string;
}

export interface ChatPurgeCandidatesResult {
  event_name: string;
  candidates: ChatPurgeCandidate[];
  protected_gift_owners: number;
}

export interface ChatPurgeExecuteResult {
  kicked: number;
  failed: number;
  skipped: number;
  protected: number;
}

export const chatMembersApi = {
  async importCsv(file: File): Promise<ChatMembersImportResult> {
    const formData = new FormData();
    formData.append('file', file);
    return postForm<ChatMembersImportResult>(`${CHAT_MEMBERS_PREFIX}/import`, formData);
  },

  async getCandidates(): Promise<ChatPurgeCandidatesResult> {
    return get<ChatPurgeCandidatesResult>(`${CHAT_PURGE_PREFIX}/candidates`);
  },

  async execute(userIds: number[]): Promise<ChatPurgeExecuteResult> {
    return post<ChatPurgeExecuteResult>(`${CHAT_PURGE_PREFIX}/execute`, {
      user_ids: userIds,
    });
  },
};
