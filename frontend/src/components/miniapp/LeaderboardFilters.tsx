"use client";

import type { LeaderboardGenderFilter } from "@/components/miniapp/MiniappLeaderboardContext";
import MiniappSearchInput from "@/components/miniapp/MiniappSearchInput";
import type { BikeTypeFilter } from "@/types";
import { BIKE_TYPE_OPTIONS } from "@/constants";

interface LeaderboardFiltersProps {
  gender: LeaderboardGenderFilter;
  bikeType: BikeTypeFilter;
  searchQuery: string;
  onGenderChange: (value: LeaderboardGenderFilter) => void;
  onBikeTypeChange: (value: BikeTypeFilter) => void;
  onSearchChange: (value: string) => void;
  onSearchClear: () => void;
}

const genderOptions: Array<{ value: LeaderboardGenderFilter; label: string }> = [
  { value: "all", label: "Все" },
  { value: "male", label: "Мужчины" },
  { value: "female", label: "Женщины" },
];

const filterButtonClass =
  "h-7 shrink-0 rounded-md border px-2.5 text-[11px] font-medium transition active:scale-[0.98]";
const activeFilterClass = "tg-filter-active shadow-sm";
const inactiveFilterClass = "tg-filter-inactive";

export default function LeaderboardFilters({
  gender,
  bikeType,
  searchQuery,
  onGenderChange,
  onBikeTypeChange,
  onSearchChange,
  onSearchClear,
}: LeaderboardFiltersProps) {
  return (
    <div className="flex w-full flex-col gap-2">
      <MiniappSearchInput
        value={searchQuery}
        onChange={onSearchChange}
        onClear={onSearchClear}
        placeholder="Поиск по имени"
        ariaLabel="Поиск участника в лидерборде"
      />
      <div className="flex max-w-full gap-1.5 overflow-x-auto pb-1">
        {genderOptions.map((option) => {
          const isActive = gender === option.value;
          return (
            <button
              key={option.value}
              type="button"
              aria-pressed={isActive}
              onClick={() => onGenderChange(option.value)}
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
    </div>
  );
}
