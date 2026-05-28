import type { PrizeDistribution, UnassignedPrizeSlot } from '@/types';

export function participantHasPrize(row: PrizeDistribution): boolean {
  return (
    (row.matched_gift_assignments?.length ?? 0) > 0 ||
    (row.matched_gifts?.length ?? 0) > 0
  );
}

export function countParticipantsWithPrizes(
  distribution: PrizeDistribution[]
): number {
  return distribution.filter(participantHasPrize).length;
}

export function countPrizeAssignmentSlots(
  distribution: PrizeDistribution[]
): number {
  return distribution.reduce((total, row) => {
    if ((row.matched_gift_assignments?.length ?? 0) > 0) {
      return total + (row.matched_gift_assignments?.length ?? 0);
    }
    return total + (row.matched_gifts?.length ?? 0);
  }, 0);
}

export function formatUnassignedPrizeSlot(slot: UnassignedPrizeSlot): string {
  return `Приз ${slot.gift_id}, место ${slot.target_rank || '-'}`;
}
