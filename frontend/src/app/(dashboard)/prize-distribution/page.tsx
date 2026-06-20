'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { eventsApi } from '@/api/events';
import { prizeDistributionApi } from '@/api/prizeDistribution';
import { extractActiveEvent } from '@/utils/events';
import type {
  PrizeDistribution,
  PrizeDistributionStats,
  UnassignedPrizeSlot,
} from '@/types';
import Select from '@/components/form/Select';
import Label from '@/components/form/Label';
import Badge from '@/components/ui/badge/Badge';
import PaginationControls from '@/components/tables/PaginationControls';
import { usePaginationParams } from '@/hooks/usePaginationParams';
import { getCriteriaColor } from '@/utils/criteria';
import { formatPrizeAssignment } from '@/utils/giftPlaceRule';
import { formatUnassignedPrizeSlot } from '@/utils/prizeDistribution';
import Link from 'next/link';

const GENDER_LABELS: Record<string, string> = {
  male: 'М',
  female: 'Ж',
};

const BIKE_TYPE_LABELS: Record<string, string> = {
  gravel: 'Гравел',
  mtb: 'МТБ',
  road: 'Шоссе',
  single_speed: 'Фикс',
  tandem: 'Тандем',
};

const MATCH_REASON_LABELS: Record<string, string> = {
  criteria: 'По критериям',
  place: 'По месту',
  match: 'Совпадение',
  no_match: 'Нет совпадения',
};

const MATCH_REASON_COLORS: Record<string, 'success' | 'info' | 'warning' | 'light'> = {
  criteria: 'success',
  place: 'info',
  match: 'warning',
  no_match: 'light',
};

