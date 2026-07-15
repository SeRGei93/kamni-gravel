import type { Gift, ManualGift, PageSize } from '@/types';
import {
  attachManualGiftAssignments,
  filterGiftsByDistribution,
  type GiftDistributionFilter,
} from './manualGiftAssignment';

export interface BuildFilteredGiftListParams {
  gifts: Gift[];
  manualGifts: ManualGift[];
  distributionFilter: GiftDistributionFilter;
  assignedGiftIds: Set<number>;
}

export function buildFilteredGiftList({
  gifts,
  manualGifts,
  distributionFilter,
  assignedGiftIds,
}: BuildFilteredGiftListParams): Gift[] {
  const giftsWithManualAssignments = attachManualGiftAssignments(gifts, manualGifts);
  return filterGiftsByDistribution(
    giftsWithManualAssignments,
    distributionFilter,
    assignedGiftIds
  );
}

export function paginateGifts(
  gifts: Gift[],
  page: number,
  pageSize: PageSize
): Gift[] {
  if (pageSize === 'all') return gifts;

  const offset = (page - 1) * pageSize;
  return gifts.slice(offset, offset + pageSize);
}

export function isCurrentGiftListRequest(
  requestVersion: number,
  latestRequestVersion: number
): boolean {
  return requestVersion === latestRequestVersion;
}

export function shouldSettleGiftListRequest(
  requestVersion: number,
  latestRequestVersion: number,
  isMounted: boolean
): boolean {
  return isMounted && isCurrentGiftListRequest(requestVersion, latestRequestVersion);
}
