"use client";

import type { BikeTypeFilter, GenderFilter } from "@/types";
import { BIKE_TYPE_OPTIONS } from "@/constants";
import MiniappSearchInput from "@/components/miniapp/MiniappSearchInput";

export type MiniappGenderFilter = "all_genders" | GenderFilter;

interface GiftFiltersProps {
  gender: MiniappGenderFilter;
  bikeType: BikeTypeFilter;
  searchQuery: string;
  isLoading?: boolean;
  onGenderChange: (value: MiniappGenderFilter) => void;
  onBikeTypeChange: (value: BikeTypeFilter) => void;
  onSearchChange: (value: string) => void;
  onSearchClear: () => void;
}

const genderOptions: Array<{ value: MiniappGenderFilter; label: string }> = [
  { value: "all_genders", label: "Все" },
  { value: "all", label: "Абсолют" },
  { value: "male", label: "Мужчины" },
  { value: "female", label: "Женщины" },
];

const filterButtonClass =
  "h-7 shrink-0 rounded-md border px-2.5 text-[11px] font-medium transition active:scale-[0.98]";
const activeFilterClass = "tg-filter-active shadow-sm";
const inactiveFilterClass = "tg-filter-inactive";

export default function GiftFilters({
  gender,
  bikeType,
  searchQuery,
  isLoading,
  onGenderChange,
  onBikeTypeChange,
  onSearchChange,
  onSearchClear,
}: GiftFiltersProps) {
  return (
    <div
      aria-busy={isLoading}
      className="flex w-full flex-col gap-2"
    >
      <div className="flex max-w-full gap-1.5 overflow-x-auto pb-1">
        {genderOptions.map((option) => {
          const value = option.value;
          const isActive = gender === value;

          return (
            <button
              key={option.value}
              type="button"
              aria-pressed={isActive}
              onClick={() => onGenderChange(value)}
              className={`${filterButtonClass} ${
                isActive ? activeFilterClass : inactiveFilterClass
              }`}
            >
              {option.label}
            </button>
          );
        })}
      </div>

      <div className="flex max-w-full gap-1.5 overflow-x-auto pb-1">
        {BIKE_TYPE_OPTIONS.map((option) => {
          const value = option.value as BikeTypeFilter;
          const isActive = bikeType === value;

          return (
            <button
              key={option.value}
              type="button"
              aria-pressed={isActive}
              onClick={() => onBikeTypeChange(value)}
              className={`${filterButtonClass} ${
                isActive ? activeFilterClass : inactiveFilterClass
              }`}
            >
              {option.label}
            </button>
          );
        })}
      </div>

      <MiniappSearchInput
        value={searchQuery}
        onChange={onSearchChange}
        onClear={onSearchClear}
        placeholder="Описание, имя или @username"
        ariaLabel="Поиск приза"
      />
    </div>
  );
}
