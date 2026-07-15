import React from 'react';
import Link from 'next/link';

import Badge from '@/components/ui/badge/Badge';
import { PARTICIPANT_STATUS_LABELS } from '@/types';
import type { PrizeDistribution, PrizeGiftAssignment } from '@/types';
import { formatPrizeAssignment } from '@/utils/giftPlaceRule';
import { giftDonorName } from '@/utils/prizeDistribution';

export interface PrizeDistributionColumn {
  key: string;
  label: string;
  alwaysVisible?: boolean;
  defaultVisible: boolean;
  align?: 'start' | 'end' | 'center';
  render: (distribution: PrizeDistribution) => React.ReactNode;
}

const GENDER_LABELS: Record<string, string> = {
  male: 'Мужчины',
  female: 'Женщины',
};

const BIKE_TYPE_LABELS: Record<string, string> = {
  gravel: 'Gravel',
  mtb: 'MTB',
  road: 'Шоссе',
  single_speed: 'Single Speed',
  tandem: 'Тандем',
};

const MATCH_REASON_LABELS: Record<string, string> = {
  criteria: 'По критериям',
  place: 'По месту',
  match: 'Без ограничений',
  no_match: 'Без приза',
};

const FALLBACK_REASON_LABELS: Record<string, string> = {
  target_unavailable: 'целевое место недоступно',
  target_out_of_range: 'целевое место вне списка участников',
};

const text = 'text-gray-800 text-theme-sm dark:text-white/90';
const muted = 'text-gray-500 text-theme-sm dark:text-gray-400';

function textOrDash(value?: string | number | null): React.ReactNode {
  if (value === undefined || value === null || value === '') {
    return <span className={muted}>-</span>;
  }
  return <span className={text}>{value}</span>;
}

function statusBadge(status: PrizeDistribution['status']): React.ReactNode {
  if (status === 'active') {
    return <Badge color="success" size="sm">{PARTICIPANT_STATUS_LABELS.active}</Badge>;
  }
  return (
    <Badge color={status === 'disqualified' ? 'error' : 'warning'} size="sm">
      {PARTICIPANT_STATUS_LABELS[status]}
    </Badge>
  );
}

function assignmentDescription(assignment: PrizeGiftAssignment): string {
  const detail = formatPrizeAssignment(assignment);
  if (!assignment.is_fallback || !assignment.fallback_reason) {
    return detail;
  }
  const fallbackReason =
    FALLBACK_REASON_LABELS[assignment.fallback_reason] ?? 'назначен ближайший доступный участник';
  return `${detail} (${fallbackReason})`;
}

function PrizeListCell({ row }: { row: PrizeDistribution }) {
  const assignments = row.matched_gift_assignments ?? [];
  if (assignments.length > 0) {
    return (
      <div className="min-w-56 space-y-2">
        {assignments.map((assignment, index) => (
          <div
            key={`${assignment.gift_id}-${index}`}
            className="border-l-2 border-brand-200 pl-2 dark:border-brand-700"
          >
            <p className="font-medium text-gray-800 text-theme-sm dark:text-white/90">
              {assignment.gift.description}
            </p>
            <p className={muted}>от {giftDonorName(assignment.gift)}</p>
            <p className="text-theme-xs text-gray-500 dark:text-gray-400">
              {assignmentDescription(assignment)}
            </p>
          </div>
        ))}
      </div>
    );
  }

  const legacyGifts = row.matched_gifts ?? [];
  if (legacyGifts.length > 0) {
    return (
      <div className="min-w-56 space-y-2">
        {legacyGifts.map((gift) => (
          <div
            key={gift.id}
            className="border-l-2 border-brand-200 pl-2 dark:border-brand-700"
          >
            <p className="font-medium text-gray-800 text-theme-sm dark:text-white/90">
              {gift.description}
            </p>
            <p className={muted}>от {giftDonorName(gift)}</p>
          </div>
        ))}
      </div>
    );
  }

  return <span className={muted}>-</span>;
}

export const PRIZE_DISTRIBUTION_COLUMNS: PrizeDistributionColumn[] = [
  {
    key: 'display_place',
    label: 'Место',
    alwaysVisible: true,
    defaultVisible: true,
    align: 'center',
    render: (row) => (
      <span className="font-semibold text-gray-800 text-theme-sm dark:text-white/90">
        {row.display_place ?? '-'}
      </span>
    ),
  },
  {
    key: 'participant',
    label: 'Участник',
    alwaysVisible: true,
    defaultVisible: true,
    render: (row) => (
      <Link
        href={`/participants/${row.participant_id}`}
        className="font-medium text-brand-500 hover:text-brand-600 dark:text-brand-400 dark:hover:text-brand-300"
      >
        {row.participant_name}
      </Link>
    ),
  },
  {
    key: 'gender',
    label: 'Пол',
    defaultVisible: true,
    render: (row) => <Badge color={row.gender === 'male' ? 'info' : 'warning'} size="sm">{GENDER_LABELS[row.gender] ?? row.gender}</Badge>,
  },
  {
    key: 'bike_type',
    label: 'Зачёт',
    defaultVisible: true,
    render: (row) => <Badge color="light" size="sm">{BIKE_TYPE_LABELS[row.bike_type] ?? row.bike_type}</Badge>,
  },
  {
    key: 'status',
    label: 'Статус',
    defaultVisible: true,
    render: (row) => statusBadge(row.status),
  },
  {
    key: 'place_absolute',
    label: 'Абсолют',
    defaultVisible: false,
    align: 'center',
    render: (row) => textOrDash(row.place_absolute || undefined),
  },
  {
    key: 'place_by_gender',
    label: 'Место по полу',
    defaultVisible: false,
    align: 'center',
    render: (row) => textOrDash(row.place_by_gender || undefined),
  },
  {
    key: 'place_by_gender_bike',
    label: 'Место по зачёту',
    defaultVisible: false,
    align: 'center',
    render: (row) => textOrDash(row.place_by_gender_bike || undefined),
  },
  {
    key: 'criteria',
    label: 'Критерии',
    defaultVisible: false,
    render: (row) =>
      row.result_criteria.length > 0 ? (
        <div className="flex min-w-48 flex-wrap gap-1">
          {row.result_criteria.map((criterion) => (
            <Badge key={criterion.id} color="light" size="sm">
              {criterion.name}
            </Badge>
          ))}
        </div>
      ) : (
        <span className={muted}>-</span>
      ),
  },
  {
    key: 'prizes',
    label: 'Призы',
    alwaysVisible: true,
    defaultVisible: true,
    render: (row) => <PrizeListCell row={row} />,
  },
  {
    key: 'match_reason',
    label: 'Причина',
    defaultVisible: true,
    render: (row) => textOrDash(MATCH_REASON_LABELS[row.match_reason] ?? row.match_reason),
  },
];

export const PRIZE_DISTRIBUTION_COLUMNS_STORAGE_KEY = 'prize-distribution:visible-columns';

export const PRIZE_DISTRIBUTION_TOGGLEABLE_COLUMN_KEYS = PRIZE_DISTRIBUTION_COLUMNS
  .filter((column) => !column.alwaysVisible)
  .map((column) => column.key);

export const PRIZE_DISTRIBUTION_DEFAULT_VISIBLE_KEYS = PRIZE_DISTRIBUTION_COLUMNS
  .filter((column) => !column.alwaysVisible && column.defaultVisible)
  .map((column) => column.key);
