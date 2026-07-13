import type { Gift, ManualGift, UpdateGiftRequest } from '@/types';

export type ManualGiftPresentationStatus =
  | 'pending_review'
  | 'manual_unassigned'
  | 'manual_assigned'
  | 'automatic_assigned'
  | 'automatic_unassigned';

export interface ManualGiftStatusPresentation {
  status: ManualGiftPresentationStatus;
  label: string;
  color: 'warning' | 'success' | 'info' | 'error';
}

export function buildManualGiftUpdate(
  manualDistribution: boolean,
  recipientParticipantID: number | null
): Pick<
  UpdateGiftRequest,
  'manual_distribution' | 'manual_recipient_participant_id'
> {
  return {
    manual_distribution: manualDistribution,
    // Disabling always clears a prior recipient so the payload cannot
    // contradict the backend invariant.
    manual_recipient_participant_id: manualDistribution
      ? recipientParticipantID
      : null,
  };
}

export function attachManualGiftAssignments(
  gifts: Gift[],
  manualGifts: ManualGift[]
): Gift[] {
  const assignments = new Map(manualGifts.map((gift) => [gift.id, gift]));
  return gifts.map((gift) => ({
    ...gift,
    manual_assignment: assignments.get(gift.id),
  }));
}

export function getManualGiftStatus(
  gift: Gift,
  assignedGiftIds: Set<number>
): ManualGiftStatusPresentation {
  if (gift.review_status === 'pending_review') {
    return {
      status: 'pending_review',
      label: 'На проверке / Не участвует',
      color: 'warning',
    };
  }

  if (gift.manual_distribution) {
    if (gift.manual_assignment?.recipient) {
      return {
        status: 'manual_assigned',
        label: 'Ручной: получатель назначен',
        color: 'info',
      };
    }
    return {
      status: 'manual_unassigned',
      label: 'Ручной: ожидает назначения',
      color: 'warning',
    };
  }

  if (assignedGiftIds.has(gift.id)) {
    return {
      status: 'automatic_assigned',
      label: 'Автоматически назначен',
      color: 'success',
    };
  }

  return {
    status: 'automatic_unassigned',
    label: 'Автоматически не назначен',
    color: 'error',
  };
}

export function isGiftDistributed(gift: Gift, assignedGiftIds: Set<number>): boolean {
  const status = getManualGiftStatus(gift, assignedGiftIds).status;
  return status === 'manual_assigned' || status === 'automatic_assigned';
}

// canAssignRandomRecipient keeps the row action in sync with the server-side
// contract: only approved gifts that have no automatic or manual recipient
// can be distributed randomly.
export function canAssignRandomRecipient(
  gift: Gift,
  assignedGiftIds: Set<number>
): boolean {
  if (gift.review_status !== 'approved') {
    return false;
  }
  const status = getManualGiftStatus(gift, assignedGiftIds).status;
  return status === 'automatic_unassigned' || status === 'manual_unassigned';
}

export function formatManualRecipientSearchLabel(name: string, username?: string): string {
  const normalizedUsername = username?.replace(/^@+/, '').trim();
  return normalizedUsername ? `${name} (@${normalizedUsername})` : name;
}
