"use client";

import type { KeyboardEvent } from "react";
import { useRouter } from "next/navigation";
import { BIKE_TYPE_OPTIONS } from "@/constants";
import type { BikeTypeFilter, GenderFilter, Gift } from "@/types";
import { getCriteriaTypeLabel } from "@/utils/criteria";
import {
  buildMissingGiftPlaceRanges,
  formatGiftPlaceRule,
  formatPlaceRange,
  getGiftFirstFixedPlace,
  type GiftPlaceRange,
} from "@/utils/giftPlaceRule";
import GiftImage from "./GiftImage";

interface GiftCatalogTableProps {
  gifts: Gift[];
  isLoading?: boolean;
  participantCount?: number;
  showPlaceGaps?: boolean;
}

type GiftCatalogRow =
  | { type: "gift"; gift: Gift }
  | { type: "gap"; ranges: GiftPlaceRange[] };

const genderText: Record<GenderFilter, string> = {
  all: "абсолют",
  male: "мужчины",
  female: "женщины",
};

const bikeText = BIKE_TYPE_OPTIONS.reduce<Record<string, string>>((acc, option) => {
  acc[option.value] = option.value === "all" ? "любой" : option.label;
  return acc;
}, {});

export default function GiftCatalogTable({
  gifts,
  isLoading,
  participantCount,
  showPlaceGaps = false,
}: GiftCatalogTableProps) {
  const rows = buildGiftCatalogRows(gifts, participantCount, showPlaceGaps);

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
          {rows.map((row) => (
            row.type === "gift" ? (
              <GiftTableRow key={`gift-${row.gift.id}`} gift={row.gift} />
            ) : (
              <GiftGapRow
                key={`gap-${row.ranges.map(formatPlaceRange).join(",")}`}
                ranges={row.ranges}
              />
            )
          ))}
        </tbody>
      </table>
    </section>
  );
}

function buildGiftCatalogRows(
  gifts: Gift[],
  participantCount?: number,
  showPlaceGaps = false
): GiftCatalogRow[] {
  if (!showPlaceGaps || !participantCount || participantCount <= 0) {
    return gifts.map((gift) => ({ type: "gift", gift }));
  }

  const gapRanges = buildMissingGiftPlaceRanges(gifts, participantCount);
  if (gapRanges.length === 0) {
    return gifts.map((gift) => ({ type: "gift", gift }));
  }

  const rows: GiftCatalogRow[] = [];
  let gapIndex = 0;

  for (const gift of gifts) {
    const giftPlace = getGiftFirstFixedPlace(gift) ?? Number.POSITIVE_INFINITY;
    const pendingRanges: GiftPlaceRange[] = [];
    while (gapIndex < gapRanges.length && gapRanges[gapIndex].start < giftPlace) {
      pendingRanges.push(gapRanges[gapIndex]);
      gapIndex += 1;
    }

    if (pendingRanges.length > 0) {
      rows.push({ type: "gap", ranges: pendingRanges });
    }

    rows.push({ type: "gift", gift });
  }

  const tailRanges: GiftPlaceRange[] = [];
  while (gapIndex < gapRanges.length) {
    tailRanges.push(gapRanges[gapIndex]);
    gapIndex += 1;
  }

  if (tailRanges.length > 0) {
    rows.push({ type: "gap", ranges: tailRanges });
  }

  return rows;
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

function GiftGapRow({ ranges }: { ranges: GiftPlaceRange[] }) {
  const placeText = ranges.map(formatPlaceRange).join(", ");
  const hasSinglePlace = ranges.length === 1 && ranges[0].start === ranges[0].end;
  const suffix = hasSinglePlace ? "еще нет приза" : "еще нет призов";

  return (
    <tr className="align-top" aria-label={`${placeText} ${suffix}`}>
      <td colSpan={3} className="px-2 py-1">
        <div className="tg-divider tg-muted rounded-md border border-dashed px-2.5 py-1.5 text-center text-[11px] font-semibold leading-4">
          {placeText} {suffix}
        </div>
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
      <ConditionLine
        label="Места"
        value={formatGiftPlaceRule(gift.place_rule ?? (gift.place ? { type: "places", places: [gift.place] } : null))}
      />
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