export default function PrizeDistributionPage() {
  const { page, pageSize, setPage, setPageSize } = usePaginationParams();

  const [activeEventId, setActiveEventId] = useState<number | null>(null);
  const [distribution, setDistribution] = useState<PrizeDistribution[]>([]);
  const [unassignedSlots, setUnassignedSlots] = useState<UnassignedPrizeSlot[]>([]);
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState<PrizeDistributionStats | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Фильтры
  const [matchReasonFilter, setMatchReasonFilter] = useState<string>('');

  const loadActiveEvent = useCallback(async () => {
    try {
      const response = await eventsApi.getActive();
      const activeEvent = extractActiveEvent(response);
      setActiveEventId(activeEvent?.id ?? null);
      if (!activeEvent) {
        setDistribution([]);
        setUnassignedSlots([]);
        setTotal(0);
        setStats(null);
        setError('Нет активного события');
      }
    } catch (err) {
      setActiveEventId(null);
      setDistribution([]);
      setUnassignedSlots([]);
      setTotal(0);
      setStats(null);
      setError('Ошибка загрузки активного события');
      console.error('Failed to load active event:', {
        operation: 'load_active_event',
        error: err,
      });
    }
  }, []);

  const loadDistribution = useCallback(async () => {
    if (!activeEventId) return;

    try {
      setIsLoading(true);
      setError(null);
      const response = await prizeDistributionApi.getPrizeDistribution(
        activeEventId,
        {
          match_reason: matchReasonFilter || undefined,
          page,
          page_size: pageSize,
        }
      );
      console.debug('[prize-distribution] loaded', {
        page,
        pageSize,
        total: response.total,
      });
      setDistribution(response.distribution);
      setUnassignedSlots(response.unassigned_slots ?? []);
      setTotal(response.total);
      setStats(response.stats ?? null);
    } catch (err) {
      setError('Ошибка загрузки распределения призов');
      console.error('Failed to load prize distribution:', {
        operation: 'load_prize_distribution',
        event_id: activeEventId,
        error: err,
      });
    } finally {
      setIsLoading(false);
    }
  }, [activeEventId, matchReasonFilter, page, pageSize]);

  useEffect(() => {
    loadActiveEvent();
  }, [loadActiveEvent]);

  useEffect(() => {
    if (activeEventId) {
      loadDistribution();
    } else {
      setDistribution([]);
      setUnassignedSlots([]);
      setTotal(0);
      setStats(null);
    }
  }, [loadDistribution, activeEventId]);

  // Сброс на первую страницу при смене фильтра (но не на первом рендере).
  const didMountRef = useRef(false);
  useEffect(() => {
    if (!didMountRef.current) {
      didMountRef.current = true;
      return;
    }
    if (page !== 1) setPage(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [matchReasonFilter]);

  // Распределение уже отфильтровано и постранично разбито на сервере.
  const filteredDistribution = distribution;

  // Статистика приходит с сервера (по всему распределению, не по странице).
  const totalParticipants = stats?.total_participants ?? 0;
  const withPrizes = stats?.with_prizes ?? 0;
  const withoutPrizes = stats?.without_prizes ?? 0;
  const totalPrizeAssignments = stats?.prize_slots ?? 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
          Распределение призов
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Автоматическое распределение призов по критериям и местам
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {/* Фильтры */}
      <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <Label>Тип совпадения</Label>
            <Select
              options={[
                { value: '', label: 'Все' },
                { value: 'criteria', label: 'По критериям' },
                { value: 'place', label: 'По месту' },
                { value: 'match', label: 'Совпадение' },
                { value: 'no_match', label: 'Нет совпадения' },
              ]}
              placeholder="Все"
              defaultValue={matchReasonFilter}
              onChange={setMatchReasonFilter}
            />
          </div>
        </div>
      </div>

      {/* Статистика */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
        <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Всего участников
          </p>
          <p className="mt-1 text-2xl font-semibold text-gray-800 dark:text-white">
            {totalParticipants}
          </p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            С призами
          </p>
          <p className="mt-1 text-2xl font-semibold text-success-600 dark:text-success-400">
            {withPrizes}
          </p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Призовых слотов
          </p>
          <p className="mt-1 text-2xl font-semibold text-brand-600 dark:text-brand-400">
            {totalPrizeAssignments}
          </p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Без призов
          </p>
          <p className="mt-1 text-2xl font-semibold text-gray-600 dark:text-gray-400">
            {withoutPrizes}
          </p>
        </div>
      </div>

      {unassignedSlots.length > 0 && (
        <div className="rounded-xl border border-warning-200 bg-warning-50 p-4 dark:border-warning-800 dark:bg-warning-900/20">
          <p className="text-sm font-semibold text-warning-700 dark:text-warning-300">
            Невыданные слоты: {unassignedSlots.length}
          </p>
          <div className="mt-2 flex flex-wrap gap-2">
            {unassignedSlots.map((slot, index) => (
              <Badge key={`${slot.gift_id}-${slot.target_rank || 'none'}-${index}`} color="warning" size="sm">
                {formatUnassignedPrizeSlot(slot)}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* Таблица */}
      <div className="rounded-xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-800">
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Участник
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Пол
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Тип
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Место (абс)
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Место (гендер)
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Место (гендер+тип)
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Критерии
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Приз
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Тип совпадения
                </th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr>
                  <td colSpan={9} className="px-4 py-8 text-center">
                    <div className="text-gray-500 dark:text-gray-400">
                      Загрузка...
                    </div>
                  </td>
                </tr>
              ) : filteredDistribution.length === 0 ? (
                <tr>
                  <td colSpan={9} className="px-4 py-8 text-center">
                    <div className="text-gray-500 dark:text-gray-400">
                      Нет данных
                    </div>
                  </td>
                </tr>
              ) : (
                filteredDistribution.map((dist) => (
                  <tr
                    key={dist.participant_id}
                    className="border-b border-gray-200 last:border-b-0 dark:border-gray-800"
                  >
                    <td className="px-4 py-3">
                      <Link
                        href={`/participants/${dist.participant_id}`}
                        className="text-sm font-medium text-brand-500 hover:text-brand-600 dark:text-brand-400"
                      >
                        {dist.participant_name}
                      </Link>
                    </td>
                    <td className="px-4 py-3">
                      <Badge
                        color={dist.gender === 'male' ? 'info' : 'warning'}
                        size="sm"
                      >
                        {GENDER_LABELS[dist.gender]}
                      </Badge>
                    </td>
                    <td className="px-4 py-3">
                      <Badge color="light" size="sm">
                        {BIKE_TYPE_LABELS[dist.bike_type]}
                      </Badge>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm font-medium text-gray-800 dark:text-white/90">
                        {dist.place_absolute}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm font-medium text-gray-800 dark:text-white/90">
                        {dist.place_by_gender}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm font-medium text-gray-800 dark:text-white/90">
                        {dist.place_by_gender_bike}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      {dist.result_criteria && dist.result_criteria.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {dist.result_criteria.map((c) => (
                            <Badge
                              key={c.id}
                              color={getCriteriaColor(c.criteria_type)}
                              size="sm"
                            >
                              {c.name}
                            </Badge>
                          ))}
                        </div>
                      ) : (
                        <span className="text-xs text-gray-500 dark:text-gray-400">
                          -
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      {dist.matched_gift_assignments && dist.matched_gift_assignments.length > 0 ? (
                        <div className="space-y-2 max-w-xs">
                          {dist.matched_gift_assignments.map((assignment, index) => (
                            <div key={`${assignment.gift_id}-${assignment.target_rank || 'none'}-${index}`} className="border-b border-gray-100 pb-2 last:border-0 last:pb-0 dark:border-gray-700">
                              <p className="text-sm text-gray-800 dark:text-white/90">
                                {assignment.gift.description}
                              </p>
                              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                                {formatPrizeAssignment(assignment)}
                              </p>
                              {assignment.gift.criteria && assignment.gift.criteria.length > 0 && (
                                <div className="mt-1 flex flex-wrap gap-1">
                                  {assignment.gift.criteria.map((c) => (
                                    <Badge
                                      key={c.id}
                                      color={getCriteriaColor(c.criteria_type)}
                                      size="sm"
                                    >
                                      {c.name}
                                    </Badge>
                                  ))}
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      ) : dist.matched_gifts && dist.matched_gifts.length > 0 ? (
                        <div className="space-y-2 max-w-xs">
                          {dist.matched_gifts.map((gift, index) => (
                            <div key={gift.id || index} className="border-b border-gray-100 pb-2 last:border-0 last:pb-0 dark:border-gray-700">
                              <p className="text-sm text-gray-800 dark:text-white/90">
                                {gift.description}
                              </p>
                              {gift.criteria && gift.criteria.length > 0 && (
                                <div className="mt-1 flex flex-wrap gap-1">
                                  {gift.criteria.map((c) => (
                                    <Badge
                                      key={c.id}
                                      color={getCriteriaColor(c.criteria_type)}
                                      size="sm"
                                    >
                                      {c.name}
                                    </Badge>
                                  ))}
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      ) : (
                        <span className="text-xs text-gray-500 dark:text-gray-400">
                          Нет призов
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <Badge
                        color={MATCH_REASON_COLORS[dist.match_reason]}
                        size="sm"
                      >
                        {MATCH_REASON_LABELS[dist.match_reason]}
                      </Badge>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Управление пагинацией */}
      <PaginationControls
        total={total}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        onPageSizeChange={setPageSize}
      />
    </div>
  );
}
