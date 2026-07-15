import type { ParticipantListParams, SortOrder } from '@/api/participants';
import type { PageSize } from '@/types';
import type { HasGiftFilter } from '@/utils/participants';

export interface ParticipantListQueryInput {
  gender: string;
  bikeType: string;
  isFinished: string;
  hasGift: HasGiftFilter;
  criteriaId: string;
  q: string;
  sortKey: string | null;
  sortOrder: SortOrder;
  page: number;
  pageSize: PageSize;
}

function hasGiftFilterToParam(value: HasGiftFilter): boolean | undefined {
  if (value === 'yes') return true;
  if (value === 'no') return false;
  return undefined;
}

function criteriaIDFilterToParam(value: string): number | undefined {
  const criteriaID = Number(value);
  return Number.isSafeInteger(criteriaID) && criteriaID > 0
    ? criteriaID
    : undefined;
}

/**
 * Собирает параметры серверного списка из состояния страницы. Таблица и экспорт
 * используют один контракт, поэтому фильтры и сортировка не расходятся.
 */
export function buildParticipantListParams(
  input: ParticipantListQueryInput,
): ParticipantListParams {
  return {
    gender: input.gender || undefined,
    bike_type: input.bikeType || undefined,
    is_finished:
      input.isFinished === '' ? undefined : input.isFinished === 'true',
    has_gift: hasGiftFilterToParam(input.hasGift),
    criteria_id: criteriaIDFilterToParam(input.criteriaId),
    q: input.q || undefined,
    sort: input.sortKey ?? undefined,
    order: input.sortKey ? input.sortOrder : undefined,
    page: input.page,
    page_size: input.pageSize,
  };
}
