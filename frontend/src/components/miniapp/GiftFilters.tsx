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
    <section className="sticky top-0 z-10 border-b border-[#2a2720]/25 bg-[#8f3d24]/95 px-3 py-3 backdrop-blur">
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
                    ? "border-[#2a2720] bg-[#dd7a3c] text-[#fff0d0]"
                    : "border-[#2a2720]/20 bg-[#fff0d0] text-[#2a2720]"
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
                    ? "border-[#2a2720] bg-[#d5a13a] text-[#211c16]"
                    : "border-[#2a2720]/20 bg-[#fff0d0] text-[#2a2720]"
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
