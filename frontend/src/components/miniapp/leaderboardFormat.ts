import type { BikeType, Gender } from "@/types";
import { BIKE_TYPE_OPTIONS } from "@/constants";

const genderShort: Record<Gender, string> = {
  male: "М",
  female: "Ж",
};

const genderFull: Record<Gender, string> = {
  male: "Мужской",
  female: "Женский",
};

const bikeLabels = BIKE_TYPE_OPTIONS.reduce<Record<string, string>>((acc, option) => {
  acc[option.value] = option.label;
  return acc;
}, {});

export function genderShortLabel(gender: Gender): string {
  return genderShort[gender] ?? gender;
}

export function genderFullLabel(gender: Gender): string {
  return genderFull[gender] ?? gender;
}

export function bikeTypeLabel(bikeType: BikeType): string {
  return bikeLabels[bikeType] ?? bikeType;
}
