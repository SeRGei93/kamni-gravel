'use client';

import React from 'react';
import Link from 'next/link';
import Badge from '../ui/badge/Badge';
import type { Participant } from '@/types';
import { PARTICIPANT_STATUS_LABELS } from '@/types';
import { formatDistanceKm, formatSpeed } from '@/utils/format';

// Реестр колонок списка участников. Набор видимых колонок настраивается
// пользователем (см. useColumnPreferences + ColumnSettings); порядок колонок
// в таблице берётся отсюда. Значения не редактируются inline — правка остаётся
// в форме результата (/participants/[id]).

export interface ParticipantColumn {
  /** Стабильный ключ колонки (используется в localStorage). */
  key: string;
  /** Заголовок колонки. */
  label: string;
  /** Колонку нельзя скрыть (идентификация строки). */
  alwaysVisible?: boolean;
  /** Видима по умолчанию (для переключаемых колонок). */
  defaultVisible: boolean;
  /** Выравнивание содержимого ячейки. По умолчанию 'start'. */
  align?: 'start' | 'end' | 'center';
  /** Рендер содержимого ячейки для участника. */
  render: (participant: Participant) => React.ReactNode;
}

const GENDER_LABELS: Record<string, string> = {
  male: 'М',
  female: 'Ж',
};

const BIKE_TYPE_LABELS: Record<string, string> = {
  gravel: 'Гравийник',
  mtb: 'МТБ',
  road: 'Шоссе',
  single_speed: 'Фикс',
  tandem: 'Тандем',
};

const cellText = 'text-gray-800 text-theme-sm dark:text-white/90';
const cellMuted = 'text-gray-500 text-theme-sm dark:text-gray-400';

/** Текст ячейки или прочерк, если значение отсутствует. */
function textOrDash(value?: string | number | null): React.ReactNode {
  if (value === undefined || value === null || value === '') {
    return <span className={cellMuted}>-</span>;
  }
  return <span className={cellText}>{value}</span>;
}

/** Дата+время в локали ru-RU или прочерк. */
function dateTimeOrDash(iso?: string | null): React.ReactNode {
  if (!iso) {
    return <span className={cellMuted}>-</span>;
  }
  return <span className={cellText}>{new Date(iso).toLocaleString('ru-RU')}</span>;
}

