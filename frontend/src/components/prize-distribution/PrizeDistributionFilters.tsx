'use client';

import type { BikeTypeFilter, GenderFilter } from '@/types';

interface PrizeDistributionFiltersProps {
  gender: GenderFilter;
  bikeType: BikeTypeFilter;
  matchReason: string;
  onGenderChange: (value: GenderFilter) => void;
  onBikeTypeChange: (value: BikeTypeFilter) => void;
  onMatchReasonChange: (value: string) => void;
}

const GENDER_OPTIONS: Array<{ value: GenderFilter; label: string }> = [
  { value: 'all', label: 'Все' },
  { value: 'male', label: 'Мужчины' },
  { value: 'female', label: 'Женщины' },
];

const BIKE_TYPE_OPTIONS: Array<{ value: BikeTypeFilter; label: string }> = [
  { value: 'all', label: 'Все' },
  { value: 'gravel', label: 'Gravel' },
  { value: 'mtb', label: 'MTB' },
  { value: 'road', label: 'Шоссе' },
  { value: 'single_speed', label: 'Single Speed' },
  { value: 'tandem', label: 'Тандем' },
];

const filterButtonClass =
  'h-9 shrink-0 rounded-lg border px-3 text-sm font-medium transition-colors';

function FilterRow<T extends string>({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: T;
  options: Array<{ value: T; label: string }>;
  onChange: (value: T) => void;
}) {
  return (
    <div className="space-y-2">
      <p className="text-sm font-medium text-gray-700 dark:text-gray-300">{label}</p>
      <div className="flex max-w-full gap-2 overflow-x-auto pb-1">
        {options.map((option) => {
          const active = value === option.value;
          return (
            <button
              key={option.value}
              type="button"
              aria-pressed={active}
              onClick={() => onChange(option.value)}
              className={`${filterButtonClass} ${
                active
                  ? 'border-brand-500 bg-brand-500 text-white'
                  : 'border-gray-200 bg-white text-gray-700 hover:border-brand-300 hover:text-brand-600 dark:border-gray-700 dark:bg-white/[0.03] dark:text-gray-300 dark:hover:border-brand-700 dark:hover:text-brand-400'
              }`}
            >
              {option.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}

export default function PrizeDistributionFilters({
  gender,
  bikeType,
  matchReason,
  onGenderChange,
  onBikeTypeChange,
  onMatchReasonChange,
}: PrizeDistributionFiltersProps) {
  return (
    <div className="space-y-4">
      <FilterRow
        label="Пол"
        value={gender}
        options={GENDER_OPTIONS}
        onChange={onGenderChange}
      />
      <FilterRow
        label="Зачёт"
        value={bikeType}
        options={BIKE_TYPE_OPTIONS}
        onChange={onBikeTypeChange}
      />
      <label className="block max-w-sm">
        <span className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          Причина совпадения
        </span>
        <select
          value={matchReason}
          onChange={(event) => onMatchReasonChange(event.target.value)}
          className="h-11 w-full rounded-lg border border-gray-300 bg-white px-3 text-sm text-gray-800 shadow-theme-xs focus:border-brand-300 focus:outline-hidden focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
        >
          <option value="all">Все</option>
          <option value="criteria">По критериям</option>
          <option value="place">По месту</option>
          <option value="match">Без ограничений</option>
          <option value="no_match">Без приза</option>
        </select>
      </label>
    </div>
  );
}
