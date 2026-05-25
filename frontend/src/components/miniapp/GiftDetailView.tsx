import Link from "next/link";
import { BIKE_TYPE_OPTIONS } from "@/constants";
import type { BikeTypeFilter, GenderFilter, Gift } from "@/types";
import { getCriteriaTypeLabel } from "@/utils/criteria";
import GiftImage from "./GiftImage";

interface GiftDetailViewProps {
  gift: Gift;
}

const genderText: Record<GenderFilter, string> = {
  all: "для всех",
  male: "мужчины",
  female: "женщины",
};

const bikeText = BIKE_TYPE_OPTIONS.reduce<Record<string, string>>((acc, option) => {
  acc[option.value] = option.value === "all" ? "любой" : option.label;
  return acc;
}, {});

export default function GiftDetailView({ gift }: GiftDetailViewProps) {
  const photo = gift.attachments?.find((attachment) => attachment.file_type === "photo");
  const donorName = [gift.first_name, gift.last_name].filter(Boolean).join(" ");
  const donor = donorName || gift.username || `Участник ${gift.user_id}`;
  const gender = (gift.gender_filter || "all") as GenderFilter;
  const bikeType = (gift.bike_type_filter || "all") as BikeTypeFilter;
  const criteria = gift.criteria ?? [];

  return (
    <main className="min-h-screen bg-[#070707] text-[#111111]">
      <section className="bg-[#070707] px-3 pb-3 pt-4 text-white">
        <div className="mx-auto w-full max-w-md border border-white/15 px-3 py-3">
          <Link
            href="/miniapp/gifts"
            className="inline-flex border border-white/20 px-2 py-1 text-[11px] font-semibold uppercase text-white"
          >
            Назад
          </Link>
          <p className="mt-4 text-[10px] font-semibold uppercase text-[#b7a87f]">
            Gift detail
          </p>
          <h1 className="mt-1 text-[38px] font-black uppercase leading-none text-white">
            Gift
          </h1>
        </div>
      </section>

      <section className="mx-auto flex w-full max-w-md flex-col gap-3 bg-white px-3 py-3">
        <article className="border border-[#d4d4d4] bg-white">
          <div className="aspect-[4/3] border-b border-[#d4d4d4] bg-[#fff7ed]">
            <GiftImage giftId={gift.id} attachment={photo} />
          </div>

          <div className="space-y-4 p-3">
            <div>
              <p className="text-[10px] font-semibold uppercase text-[#f97316]">
                Описание
              </p>
              <p className="mt-2 whitespace-pre-wrap break-words text-base font-semibold leading-6 text-[#111111]">
                {gift.description}
              </p>
            </div>

            <DetailRow label="От кого" value={donor} />

            <div className="grid grid-cols-2 border border-[#d4d4d4] text-sm">
              <DetailCell label="Пол" value={genderText[gender] ?? gender} />
              <DetailCell label="Велосипед" value={bikeText[bikeType] ?? bikeType} />
              {gift.place !== undefined && (
                <DetailCell label="Место" value={String(gift.place)} wide />
              )}
            </div>

            {criteria.length > 0 ? (
              <div>
                <p className="text-[10px] font-semibold uppercase text-[#737373]">
                  Критерии
                </p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {criteria.map((criterion) => (
                    <span
                      key={criterion.id}
                      className="border border-[#f97316] px-2 py-1 text-xs font-semibold text-[#111111]"
                    >
                      {criterion.name || getCriteriaTypeLabel(criterion.criteria_type)}
                    </span>
                  ))}
                </div>
              </div>
            ) : (
              <p className="border border-[#d4d4d4] px-3 py-2 text-xs font-semibold text-[#525252]">
                Без дополнительных критериев.
              </p>
            )}
          </div>
        </article>
      </section>
    </main>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-[#d4d4d4] px-3 py-2">
      <p className="text-[10px] font-semibold uppercase text-[#737373]">{label}</p>
      <p className="mt-1 break-words text-sm font-semibold text-[#111111]">{value}</p>
    </div>
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
      className={`border-[#d4d4d4] px-3 py-2 ${
        wide ? "col-span-2 border-t" : "border-r last:border-r-0"
      }`}
    >
      <p className="text-[10px] font-semibold uppercase text-[#737373]">{label}</p>
      <p className="mt-1 break-words font-semibold text-[#111111]">{value}</p>
    </div>
  );
}
