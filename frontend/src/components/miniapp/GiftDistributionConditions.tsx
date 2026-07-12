import { BIKE_TYPE_OPTIONS } from "@/constants";
import type { BikeTypeFilter, Criteria, GenderFilter, GiftPlaceRule } from "@/types";
import { getCriteriaTypeLabel } from "@/utils/criteria";
import { formatGiftPlaceRule } from "@/utils/giftPlaceRule";

interface GiftDistributionConditionsProps {
  genderFilter?: GenderFilter;
  bikeTypeFilter?: BikeTypeFilter;
  place?: number;
  placeRule?: GiftPlaceRule | null;
  criteria?: Criteria[];
}

const genderText: Record<GenderFilter, string> = {
  all: "абсолютный зачёт",
  male: "мужчины",
  female: "женщины",
};

const bikeText = BIKE_TYPE_OPTIONS.reduce<Record<string, string>>((acc, option) => {
  acc[option.value] = option.value === "all" ? "любой" : option.label;
  return acc;
}, {});

export default function GiftDistributionConditions({
  genderFilter = "all",
  bikeTypeFilter = "all",
  place,
  placeRule,
  criteria = [],
}: GiftDistributionConditionsProps) {
  return (
    <>
      <div className="tg-divider grid grid-cols-2 rounded-lg border text-sm">
        <DetailCell label="Пол" value={genderText[genderFilter] ?? genderFilter} />
        <DetailCell label="Велосипед" value={bikeText[bikeTypeFilter] ?? bikeTypeFilter} />
        <DetailCell
          label="Места"
          value={formatGiftPlaceRule(placeRule ?? (place ? { type: "places", places: [place] } : null))}
          wide
        />
      </div>

      {criteria.length > 0 ? (
        <div>
          <p className="tg-muted text-[10px] font-semibold uppercase">Критерии</p>
          <div className="mt-2 flex flex-wrap gap-2">
            {criteria.map((criterion) => (
              <span
                key={criterion.id}
                className="tg-soft-accent rounded-md border px-2 py-1 text-xs font-medium"
              >
                {criterion.name || getCriteriaTypeLabel(criterion.criteria_type)}
              </span>
            ))}
          </div>
        </div>
      ) : (
        <p className="tg-divider tg-muted rounded-lg border px-3 py-2 text-xs font-medium">
          Без дополнительных критериев.
        </p>
      )}
    </>
  );
}

function DetailCell({
  label,
  value,
  wide = false,
}: {
  label: string;
  value: string;
  wide?: boolean;
}) {
  return (
    <div
      className={`tg-divider px-3 py-2 ${
        wide ? "col-span-2 border-t" : "border-r last:border-r-0"
      }`}
    >
      <p className="tg-muted text-xs font-medium">{label}</p>
      <p className="tg-title mt-1 break-words font-medium">{value}</p>
    </div>
  );
}
