'use client';

import { useEffect, useState } from 'react';
import { statsApi } from '@/api/stats';
import type { Stats } from '@/types';
import StatCard from '@/components/dashboard/StatCard';
import BreakdownCard from '@/components/dashboard/BreakdownCard';
import { GroupIcon, CheckCircleIcon, BoxIcon, ShootingStarIcon } from '@/icons';

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
  const [stats, setStats] = useState<Stats[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await statsApi.getAll();
      setStats(response.stats);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка загрузки статистики');
      console.error('Failed to load stats:', err);
    } finally {
      setIsLoading(false);
    }
  };

  // Агрегируем статистику по всем событиям
  const totalStats = stats.reduce(
    (acc, stat) => ({
      participants_count: acc.participants_count + stat.participants_count,
      finished_count: acc.finished_count + stat.finished_count,
      gifts_count: acc.gifts_count + stat.gifts_count,
      prizes_assigned_count: acc.prizes_assigned_count + stat.prizes_assigned_count,
      by_gender: {
        male: (acc.by_gender.male || 0) + (stat.by_gender.male || 0),
        female: (acc.by_gender.female || 0) + (stat.by_gender.female || 0),
      },
      by_bike_type: {
        gravel: (acc.by_bike_type.gravel || 0) + (stat.by_bike_type.gravel || 0),
        mtb: (acc.by_bike_type.mtb || 0) + (stat.by_bike_type.mtb || 0),
        road: (acc.by_bike_type.road || 0) + (stat.by_bike_type.road || 0),
        single_speed: (acc.by_bike_type.single_speed || 0) + (stat.by_bike_type.single_speed || 0),
        tandem: (acc.by_bike_type.tandem || 0) + (stat.by_bike_type.tandem || 0),
      },
    }),
    {
      participants_count: 0,
      finished_count: 0,
      gifts_count: 0,
      prizes_assigned_count: 0,
      by_gender: {} as Record<string, number>,
      by_bike_type: {} as Record<string, number>,
    }
  );

  const totalParticipants = totalStats.participants_count;
  const totalGender = totalStats.by_gender.male + totalStats.by_gender.female;
  const totalBikeType =
    (totalStats.by_bike_type.gravel || 0) +
    (totalStats.by_bike_type.mtb || 0) +
    (totalStats.by_bike_type.road || 0) +
    (totalStats.by_bike_type.single_speed || 0) +
    (totalStats.by_bike_type.tandem || 0);

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
          onClick={loadStats}
          className="mt-2 text-sm text-error-600 underline dark:text-error-400"
        >
          Попробовать снова
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
          Dashboard
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Общая статистика по всем событиям
        </p>
      </div>

      {/* Карточки статистики */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4 md:gap-6">
        <StatCard
          title="Участников"
          value={totalStats.participants_count}
          icon={<GroupIcon className="size-6" />}
          color="primary"
        />
        <StatCard
          title="Проехали дистанцию"
          value={totalStats.finished_count}
          icon={<CheckCircleIcon className="size-6" />}
          color="success"
          trend={{
            value:
              totalStats.participants_count > 0
                ? (totalStats.finished_count / totalStats.participants_count) * 100
                : 0,
            isPositive: true,
          }}
        />
        <StatCard
          title="Подарков в фонде"
          value={totalStats.gifts_count}
          icon={<BoxIcon className="size-6" />}
          color="info"
        />
        <StatCard
          title="Призов распределено"
          value={totalStats.prizes_assigned_count}
          icon={<ShootingStarIcon className="size-6" />}
          color="warning"
        />
      </div>

      {/* Разбивка по зачётам */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 md:gap-6">
        {totalGender > 0 && (
          <BreakdownCard
            title="Разбивка по полу"
            data={totalStats.by_gender}
            total={totalGender}
            labels={GENDER_LABELS}
          />
        )}
        {totalBikeType > 0 && (
          <BreakdownCard
            title="Разбивка по типу велосипеда"
            data={totalStats.by_bike_type}
            total={totalBikeType}
            labels={BIKE_TYPE_LABELS}
          />
        )}
      </div>

      {/* Статистика по событиям */}
      {stats.length > 0 && (
        <div>
          <h2 className="mb-4 text-xl font-semibold text-gray-800 dark:text-white">
            Статистика по событиям
          </h2>
          <div className="space-y-4">
            {stats.map((stat) => (
              <div
                key={stat.event_id}
                className="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]"
              >
                <h3 className="mb-3 font-semibold text-gray-800 dark:text-white">
                  {stat.event_name}
                </h3>
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                  <div>
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      Участников
                    </span>
                    <p className="text-lg font-semibold text-gray-800 dark:text-white">
                      {stat.participants_count}
                    </p>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      Проехали
                    </span>
                    <p className="text-lg font-semibold text-gray-800 dark:text-white">
                      {stat.finished_count}
                    </p>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      Подарков
                    </span>
                    <p className="text-lg font-semibold text-gray-800 dark:text-white">
                      {stat.gifts_count}
                    </p>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500 dark:text-gray-400">
                      Призов
                    </span>
                    <p className="text-lg font-semibold text-gray-800 dark:text-white">
                      {stat.prizes_assigned_count}
                    </p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
