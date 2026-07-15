import type { PrizeDistribution } from '@/types';

export interface AutomaticGiftRecipient {
  participantID: number;
  participantName: string;
}

export function getAutomaticGiftRecipient(
  distribution: PrizeDistribution[],
  giftID: number
): AutomaticGiftRecipient | undefined {
  const assignment = distribution.find(
    (participant) =>
      participant.matched_gift_assignments?.some(
        (giftAssignment) => giftAssignment.gift_id === giftID
      ) || participant.matched_gifts?.some((gift) => gift.id === giftID)
  );

  if (!assignment) {
    return undefined;
  }

  return {
    participantID: assignment.participant_id,
    participantName: assignment.participant_name,
  };
}
