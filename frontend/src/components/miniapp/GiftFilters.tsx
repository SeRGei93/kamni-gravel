"use client";

import type { BikeTypeFilter, GenderFilter } from "@/types";
import { BIKE_TYPE_OPTIONS, GENDER_OPTIONS } from "@/constants";

interface GiftFiltersProps {
  gender: GenderFilter;
  bikeType: BikeTypeFilter;
  isLoading?: boolean;
  onGenderChange: (value: GenderFilter) => void;
  onBikeTypeChange: (value: BikeTypeFilter) => void;
}

const genderLabels: Record<GenderFilter, string> = {
  all: "Абсолют",
  male: "Мужчины",
  female: "Женщины",
};

const filterButtonClass =
  "h-9 shrink-0 rounded-lg border px-3 text-sm font-medium transition active:scale-[0.98]";
const activeFilterClass =
  "border-orange-500 bg-orange-500 text-white shadow-sm shadow-orange-950/50";
const inactiveFilterClass =
  "border-gray-800 bg-gray-900 text-gray-300 hover:border-gray-700 hover:bg-gray-800";

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
      className="sticky top-0 z-10 border-b border-gray-800 bg-gray-950/95 px-3 py-3 backdrop-blur"
    >
      <div className="mx-auto flex w-full max-w-md flex-col gap-3">
        <div className="flex max-w-full gap-2 overflow-x-auto pb-1">
          {GENDER_OPTIONS.map((option) => {
            const value = option.value as GenderFilter;
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
                {genderLabels[value]}
              </button>
            );
          })}
        </div>

        <div className="flex max-w-full gap-2 overflow-x-auto pb-1">
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
