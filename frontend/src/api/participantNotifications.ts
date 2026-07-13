import { get, post } from './client';

const PARTICIPANT_NOTIFICATIONS_PREFIX = '/api/participant-notifications';

export type ParticipantNotificationFilter =
  | 'all'
  | 'finished_without_gift'
  | 'gift_without_finish'
  | 'unassigned_gifts';

export interface ParticipantNotificationRecipient {
  user_id: number;
  label: string;
  username?: string;
  status: string;
  has_gift: boolean;
  has_unassigned_gifts: boolean;
}

export interface ParticipantNotificationRecipientsResponse {
  event_name: string;
  filter: ParticipantNotificationFilter;
  recipients: ParticipantNotificationRecipient[];
}

export type ParticipantNotificationJobStatus =
  | 'queued'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled';

export interface ParticipantNotificationJob {
  id: string;
  status: ParticipantNotificationJobStatus;
  requested: number;
  sent: number;
  failed: number;
  skipped: number;
  error?: string;
}

export const participantNotificationsApi = {
  async getRecipients(
    filter: ParticipantNotificationFilter,
  ): Promise<ParticipantNotificationRecipientsResponse> {
    const params = new URLSearchParams({ filter });
    return get<ParticipantNotificationRecipientsResponse>(
      `${PARTICIPANT_NOTIFICATIONS_PREFIX}/recipients?${params.toString()}`,
    );
  },

  async send(
    userIds: number[],
    text: string,
  ): Promise<ParticipantNotificationJob> {
    return post<ParticipantNotificationJob>(
      `${PARTICIPANT_NOTIFICATIONS_PREFIX}/send`,
      { user_ids: userIds, text },
    );
  },

  async getJob(jobId: string): Promise<ParticipantNotificationJob> {
    return get<ParticipantNotificationJob>(
      `${PARTICIPANT_NOTIFICATIONS_PREFIX}/jobs/${encodeURIComponent(jobId)}`,
    );
  },
};
