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
    <section className="sticky top-0 z-10 border-y border-[#262626] bg-[#070707]/95 px-3 py-3 backdrop-blur">
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
                className={`h-8 shrink-0 border px-3 text-[11px] font-semibold uppercase transition ${
                  gender === value
                    ? "border-[#f97316] bg-[#f97316] text-white"
                    : "border-white/25 bg-[#111111] text-white"
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
                className={`h-8 shrink-0 border px-3 text-[11px] font-semibold uppercase transition ${
                  bikeType === value
                    ? "border-[#b7a87f] bg-[#b7a87f] text-[#070707]"
                    : "border-white/25 bg-[#111111] text-white"
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
