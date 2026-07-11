'use client';

import { useState } from 'react';
import { Dropdown } from '../ui/dropdown/Dropdown';
import Select from '../form/Select';
import Label from '../form/Label';
import type { Criteria } from '@/types';

// Фильтры списка участников. Применяются по кнопке «Применить» (staged):
// черновик правится в поповере, а запрос обновляется только при подтверждении.
export interface ParticipantFilters {
  gender: string; // '' = все
  bikeType: string; // '' = все
  isFinished: string; // '' = все, 'true' | 'false'
  hasGift: string; // 'all' | 'yes' | 'no'
  criteriaId: string; // '' = все
}

export const EMPTY_PARTICIPANT_FILTERS: ParticipantFilters = {
  gender: '',
  bikeType: '',
  isFinished: '',
  hasGift: 'all',
  criteriaId: '',
};

const GENDER_OPTIONS = [
  { value: '', label: 'Все' },
  { value: 'male', label: 'Мужской' },
  { value: 'female', label: 'Женский' },
];

const BIKE_TYPE_OPTIONS = [
  { value: '', label: 'Все' },
  { value: 'gravel', label: 'Гравийник' },
  { value: 'mtb', label: 'МТБ' },
  { value: 'road', label: 'Шоссе' },
  { value: 'single_speed', label: 'Фикс' },
  { value: 'tandem', label: 'Тандем' },
];

const STATUS_OPTIONS = [
  { value: '', label: 'Все' },
  { value: 'true', label: 'Проехал' },
  { value: 'false', label: 'Не проехал' },
];

const HAS_GIFT_OPTIONS = [
  { value: 'all', label: 'Все' },
  { value: 'yes', label: 'Да' },
  { value: 'no', label: 'Нет' },
];

/** Число активных (не дефолтных) фильтров — для бейджа на кнопке. */
function countActive(filters: ParticipantFilters): number {
  let count = 0;
  if (filters.gender !== '') count++;
  if (filters.bikeType !== '') count++;
  if (filters.isFinished !== '') count++;
  if (filters.hasGift !== 'all') count++;
  if (filters.criteriaId !== '') count++;
  return count;
}

interface ParticipantsFilterProps {
  /** Применённые фильтры (источник истины). */
  filters: ParticipantFilters;
  /** Доступные критерии для фильтрации участников по актуальному результату. */
  criteria: Criteria[];
  /** Подтверждение фильтров из поповера. */
  onApply: (next: ParticipantFilters) => void;
}

/**
 * Кнопка «Фильтр» с выпадающим поповером (см. ui/dropdown/Dropdown). Поповер
 * держит черновик фильтров; «Применить» отправляет его наверх и закрывает.
 * При каждом открытии черновик синхронизируется с применёнными значениями, так
 * что незавершённые правки сбрасываются.
 */
export default function ParticipantsFilter({
  filters,
  criteria,
  onApply,
}: ParticipantsFilterProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [draft, setDraft] = useState<ParticipantFilters>(filters);

  const activeCount = countActive(filters);
  const criteriaOptions = [
    { value: '', label: 'Все' },
    ...criteria.map((criterion) => ({
      value: String(criterion.id),
      label: criterion.name,
    })),
  ];

  const handleToggle = () => {
    if (isOpen) {
      setIsOpen(false);
    } else {
      setDraft(filters); // синхронизируем черновик с применёнными при открытии
      setIsOpen(true);
    }
  };

  const handleApply = () => {
    onApply(draft);
    setIsOpen(false);
  };

  const handleReset = () => {
    setDraft(EMPTY_PARTICIPANT_FILTERS);
    onApply(EMPTY_PARTICIPANT_FILTERS);
    setIsOpen(false);
  };

  return (
    <div className="relative inline-block text-left">
      <button
        type="button"
        onClick={handleToggle}
        className="dropdown-toggle inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:bg-white/[0.03] dark:text-gray-300 dark:hover:bg-white/[0.05]"
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 20 20"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          aria-hidden="true"
        >
          <path
            d="M2.5 5.5H11"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
          <path
            d="M15.5 5.5H17.5"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
          <circle cx="13" cy="5.5" r="2" stroke="currentColor" strokeWidth="1.5" />
          <path
            d="M2.5 14.5H5"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
          <path
            d="M9 14.5H17.5"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
          />
          <circle cx="7" cy="14.5" r="2" stroke="currentColor" strokeWidth="1.5" />
        </svg>
        Фильтр
        {activeCount > 0 && (
          <span className="inline-flex h-5 min-w-[20px] items-center justify-center rounded-full bg-brand-500 px-1.5 text-xs font-medium text-white">
            {activeCount}
          </span>
        )}
      </button>

      <Dropdown
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        className="max-h-[calc(100vh-6rem)] w-72 overflow-y-auto p-4"
      >
        <div className="space-y-4">
          <div>
            <Label>Пол</Label>
            <Select
              options={GENDER_OPTIONS}
              placeholder="Все"
              defaultValue={draft.gender}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, gender: value }))
              }
            />
          </div>

          <div>
            <Label>Тип велосипеда</Label>
            <Select
              options={BIKE_TYPE_OPTIONS}
              placeholder="Все"
              defaultValue={draft.bikeType}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, bikeType: value }))
              }
            />
          </div>

          <div>
            <Label>Статус</Label>
            <Select
              options={STATUS_OPTIONS}
              placeholder="Все"
              defaultValue={draft.isFinished}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, isFinished: value }))
              }
            />
          </div>

          <div>
            <Label>Добавил приз</Label>
            <Select
              options={HAS_GIFT_OPTIONS}
              placeholder="Все"
              defaultValue={draft.hasGift}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, hasGift: value }))
              }
            />
          </div>

          <div>
            <Label>Критерий</Label>
            <Select
              options={criteriaOptions}
              placeholder="Все"
              key={`criteria-${draft.criteriaId}`}
              defaultValue={draft.criteriaId}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, criteriaId: value }))
              }
            />
          </div>

          <button
            type="button"
            onClick={handleApply}
            className="w-full rounded-lg bg-brand-500 px-4 py-2.5 text-sm font-medium text-white hover:bg-brand-600"
          >
            Применить
          </button>

          {activeCount > 0 && (
            <button
              type="button"
              onClick={handleReset}
              className="w-full rounded-lg px-4 py-2 text-sm font-medium text-gray-500 hover:bg-gray-50 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-white/[0.05]"
            >
              Сбросить
            </button>
          )}
        </div>
      </Dropdown>
    </div>
  );
}
