'use client';

import { useState } from 'react';
import { Dropdown } from '../ui/dropdown/Dropdown';
import Checkbox from '../form/input/Checkbox';
import type { ParticipantColumn } from './participantColumns';

interface ColumnSettingsProps {
  /** Полный реестр колонок (включая всегда видимые). */
  columns: ParticipantColumn[];
  /** Видима ли колонка (из useColumnPreferences). */
  isVisible: (key: string) => boolean;
  /** Переключить колонку. */
  toggle: (key: string) => void;
  /** Сбросить к набору по умолчанию. */
  reset: () => void;
}

/**
 * Выпадающий список «Колонки»: чекбокс на каждую переключаемую колонку
 * (всегда видимые показаны отмеченными и неактивными) + сброс к дефолту.
 *
 * Реализация учитывает контракт ui/dropdown/Dropdown: он закрывается по клику
 * вне себя только если триггер несёт класс .dropdown-toggle. Поэтому состояние
 * открытия держим локально, а чекбоксы рендерим прямо внутри <Dropdown>.
 */
export default function ColumnSettings({
  columns,
  isVisible,
  toggle,
  reset,
}: ColumnSettingsProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative inline-block text-left">
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className="dropdown-toggle inline-flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2.5 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-700 dark:bg-white/[0.03] dark:text-gray-300 dark:hover:bg-white/[0.05]"
      >
        <svg
          width="18"
          height="18"
          viewBox="0 0 18 18"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          aria-hidden="true"
        >
          <rect
            x="2.25"
            y="2.25"
            width="13.5"
            height="13.5"
            rx="1.5"
            stroke="currentColor"
            strokeWidth="1.5"
          />
          <path d="M6.75 2.25V15.75" stroke="currentColor" strokeWidth="1.5" />
          <path d="M11.25 2.25V15.75" stroke="currentColor" strokeWidth="1.5" />
        </svg>
        Колонки
      </button>

      <Dropdown
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
        className="w-64 p-2"
      >
        <div className="mb-1 px-2 py-1 text-xs font-medium text-gray-500 dark:text-gray-400">
          Отображаемые колонки
        </div>

        <div className="max-h-80 space-y-1 overflow-y-auto px-1">
          {columns.map((column) => {
            const always = Boolean(column.alwaysVisible);
            return (
              <div
                key={column.key}
                className="rounded-lg px-2 py-1.5 hover:bg-gray-50 dark:hover:bg-white/[0.05]"
              >
                <Checkbox
                  label={column.label}
                  checked={always || isVisible(column.key)}
                  disabled={always}
                  onChange={() => toggle(column.key)}
                />
              </div>
            );
          })}
        </div>

        <div className="mt-2 border-t border-gray-100 pt-2 dark:border-white/[0.05]">
          <button
            type="button"
            onClick={reset}
            className="w-full rounded-lg px-2 py-1.5 text-start text-sm font-medium text-brand-500 hover:bg-brand-50 hover:text-brand-600 dark:text-brand-400 dark:hover:bg-brand-500/10"
          >
            Сбросить по умолчанию
          </button>
        </div>
      </Dropdown>
    </div>
  );
}
