"use client";

import type { KeyboardEvent } from "react";
import { useRouter } from "next/navigation";
import { BIKE_TYPE_OPTIONS } from "@/constants";
import type { BikeTypeFilter, GenderFilter, Gift } from "@/types";
import { getCriteriaTypeLabel } from "@/utils/criteria";
import GiftImage from "./GiftImage";

interface GiftCatalogTableProps {
  gifts: Gift[];
  isLoading?: boolean;
}

const genderText: Record<GenderFilter, string> = {
  all: "абсолют",
  male: "мужчины",
  female: "женщины",
};

const bikeText = BIKE_TYPE_OPTIONS.reduce<Record<string, string>>((acc, option) => {
  acc[option.value] = option.value === "all" ? "любой" : option.label;
  return acc;
}, {});

export default function GiftCatalogTable({ gifts, isLoading }: GiftCatalogTableProps) {
  return (
    <section
      className={`tg-card overflow-hidden rounded-xl border ${
        isLoading ? "opacity-70" : ""
      }`}
      aria-busy={isLoading}
    >
      <table className="w-full table-fixed border-collapse">
        <colgroup>
          <col className="w-[52px]" />
          <col />
          <col className="w-28" />
        </colgroup>
        <thead className="tg-topbar">
          <tr className="tg-divider tg-muted border-b text-left text-[10px] font-semibold uppercase">
            <th scope="col" className="px-1.5 py-2">
              Фото
            </th>
            <th scope="col" className="px-1.5 py-2">
              Приз
            </th>
            <th scope="col" className="px-1.5 py-2">
              Условия
            </th>
          </tr>
        </thead>
        <tbody className="tg-table-body">
          {gifts.map((gift) => (
            <GiftTableRow key={gift.id} gift={gift} />
          ))}
        </tbody>
      </table>
    </section>
  );
}

function GiftTableRow({ gift }: { gift: Gift }) {
  const router = useRouter();
  const photo = gift.attachments?.find((attachment) => attachment.file_type === "photo");
  const donorName = [gift.first_name, gift.last_name].filter(Boolean).join(" ");
  const donor = donorName || gift.username || `Участник ${gift.user_id}`;
  const href = `/miniapp/gifts/${gift.id}`;

  const openGift = () => {
    router.push(href);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLTableRowElement>) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }

    event.preventDefault();
    openGift();
  };

  return (
    <tr
      role="link"
      tabIndex={0}
      aria-label={`Открыть приз ${gift.id}`}
      onClick={openGift}
      onKeyDown={handleKeyDown}
      className="tg-row-hover cursor-pointer align-top focus:outline-none focus:ring-2 focus:ring-[var(--tg-button-color)]"
    >
      <td className="py-1.5 pl-2 pr-1">
        <div className="tg-placeholder tg-divider block h-10 w-10 overflow-hidden rounded-lg border">
          <GiftImage giftId={gift.id} attachment={photo} variant="thumbnail" />
        </div>
      </td>
      <td className="min-w-0 px-1.5 py-1.5">
        <p className="tg-title line-clamp-1 break-words text-sm font-medium leading-5">
          {gift.description}
        </p>
        <p className="tg-muted mt-1 truncate text-[11px] font-medium leading-4">
          от {donor}
        </p>
      </td>
      <td className="px-1.5 py-1.5">
        <GiftCompactConditions gift={gift} />
      </td>
    </tr>
  );
}

function GiftCompactConditions({ gift }: { gift: Gift }) {
  const gender = (gift.gender_filter || "all") as GenderFilter;
  const bikeType = (gift.bike_type_filter || "all") as BikeTypeFilter;
  const criteria = gift.criteria ?? [];
  const criteriaText = criteria
    .map((criterion) => criterion.name || getCriteriaTypeLabel(criterion.criteria_type))
    .join(", ");

  return (
    <div className="tg-muted space-y-0.5 text-[10px] font-medium leading-[14px]">
      <ConditionLine label="Пол" value={genderText[gender] ?? gender} />
      <ConditionLine label="Вело" value={bikeText[bikeType] ?? bikeType} />
      {gift.place !== undefined && <ConditionLine label="Место" value={String(gift.place)} />}
      {criteriaText && <ConditionLine label="Кр." value={criteriaText} />}
    </div>
  );
}

function ConditionLine({ label, value }: { label: string; value: string }) {
  return (
    <p className="min-w-0">
      <span>{label}: </span>
      <span className="tg-title break-words">{value}</span>
    </p>
  );
}
