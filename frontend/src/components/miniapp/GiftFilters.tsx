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
  all: "Все",
  male: "Мужчины",
  female: "Женщины",
};

export default function GiftFilters({
  gender,
  bikeType,
  isLoading,
  onGenderChange,
  onBikeTypeChange,
}: GiftFiltersProps) {
  return (
    <section className="sticky top-0 z-10 border-b border-[#1f211c]/20 bg-[#7f9294]/95 px-3 py-3 backdrop-blur">
      <div className="mx-auto flex w-full max-w-md flex-col gap-3">
        <div className="flex max-w-full gap-2 overflow-x-auto pb-1">
          {GENDER_OPTIONS.map((option) => {
            const value = option.value as GenderFilter;
            return (
              <button
                key={option.value}
                type="button"
                disabled={isLoading}
                onClick={() => onGenderChange(value)}
                className={`h-9 shrink-0 rounded-md border px-3 text-sm font-medium transition ${
                  gender === value
                    ? "border-[#252821] bg-[#8f6c92] text-[#f4f0df]"
                    : "border-[#1f211c]/15 bg-[#f4f0df] text-[#252821]"
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
            return (
              <button
                key={option.value}
                type="button"
                disabled={isLoading}
                onClick={() => onBikeTypeChange(value)}
                className={`h-9 shrink-0 rounded-md border px-3 text-sm font-medium transition ${
                  bikeType === value
                    ? "border-[#252821] bg-[#9a812a] text-[#f4f0df]"
                    : "border-[#1f211c]/15 bg-[#f4f0df] text-[#252821]"
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
