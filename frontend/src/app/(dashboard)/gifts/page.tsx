'use client';

import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { giftsApi } from '@/api/gifts';
import { eventsApi } from '@/api/events';
import { participantsApi } from '@/api/participants';
import { prizeDistributionApi } from '@/api/prizeDistribution';
import { extractActiveEvent } from '@/utils/events';
import type {
  BikeTypeFilter,
  Gift,
  GenderFilter,
  GiftReviewStatus,
  Participant,
  PrizeDistributionListResponse,
} from '@/types';
import GiftsTable from '@/components/gifts/GiftsTable';
import PaginationControls from '@/components/tables/PaginationControls';
import { usePaginationParams } from '@/hooks/usePaginationParams';
import Button from '@/components/ui/button/Button';
import Input from '@/components/form/input/InputField';
import Select from '@/components/form/Select';
import Label from '@/components/form/Label';
import TextArea from '@/components/form/input/TextArea';
import GiftOwnerFilter from '@/components/gifts/GiftOwnerFilter';
import { BIKE_TYPE_OPTIONS, GENDER_OPTIONS, GIFT_REVIEW_STATUS_FILTER_OPTIONS } from '@/constants';
import { CheckLineIcon, CloseLineIcon, DownloadIcon, PlusIcon } from '@/icons';
import { getManualGiftErrorMessage } from '@/utils/manualGiftErrors';
import {
  buildManualGiftUpdate,
  type GiftDistributionFilter,
} from '@/utils/manualGiftAssignment';
import {
  buildFilteredGiftList,
  isCurrentGiftListRequest,
  paginateGifts,
  shouldSettleGiftListRequest,
} from '@/utils/giftList';
import {
  downloadGiftCsv,
  isCurrentGiftExportRequest,
  shouldSettleGiftExportRequest,
} from '@/utils/giftCsv';

type GiftReviewStatusFilter = 'all' | GiftReviewStatus;

const GIFT_DISTRIBUTION_FILTER_OPTIONS = [
  { value: 'all', label: 'Все' },
  { value: 'assigned', label: 'Распределён' },
  { value: 'unassigned', label: 'Не распределён' },
  { value: 'manual', label: 'Ручное распределение' },
  { value: 'manual_unassigned', label: 'Ручное распределение — получатель не назначен' },
];

function parseReviewStatusFilter(value: string | null): GiftReviewStatusFilter {
  return value === 'pending_review' || value === 'approved' ? value : 'all';
}

function parseDistributionFilter(value: string | null): GiftDistributionFilter {
  return value === 'assigned' ||
    value === 'unassigned' ||
    value === 'manual' ||
    value === 'manual_unassigned'
    ? value
    : 'all';
}

function parseOwnerUserID(value: string | null): number | undefined {
  const ownerUserID = Number(value);
  return Number.isSafeInteger(ownerUserID) && ownerUserID > 0
    ? ownerUserID
    : undefined;
}

function collectAssignedGiftIds(
  distribution: PrizeDistributionListResponse
): Set<number> {
  const assignedGiftIds = new Set<number>();

  distribution.distribution.forEach((participantDistribution) => {
    participantDistribution.matched_gift_assignments?.forEach((assignment) => {
      assignedGiftIds.add(assignment.gift_id);
    });
    participantDistribution.matched_gifts?.forEach((gift) => {
      assignedGiftIds.add(gift.id);
    });
  });

  return assignedGiftIds;
}

