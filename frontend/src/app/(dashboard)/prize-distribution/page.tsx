'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { eventsApi } from '@/api/events';
import { prizeDistributionApi } from '@/api/prizeDistribution';
import ColumnSettings from '@/components/participants/ColumnSettings';
import PrizeDistributionFilters from '@/components/prize-distribution/PrizeDistributionFilters';
import PrizeDistributionTable from '@/components/prize-distribution/PrizeDistributionTable';
import {
  PRIZE_DISTRIBUTION_COLUMNS,
  PRIZE_DISTRIBUTION_COLUMNS_STORAGE_KEY,
  PRIZE_DISTRIBUTION_DEFAULT_VISIBLE_KEYS,
  PRIZE_DISTRIBUTION_TOGGLEABLE_COLUMN_KEYS,
} from '@/components/prize-distribution/prizeDistributionColumns';
import Badge from '@/components/ui/badge/Badge';
import PaginationControls from '@/components/tables/PaginationControls';
import { useColumnPreferences } from '@/hooks/useColumnPreferences';
import { usePaginationParams } from '@/hooks/usePaginationParams';
import type {
  BikeTypeFilter,
  GenderFilter,
  PrizeDistribution,
  PrizeDistributionStats,
  UnassignedPrizeSlot,
} from '@/types';
import { extractActiveEvent } from '@/utils/events';
import {
  formatUnassignedPrizeSlot,
  isCurrentPrizeDistributionRequest,
} from '@/utils/prizeDistribution';