// Упорядоченный реестр колонок. Порядок здесь определяет порядок в таблице и
// в выпадающем списке настроек.
export const PARTICIPANT_COLUMNS: ParticipantColumn[] = [
  {
    key: 'user_id',
    label: 'Telegram ID',
    defaultVisible: false,
    render: (p) => textOrDash(p.user_id),
  },
  {
    key: 'place',
    label: 'Место',
    defaultVisible: true,
    render: (p) => (
      <span className="font-medium text-gray-800 text-theme-sm dark:text-white/90">
        {p.place && p.place > 0 ? p.place : '-'}
      </span>
    ),
  },
  {
    key: 'username',
    label: 'Username',
    alwaysVisible: true,
    defaultVisible: true,
    render: (p) => (
      <Link
        href={`/participants/${p.id}`}
        className="font-medium text-brand-500 hover:text-brand-600 dark:text-brand-400 dark:hover:text-brand-300"
      >
        {p.username || `@user${p.user_id}`}
      </Link>
    ),
  },
  {
    key: 'name',
    label: 'Имя',
    alwaysVisible: true,
    defaultVisible: true,
    render: (p) =>
      textOrDash(`${p.first_name ?? ''} ${p.last_name ?? ''}`.trim()),
  },
  {
    key: 'gender',
    label: 'Пол',
    defaultVisible: true,
    render: (p) => (
      <Badge color={p.gender === 'male' ? 'info' : 'warning'} size="sm">
        {GENDER_LABELS[p.gender] || p.gender}
      </Badge>
    ),
  },
  {
    key: 'bike_type',
    label: 'Велосипед',
    defaultVisible: true,
    render: (p) => (
      <Badge color="light" size="sm">
        {BIKE_TYPE_LABELS[p.bike_type] || p.bike_type}
      </Badge>
    ),
  },
  {
    key: 'status',
    label: 'Статус',
    defaultVisible: true,
    render: (p) =>
      p.status && p.status !== 'active' ? (
        <Badge
          color={p.status === 'disqualified' ? 'error' : 'warning'}
          size="sm"
        >
          {PARTICIPANT_STATUS_LABELS[p.status]}
        </Badge>
      ) : (
        <span className={cellMuted}>-</span>
      ),
  },
  {
    key: 'elapsed_time',
    label: 'Общее время',
    defaultVisible: true,
    render: (p) => textOrDash(p.elapsed_time),
  },
  {
    key: 'moving_time',
    label: 'Чистое время',
    defaultVisible: true,
    render: (p) => textOrDash(p.moving_time),
  },
  {
    key: 'has_gift',
    label: 'Добавил приз',
    defaultVisible: true,
    render: (p) =>
      p.has_gift ? (
        <Badge color="success" size="sm">
          Да
        </Badge>
      ) : (
        <Badge color="light" size="sm">
          Нет
        </Badge>
      ),
  },
  {
    key: 'prizes_count',
    label: 'Получит приз',
    defaultVisible: true,
    render: (p) =>
      p.prizes_count > 0 ? (
        <Badge color="warning" size="sm">
          {p.prizes_count}
        </Badge>
      ) : (
        <span className={cellMuted}>-</span>
      ),
  },
  // --- Off-by-default: места в разных зачётах ---
  {
    key: 'place_absolute',
    label: 'Абсолют',
    defaultVisible: false,
    render: (p) => textOrDash(p.place_absolute),
  },
  {
    key: 'place_by_gender',
    label: 'Место (пол)',
    defaultVisible: false,
    render: (p) => textOrDash(p.place_by_gender),
  },
  {
    key: 'place_by_gender_bike',
    label: 'Место (пол+вел)',
    defaultVisible: false,
    render: (p) => textOrDash(p.place_by_gender_bike),
  },
  // --- Off-by-default: метрики заезда ---
  {
    key: 'started_at',
    label: 'Старт',
    defaultVisible: false,
    render: (p) => dateTimeOrDash(p.started_at),
  },
  {
    key: 'ride_finished_at',
    label: 'Финиш',
    defaultVisible: false,
    render: (p) => dateTimeOrDash(p.ride_finished_at),
  },
  {
    key: 'distance_km',
    label: 'Дистанция',
    defaultVisible: false,
    render: (p) => textOrDash(formatDistanceKm(p.distance_meters)),
  },
  {
    key: 'peak_speed_kmh',
    label: 'Пиковая скорость',
    defaultVisible: false,
    render: (p) => textOrDash(formatSpeed(p.peak_speed_kmh)),
  },
  {
    key: 'avg_speed_kmh',
    label: 'Ср. скорость',
    defaultVisible: false,
    render: (p) => textOrDash(formatSpeed(p.avg_speed_kmh)),
  },
  {
    key: 'avg_moving_speed_kmh',
    label: 'Ср. скорость (движение)',
    defaultVisible: false,
    render: (p) => textOrDash(formatSpeed(p.avg_moving_speed_kmh)),
  },
  {
    key: 'idle_time',
    label: 'Простой',
    defaultVisible: false,
    render: (p) => textOrDash(p.idle_time),
  },
  {
    key: 'calories',
    label: 'Калории',
    defaultVisible: false,
    render: (p) => textOrDash(p.calories),
  },
  {
    key: 'avg_heart_rate',
    label: 'Ср. пульс',
    defaultVisible: false,
    render: (p) => textOrDash(p.avg_heart_rate),
  },
  {
    key: 'max_heart_rate',
    label: 'Макс. пульс',
    defaultVisible: false,
    render: (p) => textOrDash(p.max_heart_rate),
  },
  {
    key: 'avg_cadence',
    label: 'Каденс',
    defaultVisible: false,
    render: (p) => textOrDash(p.avg_cadence),
  },
  {
    key: 'ride_date',
    label: 'Дата заезда',
    defaultVisible: false,
    render: (p) => textOrDash(p.ride_date),
  },
];

/** localStorage-ключ настроек колонок списка участников. */
export const PARTICIPANT_COLUMNS_STORAGE_KEY = 'participants:visible-columns';

/** Все ключи колонок (включая всегда видимые). */
export const ALL_COLUMN_KEYS: string[] = PARTICIPANT_COLUMNS.map((c) => c.key);

/** Ключи переключаемых колонок (которыми можно управлять). */
export const TOGGLEABLE_COLUMN_KEYS: string[] = PARTICIPANT_COLUMNS.filter(
  (c) => !c.alwaysVisible,
).map((c) => c.key);

/** Ключи переключаемых колонок, видимых по умолчанию. */
export const DEFAULT_VISIBLE_KEYS: string[] = PARTICIPANT_COLUMNS.filter(
  (c) => !c.alwaysVisible && c.defaultVisible,
).map((c) => c.key);
