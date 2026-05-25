import Link from "next/link";
import { BIKE_TYPE_OPTIONS } from "@/constants";
import type { BikeTypeFilter, GenderFilter, Gift } from "@/types";
import { getCriteriaTypeLabel } from "@/utils/criteria";
import GiftImage from "./GiftImage";

interface GiftDetailViewProps {
  gift: Gift;
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

export default function GiftDetailView({ gift }: GiftDetailViewProps) {
  const photo = gift.attachments?.find((attachment) => attachment.file_type === "photo");
  const donorName = [gift.first_name, gift.last_name].filter(Boolean).join(" ");
  const donor = donorName || gift.username || `Участник ${gift.user_id}`;
  const gender = (gift.gender_filter || "all") as GenderFilter;
  const bikeType = (gift.bike_type_filter || "all") as BikeTypeFilter;
  const criteria = gift.criteria ?? [];

  return (
    <main className="min-h-screen bg-gray-50 text-gray-900">
      <section className="border-b border-gray-200 bg-white px-3 py-3">
        <div className="mx-auto flex w-full max-w-md items-center justify-between gap-3">
          <Link
            href="/miniapp/gifts"
            className="inline-flex rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-700"
          >
            Назад
          </Link>
          <h1 className="truncate text-lg font-semibold text-gray-900">
            Подарок
          </h1>
        </div>
      </section>

      <section className="mx-auto flex w-full max-w-md flex-col gap-3 px-3 py-3">
        <article className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
          <div className="aspect-[4/3] border-b border-gray-200 bg-orange-50">
            <GiftImage giftId={gift.id} attachment={photo} />
          </div>

          <div className="space-y-4 p-3">
            <div>
              <p className="text-xs font-medium text-orange-600">
                Описание
              </p>
              <p className="mt-2 whitespace-pre-wrap break-words text-base font-medium leading-6 text-gray-900">
                {gift.description}
              </p>
            </div>

            <DetailRow label="От кого" value={donor} />

            <div className="grid grid-cols-2 rounded-lg border border-gray-200 text-sm">
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
                      className="rounded-md border border-orange-200 bg-orange-50 px-2 py-1 text-xs font-medium text-orange-700"
                    >
                      {criterion.name || getCriteriaTypeLabel(criterion.criteria_type)}
                    </span>
                  ))}
                </div>
              </div>
            ) : (
              <p className="rounded-lg border border-gray-200 px-3 py-2 text-xs font-medium text-gray-500">
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
    <div className="rounded-lg border border-gray-200 px-3 py-2">
      <p className="text-xs font-medium text-gray-500">{label}</p>
      <p className="mt-1 break-words text-sm font-medium text-gray-900">{value}</p>
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
      className={`border-gray-200 px-3 py-2 ${
        wide ? "col-span-2 border-t" : "border-r last:border-r-0"
      }`}
    >
      <p className="text-xs font-medium text-gray-500">{label}</p>
      <p className="mt-1 break-words font-medium text-gray-900">{value}</p>
    </div>
  );
}
