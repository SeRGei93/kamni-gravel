"use client";

import type { BikeTypeFilter, GenderFilter } from "@/types";
import { BIKE_TYPE_OPTIONS } from "@/constants";

export type MiniappGenderFilter = "all_genders" | GenderFilter;

interface GiftFiltersProps {
  gender: MiniappGenderFilter;
  bikeType: BikeTypeFilter;
  isLoading?: boolean;
  onGenderChange: (value: MiniappGenderFilter) => void;
  onBikeTypeChange: (value: BikeTypeFilter) => void;
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
  isLoading,
  onGenderChange,
  onBikeTypeChange,
}: GiftFiltersProps) {
  return (
    <section
      aria-busy={isLoading}
      className="tg-topbar sticky top-0 z-10 border-b px-3 py-2 backdrop-blur"
    >
      <div className="mx-auto flex w-full max-w-md flex-col gap-2">
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
      </div>
    </section>
  );
}
