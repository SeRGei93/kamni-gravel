'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { participantsApi } from '@/api/participants';
import { criteriaApi } from '@/api/criteria';
import { eventsApi } from '@/api/events';
import { extractActiveEvent } from '@/utils/events';
import { type HasGiftFilter } from '@/utils/participants';
import { buildParticipantListParams } from '@/utils/participantListParams';
import {
  downloadParticipantCsv,
  isCurrentParticipantExportRequest,
  shouldSettleParticipantExportRequest,
} from '@/utils/participantCsv';
import type { Criteria, Participant } from '@/types';
import Button from '@/components/ui/button/Button';
import { DownloadIcon } from '@/icons';
import ParticipantsTable from '@/components/participants/ParticipantsTable';
import ColumnSettings from '@/components/participants/ColumnSettings';
import {
  PARTICIPANT_COLUMNS,
  PARTICIPANT_COLUMNS_STORAGE_KEY,
  TOGGLEABLE_COLUMN_KEYS,
  DEFAULT_VISIBLE_KEYS,
} from '@/components/participants/participantColumns';
import { useColumnPreferences } from '@/hooks/useColumnPreferences';
import ParticipantsFilter, {
  type ParticipantFilters,
} from '@/components/participants/ParticipantsFilter';
import PaginationControls from '@/components/tables/PaginationControls';
import { usePaginationParams } from '@/hooks/usePaginationParams';
import { useSortParams } from '@/hooks/useSortParams';
import { useFilterParams } from '@/hooks/useFilterParams';

