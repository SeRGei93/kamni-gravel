'use client';

import { useEffect, useState } from 'react';
import { statsApi } from '@/api/stats';
import { eventsApi } from '@/api/events';
import { extractActiveEvent } from '@/utils/events';
import type { Stats, EventDailyStats } from '@/types';
import StatCard from '@/components/dashboard/StatCard';
import BreakdownCard from '@/components/dashboard/BreakdownCard';
import EventDailyChart from '@/components/charts/EventDailyChart';
import { GroupIcon, CheckCircleIcon, BoxIcon, ShootingStarIcon } from '@/icons';

// formatDay превращает "YYYY-MM-DD" в короткую подпись "dd.MM" (без date-библиотек).
function formatDay(iso: string): string {
  const d = new Date(`${iso}T00:00:00`);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit' });
}

const GENDER_LABELS: Record<string, string> = {
  male: 'Мужчины',
  female: 'Женщины',
};

const BIKE_TYPE_LABELS: Record<string, string> = {
  gravel: 'Гравийник',
  mtb: 'МТБ',
  road: 'Шоссе',
  single_speed: 'Фикс',
  tandem: 'Тандем',
};

export default function DashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [dailyStats, setDailyStats] = useState<EventDailyStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadDashboard();
  }, []);

  // Грузим статистику и посуточные данные только по активному событию.
  const loadDashboard = async () => {
    try {
      setIsLoading(true);
      setError(null);

      const eventsResponse = await eventsApi.getActive();
      const activeEvent = extractActiveEvent(eventsResponse);
      if (!activeEvent) {
        setStats(null);
        setDailyStats(null);
        return;
      }

      const [eventStats, daily] = await Promise.all([
        statsApi.getByEvent(activeEvent.id),
        statsApi.getDailyByEvent(activeEvent.id),
      ]);
      setStats(eventStats);
      setDailyStats(daily);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка загрузки статистики');
      console.error('Failed to load dashboard:', err);
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-gray-500 dark:text-gray-400">Загрузка статистики...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
        <p className="text-error-600 dark:text-error-400">{error}</p>
        <button
          onClick={loadDashboard}
          className="mt-2 text-sm text-error-600 underline dark:text-error-400"
        >
          Попробовать снова
        </button>
      </div>
    );
  }

  if (!stats) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
            Dashboard
          </h1>
          <p className="text-gray-600 dark:text-gray-400">Нет активного события</p>
        </div>
      </div>
    );
  }

  const genderTotal = (stats.by_gender.male || 0) + (stats.by_gender.female || 0);
  const bikeTypeTotal =
    (stats.by_bike_type.gravel || 0) +
    (stats.by_bike_type.mtb || 0) +
    (stats.by_bike_type.road || 0) +
    (stats.by_bike_type.single_speed || 0) +
    (stats.by_bike_type.tandem || 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
          Dashboard
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Активное событие: {stats.event_name}
        </p>
      </div>

      {/* Карточки статистики активного события */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:gap-6 lg:grid-cols-4">
        <StatCard
          title="Участников"
          value={stats.participants_count}
          icon={<GroupIcon className="size-6" />}
          color="primary"
        />
        <StatCard
          title="Проехали дистанцию"
          value={stats.finished_count}
          icon={<CheckCircleIcon className="size-6" />}
          color="success"
          trend={{
            value:
              stats.participants_count > 0
                ? (stats.finished_count / stats.participants_count) * 100
                : 0,
            isPositive: true,
          }}
        />
        <StatCard
          title="Призов в фонде"
          value={stats.gifts_count}
          icon={<BoxIcon className="size-6" />}
          color="info"
        />
        <StatCard
          title="Призов распределено"
          value={stats.prizes_assigned_count}
          icon={<ShootingStarIcon className="size-6" />}
          color="warning"
        />
      </div>

      {/* Посуточные графики активного события — на всю ширину */}
      {dailyStats && (
        <div className="space-y-4 md:space-y-6">
          <EventDailyChart
            title="Проехавшие по дням"
            categories={dailyStats.finishes.map((p) => formatDay(p.date))}
            data={dailyStats.finishes.map((p) => p.count)}
            color="#12b76a"
          />
          <EventDailyChart
            title="Новые участники по дням"
            categories={dailyStats.registrations.map((p) => formatDay(p.date))}
            data={dailyStats.registrations.map((p) => p.count)}
            color="#465fff"
          />
        </div>
      )}

      {/* Разбивка по зачётам активного события */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:gap-6">
        {genderTotal > 0 && (
          <BreakdownCard
            title="Разбивка по полу"
            data={stats.by_gender}
            total={genderTotal}
            labels={GENDER_LABELS}
          />
        )}
        {bikeTypeTotal > 0 && (
          <BreakdownCard
            title="Разбивка по типу велосипеда"
            data={stats.by_bike_type}
            total={bikeTypeTotal}
            labels={BIKE_TYPE_LABELS}
          />
        )}
      </div>
    </div>
  );
}