export default function GiftsPage() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const reviewStatusParam = searchParams.get('review_status');
  const searchQueryParam = searchParams.get('q') ?? '';
  const ownerUserIDParam = searchParams.get('owner_user_id');
  const distributionParam = searchParams.get('distribution');

  const { page, pageSize, setPage, setPageSize } = usePaginationParams();

  const [activeEventId, setActiveEventId] = useState<number | null>(null);
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [total, setTotal] = useState(0);
  const [statusCounts, setStatusCounts] = useState<Record<string, number>>({});
  const [giftOwners, setGiftOwners] = useState<Participant[]>([]);
  const [reviewStatusFilter, setReviewStatusFilter] =
    useState<GiftReviewStatusFilter>(
      parseReviewStatusFilter(reviewStatusParam)
    );
  const [assignedGiftIds, setAssignedGiftIds] = useState<Set<number>>(new Set());
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isExporting, setIsExporting] = useState(false);
  const [exportError, setExportError] = useState<string | null>(null);
  const [isCreatingGift, setIsCreatingGift] = useState(false);
  const [isSavingGift, setIsSavingGift] = useState(false);
  const [manualGiftUserId, setManualGiftUserId] = useState('');
  const [manualGiftDescription, setManualGiftDescription] = useState('');
  const [manualGiftGenderFilter, setManualGiftGenderFilter] =
    useState<GenderFilter>('all');
  const [manualGiftBikeTypeFilter, setManualGiftBikeTypeFilter] =
    useState<BikeTypeFilter>('all');
  const [searchInput, setSearchInput] = useState(searchQueryParam);
  const giftListRequestVersionRef = useRef(0);
  const exportRequestVersionRef = useRef(0);
  const exportingRequestVersionRef = useRef<number | null>(null);
  const isMountedRef = useRef(false);
  const ownerUserIDFilter = parseOwnerUserID(ownerUserIDParam);
  const distributionFilter = parseDistributionFilter(distributionParam);

  const listQueryString = useMemo(() => {
    const params = new URLSearchParams();

    if (reviewStatusFilter !== 'all') {
      params.set('review_status', reviewStatusFilter);
    }
    if (searchQueryParam) {
      params.set('q', searchQueryParam);
    }
    if (ownerUserIDFilter) {
      params.set('owner_user_id', String(ownerUserIDFilter));
    }
    if (distributionFilter !== 'all') {
      params.set('distribution', distributionFilter);
    }

    return params.toString();
  }, [distributionFilter, ownerUserIDFilter, reviewStatusFilter, searchQueryParam]);

  const updateListQuery = useCallback(
    (next: {
      reviewStatus?: GiftReviewStatusFilter;
      searchQuery?: string;
      ownerUserID?: number | undefined;
      distribution?: GiftDistributionFilter;
    }) => {
      const params = new URLSearchParams(searchParams.toString());

      if ('reviewStatus' in next) {
        if (next.reviewStatus && next.reviewStatus !== 'all') {
          params.set('review_status', next.reviewStatus);
        } else {
          params.delete('review_status');
        }
        // Смена фильтра сбрасывает на первую страницу.
        params.delete('page');
      }
      if ('searchQuery' in next) {
        if (next.searchQuery) {
          params.set('q', next.searchQuery);
        } else {
          params.delete('q');
        }
        params.delete('page');
      }
      if ('ownerUserID' in next) {
        if (next.ownerUserID) {
          params.set('owner_user_id', String(next.ownerUserID));
        } else {
          params.delete('owner_user_id');
        }
        params.delete('page');
      }
      if ('distribution' in next) {
        if (next.distribution && next.distribution !== 'all') {
          params.set('distribution', next.distribution);
        } else {
          params.delete('distribution');
        }
        params.delete('page');
      }

      const query = params.toString();
      router.replace(`${pathname}${query ? `?${query}` : ''}`, {
        scroll: false,
      });
    },
    [pathname, router, searchParams]
  );

  const loadActiveEvent = useCallback(async () => {
    try {
      const response = await eventsApi.getActive();
      const activeEvent = extractActiveEvent(response);
      setActiveEventId(activeEvent?.id ?? null);
      if (!activeEvent) {
        setGifts([]);
        setTotal(0);
        setStatusCounts({});
        setAssignedGiftIds(new Set());
        setError('Нет активного события');
      }
    } catch (err) {
      setActiveEventId(null);
      setError('Ошибка загрузки активного события');
      console.error('Failed to load active event:', {
        operation: 'load_active_event',
        error: err,
      });
    }
  }, []);

  // Загрузка активного события
  useEffect(() => {
    isMountedRef.current = true;
    void loadActiveEvent();

    return () => {
      isMountedRef.current = false;
      giftListRequestVersionRef.current += 1;
    };
  }, [loadActiveEvent]);

  const loadGiftOwners = useCallback(async () => {
    if (!activeEventId) {
      setGiftOwners([]);
      return;
    }

    try {
      const response = await participantsApi.getByEvent(activeEventId);
      setGiftOwners(response.participants.filter((participant) => participant.has_gift));
    } catch (loadError) {
      setGiftOwners([]);
      console.error('Failed to load gift owner filter options:', {
        operation: 'load_gift_owners',
        event_id: activeEventId,
        error: loadError,
      });
    }
  }, [activeEventId]);

  useEffect(() => {
    void loadGiftOwners();
  }, [loadGiftOwners]);

  useEffect(() => {
    const nextReviewStatus = parseReviewStatusFilter(reviewStatusParam);
    setReviewStatusFilter((current) =>
      current === nextReviewStatus ? current : nextReviewStatus
    );
  }, [reviewStatusParam]);

  useEffect(() => {
    setSearchInput(searchQueryParam);
  }, [searchQueryParam]);

  useEffect(() => {
    const timer = setTimeout(() => {
      const nextSearchQuery = searchInput.trim();
      if (nextSearchQuery !== searchQueryParam) {
        updateListQuery({ searchQuery: nextSearchQuery });
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput, searchQueryParam, updateListQuery]);

  const loadGifts = useCallback(async () => {
    if (!activeEventId) return;

    const requestVersion = ++giftListRequestVersionRef.current;
    const eventId = activeEventId;

    try {
      setIsLoading(true);
      setError(null);

      const filters = {
        review_status:
          reviewStatusFilter === 'all' ? undefined : reviewStatusFilter,
        owner_user_id: ownerUserIDFilter,
        q: searchQueryParam || undefined,
      };
      const shouldFilterDistribution = distributionFilter !== 'all';

      console.debug('[gifts] list load started', {
        event_id: eventId,
        request_version: requestVersion,
        page,
        page_size: pageSize,
        distribution: distributionFilter,
        review_status: reviewStatusFilter,
        owner_user_id: ownerUserIDFilter,
        has_search_query: Boolean(searchQueryParam),
      });

      // Для фильтра распределения нужен полный отфильтрованный набор: статус
      // автоматического приза вычисляет серверный движок, а ручного — сохранённый
      // получатель. После этого применяем обычную пагинацию на клиенте.
      const [response, manualGiftResponse] = await Promise.all([
        shouldFilterDistribution
          ? giftsApi.getByEvent(eventId, filters)
          : giftsApi.listByEvent({
              eventId,
              ...filters,
              page,
              page_size: pageSize,
            }),
        giftsApi.getManualByEvent(eventId),
      ]);

      if (!isCurrentGiftListRequest(requestVersion, giftListRequestVersionRef.current)) {
        console.debug('[gifts] list response invalidated', {
          event_id: eventId,
          request_version: requestVersion,
          reason: 'newer_list_request',
        });
        return;
      }

      let assignedIds = new Set<number>();
      try {
        const distribution = await prizeDistributionApi.getPrizeDistribution(eventId);
        assignedIds = collectAssignedGiftIds(distribution);
      } catch (distributionError) {
        console.error('Failed to load prize distribution:', {
          event_id: eventId,
          operation: 'load_prize_distribution',
          error: distributionError,
        });
        if (shouldFilterDistribution) {
          throw distributionError;
        }
      }

      if (!isCurrentGiftListRequest(requestVersion, giftListRequestVersionRef.current)) {
        console.debug('[gifts] list response invalidated', {
          event_id: eventId,
          request_version: requestVersion,
          reason: 'newer_distribution_request',
        });
        return;
      }

      const filteredGifts = buildFilteredGiftList({
        gifts: response.gifts,
        manualGifts: manualGiftResponse.gifts,
        distributionFilter,
        assignedGiftIds: assignedIds,
      });

      setAssignedGiftIds(assignedIds);
      if (shouldFilterDistribution) {
        setGifts(paginateGifts(filteredGifts, page, pageSize));
        setTotal(filteredGifts.length);
      } else {
        setGifts(filteredGifts);
        setTotal(response.total);
      }

      setStatusCounts(response.status_counts ?? {});
      console.debug('[gifts] list load completed', {
        event_id: eventId,
        request_version: requestVersion,
        distribution: distributionFilter,
        base_count: response.gifts.length,
        filtered_count: filteredGifts.length,
        total: shouldFilterDistribution ? filteredGifts.length : response.total,
      });
    } catch (err) {
      if (!isCurrentGiftListRequest(requestVersion, giftListRequestVersionRef.current)) {
        console.debug('[gifts] list response invalidated', {
          event_id: eventId,
          request_version: requestVersion,
          reason: 'newer_list_request_after_error',
        });
        return;
      }

      setError('Ошибка загрузки призов');
      console.error('[gifts] list load failed', {
        event_id: eventId,
        request_version: requestVersion,
        review_status: reviewStatusFilter,
        owner_user_id: ownerUserIDFilter,
        distribution: distributionFilter,
        has_search_query: Boolean(searchQueryParam),
        operation: 'load_gifts',
        error: err,
      });
    } finally {
      if (
        shouldSettleGiftListRequest(
          requestVersion,
          giftListRequestVersionRef.current,
          isMountedRef.current
        )
      ) {
        setIsLoading(false);
      }
    }
  }, [
    activeEventId,
    distributionFilter,
    ownerUserIDFilter,
    page,
    pageSize,
    reviewStatusFilter,
    searchQueryParam,
  ]);

  // Загрузка призов при изменении фильтров/страницы
  useEffect(() => {
    if (activeEventId) {
      void loadGifts();
    } else {
      setGifts([]);
      setTotal(0);
      setStatusCounts({});
      setGiftOwners([]);
      setAssignedGiftIds(new Set());
    }

    return () => {
      giftListRequestVersionRef.current += 1;
    };
  }, [activeEventId, loadGifts]);

  // Экспорт строится по полному набору, поэтому результат устаревает при
  // изменении любого URL-фильтра или активного события, но не при пагинации.
  useEffect(() => {
    return () => {
      exportRequestVersionRef.current += 1;

      if (exportingRequestVersionRef.current !== null) {
        exportingRequestVersionRef.current = null;
        queueMicrotask(() => {
          if (isMountedRef.current) {
            setIsExporting(false);
          }
        });
      }
    };
  }, [
    activeEventId,
    distributionFilter,
    ownerUserIDFilter,
    reviewStatusFilter,
    searchQueryParam,
  ]);

  const handleExport = useCallback(async () => {
    if (!activeEventId) return;

    const requestVersion = ++exportRequestVersionRef.current;
    exportingRequestVersionRef.current = requestVersion;
    const eventId = activeEventId;
    const filters = {
      review_status:
        reviewStatusFilter === 'all' ? undefined : reviewStatusFilter,
      owner_user_id: ownerUserIDFilter,
      q: searchQueryParam || undefined,
    };
    const filterKeys = Object.entries({
      ...filters,
      distribution:
        distributionFilter === 'all' ? undefined : distributionFilter,
    })
      .filter(([, value]) => value !== undefined)
      .map(([key]) => key);

    setIsExporting(true);
    setExportError(null);
    console.debug('[gifts] export started', {
      event_id: eventId,
      request_version: requestVersion,
      filter_keys: filterKeys,
    });

    try {
      const [giftResponse, manualGiftResponse, distribution] = await Promise.all([
        giftsApi.getByEvent(eventId, filters),
        giftsApi.getManualByEvent(eventId),
        prizeDistributionApi.getPrizeDistribution(eventId),
      ]);

      if (
        !isMountedRef.current ||
        !isCurrentGiftExportRequest(
          requestVersion,
          exportRequestVersionRef.current
        )
      ) {
        console.debug('[gifts] export invalidated', {
          event_id: eventId,
          request_version: requestVersion,
          reason: 'list_state_changed',
        });
        return;
      }

      const assignedGiftIds = collectAssignedGiftIds(distribution);
      const exportGifts = buildFilteredGiftList({
        gifts: giftResponse.gifts,
        manualGifts: manualGiftResponse.gifts,
        distributionFilter,
        assignedGiftIds,
      });

      downloadGiftCsv({
        eventId,
        gifts: exportGifts,
        assignedGiftIds,
      });
      console.debug('[gifts] export completed', {
        event_id: eventId,
        request_version: requestVersion,
        row_count: exportGifts.length,
        filter_keys: filterKeys,
      });
    } catch (exportFailure) {
      if (
        !isMountedRef.current ||
        !isCurrentGiftExportRequest(
          requestVersion,
          exportRequestVersionRef.current
        )
      ) {
        console.debug('[gifts] export invalidated', {
          event_id: eventId,
          request_version: requestVersion,
          reason: 'list_state_changed_after_error',
        });
        return;
      }

      setExportError('Не удалось выгрузить список призов');
      console.error('[gifts] export failed', {
        event_id: eventId,
        request_version: requestVersion,
        filter_keys: filterKeys,
        error: exportFailure,
      });
    } finally {
      if (
        shouldSettleGiftExportRequest(
          requestVersion,
          exportRequestVersionRef.current,
          isMountedRef.current
        )
      ) {
        exportingRequestVersionRef.current = null;
        setIsExporting(false);
        console.debug('[gifts] export settled', {
          event_id: eventId,
          request_version: requestVersion,
        });
      } else if (exportingRequestVersionRef.current === requestVersion) {
        exportingRequestVersionRef.current = null;
        console.debug('[gifts] export settled', {
          event_id: eventId,
          request_version: requestVersion,
          state_update: 'skipped_after_unmount_or_invalidation',
        });
      }
    }
  }, [
    activeEventId,
    distributionFilter,
    ownerUserIDFilter,
    reviewStatusFilter,
    searchQueryParam,
  ]);

  const handleApprove = async (gift: Gift) => {
    try {
      await giftsApi.update(gift.id, {
        description: gift.description,
        gender_filter: gift.gender_filter || 'all',
        bike_type_filter: gift.bike_type_filter || 'all',
        review_status: 'approved',
        place_rule:
          gift.place_rule ??
          (gift.place ? { type: 'places', places: [gift.place] } : null),
        criteria_ids: gift.criteria?.map((criteria) => criteria.id) || [],
        manual_distribution: gift.manual_distribution ?? false,
        manual_recipient_participant_id:
          gift.manual_assignment?.recipient?.id ?? null,
      });
      await loadGifts();
    } catch (err) {
      setError('Ошибка проверки приза');
      console.error('Failed to approve gift:', {
        gift_id: gift.id,
        event_id: activeEventId,
        operation: 'approve_gift',
        error: err,
      });
      throw err;
    }
  };

  const handleEnableManualAssignment = async (gift: Gift) => {
    try {
      setError(null);
      await giftsApi.update(gift.id, buildManualGiftUpdate(true, null));
      await loadGifts();
    } catch (err) {
      setError('Не удалось включить ручное назначение');
      console.error('Failed to enable manual gift assignment:', {
        gift_id: gift.id,
        event_id: activeEventId,
        operation: 'enable_manual_gift_assignment',
        error: err,
      });
      throw err;
    }
  };

  const handleAssignRandomRecipient = async (gift: Gift) => {
    try {
      setError(null);
      await giftsApi.assignRandomRecipient(gift.id);
      await loadGifts();
      console.info('Random gift recipient assigned:', {
        operation: 'assign_random_gift_recipient',
        gift_id: gift.id,
        event_id: activeEventId,
      });
    } catch (err) {
      setError('Не удалось случайно распределить приз. Обновите список и повторите.');
      console.error('Failed to assign random gift recipient:', {
        operation: 'assign_random_gift_recipient',
        gift_id: gift.id,
        event_id: activeEventId,
        error: err,
      });
      throw err;
    }
  };

  const resetManualGiftForm = () => {
    setManualGiftUserId('');
    setManualGiftDescription('');
    setManualGiftGenderFilter('all');
    setManualGiftBikeTypeFilter('all');
  };

  const handleManualGiftSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!activeEventId) {
      setError('Активное событие не найдено. Обновите страницу.');
      return;
    }

    const telegramUserId = Number(manualGiftUserId);
    if (!Number.isInteger(telegramUserId) || telegramUserId <= 0) {
      setError('Введите корректный Telegram ID');
      return;
    }

    const description = manualGiftDescription.trim();
    if (!description) {
      setError('Введите описание приза');
      return;
    }

    try {
      setIsSavingGift(true);
      setError(null);
      await giftsApi.create(activeEventId, {
        user_id: telegramUserId,
        description,
        gender_filter: manualGiftGenderFilter,
        bike_type_filter: manualGiftBikeTypeFilter,
      });
      resetManualGiftForm();
      setIsCreatingGift(false);

      if (reviewStatusFilter === 'approved') {
        setReviewStatusFilter('pending_review');
        updateListQuery({ reviewStatus: 'pending_review' });
      } else {
        await loadGifts();
      }
    } catch (err) {
      setError(getManualGiftErrorMessage(err, telegramUserId));
      console.error('[FIX:manual-gift-error] Failed to create gift:', {
        operation: 'create_manual_gift',
        event_id: activeEventId,
        telegram_user_id: telegramUserId,
        error: err,
      });
    } finally {
      setIsSavingGift(false);
    }
  };

  // Счётчики по статусам приходят с сервера (по всему событию, не по странице).
  const totalGiftsCount = statusCounts.all ?? 0;
  const totalPendingReviewCount = statusCounts.pending_review ?? 0;
  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
            Призы
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Призы проходят проверку администратора перед распределением
          </p>
        </div>
        {!isCreatingGift && (
          <div className="flex items-center gap-3">
            <Button
              size="sm"
              variant="outline"
              startIcon={<DownloadIcon />}
              onClick={() => void handleExport()}
              disabled={!activeEventId || isExporting}
            >
              {isExporting ? 'Экспорт…' : 'Экспорт в CSV'}
            </Button>
            <Button
              size="sm"
              startIcon={<PlusIcon />}
              onClick={() => setIsCreatingGift(true)}
              disabled={!activeEventId}
            >
              Добавить приз вручную
            </Button>
          </div>
        )}
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {exportError && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{exportError}</p>
        </div>
      )}

      {activeEventId && totalPendingReviewCount > 0 && (
        <div className="rounded-lg border border-warning-200 bg-warning-50 p-4 dark:border-warning-800 dark:bg-warning-900/20">
          <p className="text-sm font-medium text-warning-700 dark:text-orange-300">
            На проверке {totalPendingReviewCount} призов. До проверки они не участвуют в распределении.
          </p>
        </div>
      )}

      {isCreatingGift && (
        <form
          onSubmit={handleManualGiftSubmit}
          className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]"
        >
          <div className="mb-4 flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold text-gray-800 dark:text-white">
              Добавить приз вручную
            </h2>
            <div className="flex gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                startIcon={<CloseLineIcon />}
                onClick={() => {
                  resetManualGiftForm();
                  setIsCreatingGift(false);
                }}
                disabled={isSavingGift}
              >
                Отмена
              </Button>
              <Button
                type="submit"
                size="sm"
                startIcon={<CheckLineIcon />}
                disabled={isSavingGift || !activeEventId}
              >
                {isSavingGift ? 'Сохранение...' : 'Сохранить'}
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-4">
            <div>
              <Label>Telegram ID</Label>
              <Input
                type="number"
                min="1"
                placeholder="7045033621"
                value={manualGiftUserId}
                onChange={(event) => setManualGiftUserId(event.target.value)}
                required
              />
            </div>
            <div>
              <Label>Пол</Label>
              <Select
                options={GENDER_OPTIONS}
                key={`manual-gift-gender-${manualGiftGenderFilter}`}
                defaultValue={manualGiftGenderFilter}
                onChange={(value) => setManualGiftGenderFilter(value as GenderFilter)}
                required
              />
            </div>
            <div>
              <Label>Тип велосипеда</Label>
              <Select
                options={BIKE_TYPE_OPTIONS}
                key={`manual-gift-bike-${manualGiftBikeTypeFilter}`}
                defaultValue={manualGiftBikeTypeFilter}
                onChange={(value) => setManualGiftBikeTypeFilter(value as BikeTypeFilter)}
                required
              />
            </div>
            <div className="lg:col-span-4">
              <Label>Описание приза</Label>
              <TextArea
                placeholder="Введите описание приза"
                value={manualGiftDescription}
                onChange={setManualGiftDescription}
                rows={3}
                required
              />
            </div>
          </div>
        </form>
      )}

      {/* Фильтры */}
      <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <div>
            <Label>Поиск</Label>
            <Input
              type="search"
              autoComplete="off"
              placeholder="Описание, имя или @username"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
            />
          </div>
          <div>
            <Label>Автор приза</Label>
            <GiftOwnerFilter
              owners={giftOwners}
              value={ownerUserIDFilter}
              onChange={(ownerUserID) => {
                updateListQuery({ ownerUserID });
              }}
            />
          </div>
          <div>
            <Label>Распределение</Label>
            <Select
              options={GIFT_DISTRIBUTION_FILTER_OPTIONS}
              key={`gift-distribution-${distributionFilter}`}
              defaultValue={distributionFilter}
              onChange={(value) => {
                updateListQuery({ distribution: parseDistributionFilter(value) });
              }}
            />
          </div>
          <div>
            <Label>Статус проверки</Label>
            <Select
              options={GIFT_REVIEW_STATUS_FILTER_OPTIONS}
              key={`review-status-${reviewStatusFilter}`}
              defaultValue={reviewStatusFilter}
              onChange={(value) => {
                const nextReviewStatus = parseReviewStatusFilter(value);
                setReviewStatusFilter(nextReviewStatus);
                updateListQuery({ reviewStatus: nextReviewStatus });
              }}
            />
          </div>
        </div>
      </div>

      {/* Информация о количестве */}
      <p className="text-sm text-gray-600 dark:text-gray-400">
        Всего призов: {totalGiftsCount} · Найдено: {total} · На проверке: {totalPendingReviewCount}
      </p>

      {/* Таблица */}
      <GiftsTable
        gifts={gifts}
        assignedGiftIds={assignedGiftIds}
        isLoading={isLoading}
        onApprove={handleApprove}
        onEnableManualAssignment={handleEnableManualAssignment}
        onAssignRandomRecipient={handleAssignRandomRecipient}
        editQueryString={listQueryString}
      />

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