export default function PrizeDistributionPage() {
  const { page, pageSize, setPage, setPageSize } = usePaginationParams();
  const requestVersionRef = useRef(0);

  const [activeEventId, setActiveEventId] = useState<number | null>(null);
  const [distribution, setDistribution] = useState<PrizeDistribution[]>([]);
  const [unassignedSlots, setUnassignedSlots] = useState<UnassignedPrizeSlot[]>([]);
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState<PrizeDistributionStats | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [genderFilter, setGenderFilter] = useState<GenderFilter>('all');
  const [bikeTypeFilter, setBikeTypeFilter] = useState<BikeTypeFilter>('all');
  const [matchReasonFilter, setMatchReasonFilter] = useState('all');

  const { isVisible, toggle, reset } = useColumnPreferences(
    PRIZE_DISTRIBUTION_COLUMNS_STORAGE_KEY,
    PRIZE_DISTRIBUTION_TOGGLEABLE_COLUMN_KEYS,
    PRIZE_DISTRIBUTION_DEFAULT_VISIBLE_KEYS
  );
  const visibleColumns = useMemo(
    () =>
      PRIZE_DISTRIBUTION_COLUMNS.filter(
        (column) => column.alwaysVisible || isVisible(column.key)
      ),
    [isVisible]
  );

  const invalidatePendingRequests = useCallback((reason: string) => {
    const requestVersion = ++requestVersionRef.current;
    console.debug('[FIX:prize-distribution] request invalidated', {
      reason,
      request_version: requestVersion,
    });
  }, []);

  const loadActiveEvent = useCallback(async () => {
    try {
      const response = await eventsApi.getActive();
      const activeEvent = extractActiveEvent(response);
      setActiveEventId(activeEvent?.id ?? null);
      if (!activeEvent) {
        invalidatePendingRequests('active_event_unavailable');
        setDistribution([]);
        setUnassignedSlots([]);
        setTotal(0);
        setStats(null);
        setIsLoading(false);
        setError('Нет активного события');
      }
    } catch (loadError) {
      invalidatePendingRequests('active_event_load_failed');
      setActiveEventId(null);
      setDistribution([]);
      setUnassignedSlots([]);
      setTotal(0);
      setStats(null);
      setIsLoading(false);
      setError('Ошибка загрузки активного события');
      console.error('[FIX:prize-distribution] failed to load active event', {
        operation: 'load_active_event',
        error: loadError,
      });
    }
  }, [invalidatePendingRequests]);

  const loadDistribution = useCallback(async () => {
    if (!activeEventId) {
      return;
    }

    const requestVersion = ++requestVersionRef.current;
    setIsLoading(true);
    setError(null);

    try {
      const response = await prizeDistributionApi.getPrizeDistribution(activeEventId, {
        gender: genderFilter,
        bike_type: bikeTypeFilter,
        match_reason: matchReasonFilter,
        page,
        page_size: pageSize,
      });
      if (
        !isCurrentPrizeDistributionRequest(
          requestVersion,
          requestVersionRef.current
        )
      ) {
        return;
      }

      console.debug('[FIX:prize-distribution] loaded', {
        event_id: activeEventId,
        gender: genderFilter,
        bike_type: bikeTypeFilter,
        match_reason: matchReasonFilter,
        page,
        page_size: pageSize,
        total: response.total,
      });
      setDistribution(response.distribution);
      setUnassignedSlots(response.unassigned_slots ?? []);
      setTotal(response.total);
      setStats(response.stats ?? null);
    } catch (loadError) {
      if (
        !isCurrentPrizeDistributionRequest(
          requestVersion,
          requestVersionRef.current
        )
      ) {
        return;
      }

      setError('Ошибка загрузки распределения призов');
      console.error('[FIX:prize-distribution] failed to load distribution', {
        operation: 'load_prize_distribution',
        event_id: activeEventId,
        gender: genderFilter,
        bike_type: bikeTypeFilter,
        match_reason: matchReasonFilter,
        page,
        page_size: pageSize,
        error: loadError,
      });
    } finally {
      if (
        isCurrentPrizeDistributionRequest(
          requestVersion,
          requestVersionRef.current
        )
      ) {
        setIsLoading(false);
      }
    }
  }, [activeEventId, bikeTypeFilter, genderFilter, matchReasonFilter, page, pageSize]);

  const applyFilterChange = useCallback(
    (change: () => void) => {
      invalidatePendingRequests('filters_changed');
      change();
      setPage(1);
    },
    [invalidatePendingRequests, setPage]
  );

  const handleGenderChange = useCallback(
    (gender: GenderFilter) => applyFilterChange(() => setGenderFilter(gender)),
    [applyFilterChange]
  );
  const handleBikeTypeChange = useCallback(
    (bikeType: BikeTypeFilter) => applyFilterChange(() => setBikeTypeFilter(bikeType)),
    [applyFilterChange]
  );
  const handleMatchReasonChange = useCallback(
    (matchReason: string) => applyFilterChange(() => setMatchReasonFilter(matchReason)),
    [applyFilterChange]
  );

  useEffect(() => {
    void loadActiveEvent();
  }, [loadActiveEvent]);

  useEffect(() => {
    if (activeEventId) {
      void loadDistribution();
      return () => invalidatePendingRequests('effect_cleanup');
    }

    invalidatePendingRequests('active_event_unavailable');
    setDistribution([]);
    setUnassignedSlots([]);
    setTotal(0);
    setStats(null);
    setIsLoading(false);
  }, [activeEventId, invalidatePendingRequests, loadDistribution]);

  const totalParticipants = stats?.total_participants ?? 0;
  const withPrizes = stats?.with_prizes ?? 0;
  const withoutPrizes = stats?.without_prizes ?? 0;
  const totalPrizeAssignments = stats?.prize_slots ?? 0;

  return (
    <div className="min-w-0 space-y-6">
      <div>
        <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
          Награждение участников
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Автоматические призы в порядке награждения. Ручные назначения сюда не входят.
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Выберите срез, затем проходите участников сверху вниз.
          </p>
          <ColumnSettings
            columns={PRIZE_DISTRIBUTION_COLUMNS}
            isVisible={isVisible}
            toggle={toggle}
            reset={reset}
          />
        </div>
        <PrizeDistributionFilters
          gender={genderFilter}
          bikeType={bikeTypeFilter}
          matchReason={matchReasonFilter}
          onGenderChange={handleGenderChange}
          onBikeTypeChange={handleBikeTypeChange}
          onMatchReasonChange={handleMatchReasonChange}
        />
      </div>

      <section aria-label="Статистика автоматического распределения">
        <p className="mb-3 text-sm text-gray-600 dark:text-gray-400">
          Статистика по всему событию, независимо от выбранного среза.
        </p>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard label="Всего участников" value={totalParticipants} />
          <StatCard label="С автоматическими призами" value={withPrizes} valueClass="text-success-600 dark:text-success-400" />
          <StatCard label="Автоматических слотов" value={totalPrizeAssignments} valueClass="text-brand-600 dark:text-brand-400" />
          <StatCard label="Без автоматического приза" value={withoutPrizes} valueClass="text-gray-600 dark:text-gray-400" />
        </div>
      </section>

      {unassignedSlots.length > 0 && (
        <div className="rounded-xl border border-warning-200 bg-warning-50 p-4 dark:border-warning-800 dark:bg-warning-900/20">
          <p className="text-sm font-semibold text-warning-700 dark:text-warning-300">
            Невыданные автоматические слоты по всему событию: {unassignedSlots.length}
          </p>
          <div className="mt-2 flex flex-wrap gap-2">
            {unassignedSlots.map((slot, index) => (
              <Badge key={`${slot.gift_id}-${slot.target_rank ?? 'none'}-${index}`} color="warning" size="sm">
                {formatUnassignedPrizeSlot(slot)}
              </Badge>
            ))}
          </div>
        </div>
      )}

      <PrizeDistributionTable
        distribution={distribution}
        columns={visibleColumns}
        isLoading={isLoading}
      />

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

function StatCard({
  label,
  value,
  valueClass = 'text-gray-800 dark:text-white',
}: {
  label: string;
  value: number;
  valueClass?: string;
}) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
      <p className="text-sm text-gray-600 dark:text-gray-400">{label}</p>
      <p className={`mt-1 text-2xl font-semibold ${valueClass}`}>{value}</p>
    </div>
  );
}
