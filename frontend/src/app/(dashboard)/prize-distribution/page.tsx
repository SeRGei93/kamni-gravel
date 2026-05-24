'use client';

import { useEffect, useState } from 'react';
import { eventsApi } from '@/api/events';
import { prizeDistributionApi } from '@/api/prizeDistribution';
import type { Event, PrizeDistribution } from '@/types';
import Select from '@/components/form/Select';
import Label from '@/components/form/Label';
import Badge from '@/components/ui/badge/Badge';
import { getCriteriaColor } from '@/utils/criteria';
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
  const [events, setEvents] = useState<Event[]>([]);
  const [selectedEventId, setSelectedEventId] = useState<number | null>(null);
  const [distribution, setDistribution] = useState<PrizeDistribution[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Фильтры
  const [matchReasonFilter, setMatchReasonFilter] = useState<string>('');

  useEffect(() => {
    loadEvents();
  }, []);

  useEffect(() => {
    if (selectedEventId) {
      loadDistribution();
    } else {
      setDistribution([]);
    }
  }, [selectedEventId]);

  const loadEvents = async () => {
    try {
      const response = await eventsApi.getAll();
      setEvents(response.events);
      // Автоматически выбираем первое активное событие
      const activeEvent = response.events.find((e) => e.active);
      if (activeEvent) {
        setSelectedEventId(activeEvent.id);
      } else if (response.events.length > 0) {
        setSelectedEventId(response.events[0].id);
      }
    } catch (err) {
      setError('Ошибка загрузки событий');
      console.error('Failed to load events:', err);
    }
  };

  const loadDistribution = async () => {
    if (!selectedEventId) return;

    try {
      setIsLoading(true);
      setError(null);
      const response = await prizeDistributionApi.getPrizeDistribution(
        selectedEventId
      );
      setDistribution(response.distribution);
    } catch (err) {
      setError('Ошибка загрузки распределения призов');
      console.error('Failed to load prize distribution:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const eventOptions = events.map((e) => ({
    value: String(e.id),
    label: e.name,
  }));

  // Фильтрация
  const filteredDistribution = distribution.filter((d) => {
    if (matchReasonFilter && d.match_reason !== matchReasonFilter) {
      return false;
    }
    return true;
  });

  // Статистика
  const withPrizes = distribution.filter((d) => d.matched_gifts && d.matched_gifts.length > 0).length;
  const withoutPrizes = distribution.length - withPrizes;

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
            <Label>Событие</Label>
            <Select
              options={eventOptions}
              placeholder="Выберите событие"
              defaultValue={selectedEventId ? String(selectedEventId) : ''}
              onChange={(value) =>
                setSelectedEventId(value ? Number(value) : null)
              }
            />
          </div>

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
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]">
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Всего участников
          </p>
          <p className="mt-1 text-2xl font-semibold text-gray-800 dark:text-white">
            {distribution.length}
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
            Без призов
          </p>
          <p className="mt-1 text-2xl font-semibold text-gray-600 dark:text-gray-400">
            {withoutPrizes}
          </p>
        </div>
      </div>

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
                  Критерии
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Подарок
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400">
                  Тип совпадения
                </th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr>
                  <td colSpan={8} className="px-4 py-8 text-center">
                    <div className="text-gray-500 dark:text-gray-400">
                      Загрузка...
                    </div>
                  </td>
                </tr>
              ) : filteredDistribution.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-4 py-8 text-center">
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
                      {dist.matched_gifts && dist.matched_gifts.length > 0 ? (
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
                          Нет подарков
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
    </div>
  );
}