export default function ParticipantsPage() {
  const { page, pageSize, setPage, setPageSize } = usePaginationParams();

  const [activeEventId, setActiveEventId] = useState<number | null>(null);
  const [participants, setParticipants] = useState<Participant[]>([]);
  const [criteria, setCriteria] = useState<Criteria[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isExporting, setIsExporting] = useState(false);
  const [exportError, setExportError] = useState<string | null>(null);
  const exportRequestVersionRef = useRef(0);
  const exportingRequestVersionRef = useRef<number | null>(null);
  const isMountedRef = useRef(false);

  // Фильтры хранятся в URL — переживают перезагрузку и шарятся ссылкой.
  const { gender, bikeType, isFinished, hasGift, criteriaId, q, setFilters } = useFilterParams();
  // Локальное состояние поля поиска для мгновенного ввода; в URL значение
  // коммитится дебаунсом. Инициализируем из URL при загрузке страницы.
  const [searchInput, setSearchInput] = useState(q);

  // Сортировка (server-side) хранится в URL (?sort=&order=), поэтому переживает
  // перезагрузку страницы. null — сортировки нет (порядок по умолчанию).
  const { sortKey, sortOrder, setSort } = useSortParams();

  // Настраиваемые колонки (набор сохраняется в localStorage).
  const { isVisible, toggle, reset } = useColumnPreferences(
    PARTICIPANT_COLUMNS_STORAGE_KEY,
    TOGGLEABLE_COLUMN_KEYS,
    DEFAULT_VISIBLE_KEYS,
  );
  const visibleColumns = useMemo(
    () =>
      PARTICIPANT_COLUMNS.filter(
        (column) => column.alwaysVisible || isVisible(column.key),
      ),
    [isVisible],
  );

  // Применённые фильтры для поповера «Фильтр».
  const appliedFilters = useMemo<ParticipantFilters>(
    () => ({
      gender,
      bikeType,
      isFinished,
      hasGift,
      criteriaId,
    }),
    [gender, bikeType, isFinished, hasGift, criteriaId],
  );

  const handleApplyFilters = useCallback(
    (next: ParticipantFilters) => {
      setFilters({
        gender: next.gender,
        bikeType: next.bikeType,
        isFinished: next.isFinished,
        hasGift: next.hasGift as HasGiftFilter,
        criteriaId: next.criteriaId,
      });
    },
    [setFilters],
  );

  // Дебаунс поиска: значение коммитится в URL (server-side поиск по всему списку).
  useEffect(() => {
    const timer = setTimeout(() => {
      const trimmed = searchInput.trim();
      if (trimmed !== q) {
        setFilters({ q: trimmed });
      }
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput, q, setFilters]);

  const loadActiveEvent = useCallback(async () => {
    try {
      const response = await eventsApi.getActive();
      const activeEvent = extractActiveEvent(response);
      setActiveEventId(activeEvent?.id ?? null);
      if (!activeEvent) {
        setParticipants([]);
        setTotal(0);
        setError('Нет активного события');
      }
    } catch (err) {
      setActiveEventId(null);
      setParticipants([]);
      setTotal(0);
      setError('Ошибка загрузки активного события');
      console.error('Failed to load active event:', {
        operation: 'load_active_event',
        error: err,
      });
    }
  }, []);

  const loadCriteria = useCallback(async () => {
    try {
      const response = await criteriaApi.getAll();
      setCriteria(response.criteria);
    } catch (loadError) {
      setCriteria([]);
      console.error('Failed to load participant criteria filter options:', {
        operation: 'load_participant_criteria_filter_options',
        error: loadError,
      });
    }
  }, []);

  const loadParticipants = useCallback(async () => {
    if (!activeEventId) {
      setParticipants([]);
      setTotal(0);
      return;
    }

    try {
      setIsLoading(true);
      setError(null);

      const requestParams = buildParticipantListParams({
        gender,
        bikeType,
        isFinished,
        hasGift,
        criteriaId,
        q,
        sortKey,
        sortOrder,
        page,
        pageSize,
      });
      const response = await participantsApi.listByEvent(
        activeEventId,
        requestParams,
      );
      console.debug('[participants] loaded', {
        event_id: activeEventId,
        page,
        page_size: pageSize,
        sort: requestParams.sort,
        order: requestParams.order,
        total: response.total,
      });
      setParticipants(response.participants);
      setTotal(response.total);
    } catch (err) {
      setError('Ошибка загрузки участников');
      console.error('Failed to load participants:', {
        operation: 'load_participants',
        event_id: activeEventId,
        error: err,
      });
    } finally {
      setIsLoading(false);
    }
  }, [
    gender,
    bikeType,
    isFinished,
    hasGift,
    criteriaId,
    q,
    sortKey,
    sortOrder,
    activeEventId,
    page,
    pageSize,
  ]);

  // Загрузка активного события
  useEffect(() => {
    isMountedRef.current = true;
    loadActiveEvent();
    return () => {
      isMountedRef.current = false;
    };
  }, [loadActiveEvent]);

  useEffect(() => {
    void loadCriteria();
  }, [loadCriteria]);

  // Загрузка участников при изменении фильтров/страницы
  useEffect(() => {
    loadParticipants();
  }, [loadParticipants]);

  // Тристейт-переключение сортировки по клику в шапке: asc → desc → сброс.
  const handleSortChange = useCallback(
    (key: string) => {
      if (sortKey !== key) {
        setSort(key, 'asc');
        return;
      }
      if (sortOrder === 'asc') {
        setSort(key, 'desc');
        return;
      }
      // Был desc — сбрасываем сортировку к порядку по умолчанию.
      setSort(null);
    },
    [sortKey, sortOrder, setSort],
  );

  // Скрыли активную колонку сортировки через настройки — сбрасываем сортировку,
  // чтобы не осталось «невидимой» активной сортировки без контрола в шапке.
  useEffect(() => {
    if (sortKey && !visibleColumns.some((column) => column.key === sortKey)) {
      setSort(null);
    }
  }, [visibleColumns, sortKey, setSort]);

  // Ответ экспорта больше не актуален, если пользователь сменил условия списка
  // или ушёл со страницы до его завершения.
  useEffect(() => {
    return () => {
      exportRequestVersionRef.current += 1;
    };
  }, [
    activeEventId,
    bikeType,
    criteriaId,
    gender,
    hasGift,
    isFinished,
    q,
    sortKey,
    sortOrder,
    visibleColumns,
  ]);

  const handleExport = useCallback(async () => {
    if (!activeEventId) return;

    const requestVersion = ++exportRequestVersionRef.current;
    exportingRequestVersionRef.current = requestVersion;
    const eventId = activeEventId;
    const columns = visibleColumns;
    const requestParams = buildParticipantListParams({
      gender,
      bikeType,
      isFinished,
      hasGift,
      criteriaId,
      q,
      sortKey,
      sortOrder,
      page: 1,
      pageSize: 'all',
    });

    setIsExporting(true);
    setExportError(null);
    console.debug('[participants] export started', {
      event_id: eventId,
      filter_keys: Object.entries(requestParams)
        .filter(
          ([key, value]) =>
            !['page', 'page_size', 'sort', 'order'].includes(key) &&
            value !== undefined,
        )
        .map(([key]) => key),
      sort: requestParams.sort,
      order: requestParams.order,
      column_keys: columns.map((column) => column.key),
    });

    try {
      const response = await participantsApi.listByEvent(eventId, requestParams);
      if (
        !isCurrentParticipantExportRequest(
          requestVersion,
          exportRequestVersionRef.current,
        )
      ) {
        console.debug('[participants] export invalidated', {
          event_id: eventId,
          request_version: requestVersion,
          reason: 'list_state_changed',
        });
        return;
      }

      downloadParticipantCsv({
        eventId,
        columns,
        participants: response.participants,
      });
      console.debug('[participants] export completed', {
        event_id: eventId,
        row_count: response.participants.length,
        column_keys: columns.map((column) => column.key),
      });
    } catch (exportFailure) {
      if (
        !isCurrentParticipantExportRequest(
          requestVersion,
          exportRequestVersionRef.current,
        )
      ) {
        console.debug('[participants] export invalidated', {
          event_id: eventId,
          request_version: requestVersion,
          reason: 'list_state_changed',
        });
        return;
      }

      setExportError('Не удалось выгрузить список участников');
      console.error('[participants] export failed', {
        event_id: eventId,
        error: exportFailure,
      });
    } finally {
      if (
        shouldSettleParticipantExportRequest(
          requestVersion,
          exportingRequestVersionRef.current,
          isMountedRef.current,
        )
      ) {
        exportingRequestVersionRef.current = null;
        setIsExporting(false);
        console.debug('[FIX:participants-export] export settled', {
          event_id: eventId,
          request_version: requestVersion,
        });
      } else if (exportingRequestVersionRef.current === requestVersion) {
        exportingRequestVersionRef.current = null;
        console.debug('[FIX:participants-export] state reset skipped after unmount', {
          event_id: eventId,
          request_version: requestVersion,
        });
      }
    }
  }, [
    activeEventId,
    bikeType,
    criteriaId,
    gender,
    hasGift,
    isFinished,
    q,
    sortKey,
    sortOrder,
    visibleColumns,
  ]);

  return (
    <div className="min-w-0 space-y-6">
      <div>
        <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
          Участники
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Список участников велогонки с фильтрацией
        </p>
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

      {/* Панель инструментов: поиск слева, колонки и фильтр справа */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        {/* Поиск работает как фильтр по всему списку (server-side, по всем
            страницам): запрос уходит на бэкенд, total и пагинация считаются
            по совпадениям. */}
        <div className="relative w-full sm:max-w-xs">
          <span className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2">
            <svg
              className="fill-gray-500 dark:fill-gray-400"
              width="20"
              height="20"
              viewBox="0 0 20 20"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
              aria-hidden="true"
            >
              <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M3.04175 9.37363C3.04175 5.87693 5.87711 3.04199 9.37508 3.04199C12.8731 3.04199 15.7084 5.87693 15.7084 9.37363C15.7084 12.8703 12.8731 15.7053 9.37508 15.7053C5.87711 15.7053 3.04175 12.8703 3.04175 9.37363ZM9.37508 1.54199C5.04902 1.54199 1.54175 5.04817 1.54175 9.37363C1.54175 13.6991 5.04902 17.2053 9.37508 17.2053C11.2674 17.2053 13.003 16.5344 14.357 15.4176L17.177 18.238C17.4699 18.5309 17.9448 18.5309 18.2377 18.238C18.5306 17.9451 18.5306 17.4703 18.2377 17.1774L15.418 14.3573C16.5365 13.0033 17.2084 11.2669 17.2084 9.37363C17.2084 5.04817 13.7011 1.54199 9.37508 1.54199Z"
                fill=""
              />
            </svg>
          </span>
          <input
            type="text"
            autoComplete="off"
            placeholder="Поиск по имени или username..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="h-11 w-full rounded-lg border border-gray-200 bg-transparent py-2.5 pl-11 pr-4 text-sm text-gray-800 shadow-theme-xs placeholder:text-gray-400 focus:border-brand-300 focus:outline-hidden focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:placeholder:text-white/30 dark:focus:border-brand-800"
          />
        </div>

        <div className="flex items-center gap-3">
          <ColumnSettings
            columns={PARTICIPANT_COLUMNS}
            isVisible={isVisible}
            toggle={toggle}
            reset={reset}
          />
          <ParticipantsFilter
            filters={appliedFilters}
            criteria={criteria}
            onApply={handleApplyFilters}
          />
          <Button
            size="sm"
            variant="outline"
            startIcon={<DownloadIcon />}
            onClick={() => void handleExport()}
            disabled={!activeEventId || isExporting}
          >
            {isExporting ? 'Экспорт…' : 'Экспорт в CSV'}
          </Button>
        </div>
      </div>

      {/* Таблица */}
      <ParticipantsTable
        participants={participants}
        columns={visibleColumns}
        isLoading={isLoading}
        sortKey={sortKey}
        sortOrder={sortOrder}
        onSortChange={handleSortChange}
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
