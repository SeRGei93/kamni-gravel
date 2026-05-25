import { BIKE_TYPE_OPTIONS } from "@/constants";
import type { BikeTypeFilter, GenderFilter, Gift } from "@/types";
import { getCriteriaTypeLabel } from "@/utils/criteria";
import GiftImage from "./GiftImage";

interface GiftCatalogTableProps {
  gifts: Gift[];
  isLoading?: boolean;
}

const genderText: Record<GenderFilter, string> = {
  all: "все",
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
      className={`overflow-hidden rounded-lg border border-[#2a2720]/25 bg-[#fff0d0] shadow-sm ${
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
        <thead className="bg-[#2a2720]">
          <tr className="border-b border-[#2a2720]/15 text-left text-[10px] font-semibold uppercase tracking-[0.08em] text-[#fff0d0]">
            <th scope="col" className="px-1.5 py-2">
              Фото
            </th>
            <th scope="col" className="px-1.5 py-2">
              Подарок
            </th>
            <th scope="col" className="px-1.5 py-2">
              Условия
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[#2a2720]/10">
          {gifts.map((gift) => (
            <GiftTableRow key={gift.id} gift={gift} />
          ))}
        </tbody>
      </table>
    </section>
  );
}

function GiftTableRow({ gift }: { gift: Gift }) {
  const photo = gift.attachments?.find((attachment) => attachment.file_type === "photo");
  const donorName = [gift.first_name, gift.last_name].filter(Boolean).join(" ");
  const donor = donorName || gift.username || `Участник ${gift.user_id}`;

  return (
    <tr className="align-top">
      <td className="py-1.5 pl-2 pr-1">
        <div className="h-10 w-10 overflow-hidden rounded-md border border-[#2a2720]/15 bg-[#dd7a3c]/20">
          <GiftImage giftId={gift.id} attachment={photo} />
        </div>
      </td>
      <td className="min-w-0 px-1.5 py-1.5">
        <p className="line-clamp-2 break-words text-sm font-semibold leading-5 text-[#211c16]">
          {gift.description}
        </p>
        <p className="mt-1 truncate text-[11px] leading-4 text-[#8f3d24]">от {donor}</p>
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
    <div className="space-y-0.5 text-[10px] font-medium leading-[14px] text-[#4a3020]">
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
      <span className="text-[#c96534]">{label}: </span>
      <span className="break-words text-[#2a2720]">{value}</span>
    </p>
  );
}
