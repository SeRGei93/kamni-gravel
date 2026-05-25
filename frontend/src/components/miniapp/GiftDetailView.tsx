import Link from "next/link";
import { BIKE_TYPE_OPTIONS } from "@/constants";
import type { BikeTypeFilter, GenderFilter, Gift, GiftAttachment } from "@/types";
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
  const photos = gift.attachments?.filter((attachment) => attachment.file_type === "photo") ?? [];
  const donorName = [gift.first_name, gift.last_name].filter(Boolean).join(" ");
  const donor = donorName || gift.username || `Участник ${gift.user_id}`;
  const gender = (gift.gender_filter || "all") as GenderFilter;
  const bikeType = (gift.bike_type_filter || "all") as BikeTypeFilter;
  const criteria = gift.criteria ?? [];

  return (
    <main className="min-h-screen bg-gray-950 text-gray-100" style={{ colorScheme: "dark" }}>
      <section className="border-b border-gray-800 bg-gray-900 px-3 py-3">
        <div className="mx-auto flex w-full max-w-md items-center justify-between gap-3">
          <Link
            href="/miniapp/gifts"
            className="inline-flex rounded-lg border border-gray-700 bg-gray-900 px-3 py-2 text-sm font-medium text-gray-200"
          >
            Назад
          </Link>
          <h1 className="truncate text-lg font-semibold text-white">
            Подарок
          </h1>
        </div>
      </section>

      <section className="mx-auto flex w-full max-w-md flex-col gap-3 px-3 py-3">
        <article className="overflow-hidden rounded-xl border border-gray-800 bg-gray-900 shadow-sm">
          <GiftPhotoGallery giftId={gift.id} photos={photos} />

          <div className="space-y-4 p-3">
            <div>
              <p className="text-xs font-medium text-orange-400">
                Описание
              </p>
              <p className="mt-2 whitespace-pre-wrap break-words text-base font-medium leading-6 text-white">
                {gift.description}
              </p>
            </div>

            <DetailRow label="От кого" value={donor} />

            <div className="grid grid-cols-2 rounded-lg border border-gray-800 text-sm">
              <DetailCell label="Пол" value={genderText[gender] ?? gender} />
              <DetailCell label="Велосипед" value={bikeText[bikeType] ?? bikeType} />
              {gift.place !== undefined && (
                <DetailCell label="Место" value={String(gift.place)} wide />
              )}
            </div>

            {criteria.length > 0 ? (
              <div>
                <p className="text-[10px] font-semibold uppercase text-gray-400">
                  Критерии
                </p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {criteria.map((criterion) => (
                    <span
                      key={criterion.id}
                      className="rounded-md border border-orange-900/70 bg-orange-950/35 px-2 py-1 text-xs font-medium text-orange-300"
                    >
                      {criterion.name || getCriteriaTypeLabel(criterion.criteria_type)}
                    </span>
                  ))}
                </div>
              </div>
            ) : (
              <p className="rounded-lg border border-gray-800 px-3 py-2 text-xs font-medium text-gray-400">
                Без дополнительных критериев.
              </p>
            )}
          </div>
        </article>
      </section>
    </main>
  );
}

function GiftPhotoGallery({
  giftId,
  photos,
}: {
  giftId: number;
  photos: GiftAttachment[];
}) {
  const [primaryPhoto, ...secondaryPhotos] = photos;

  return (
    <div className="border-b border-gray-800 bg-gray-800">
      <div className="aspect-[4/3]">
        <GiftImage giftId={giftId} attachment={primaryPhoto} />
      </div>

      {secondaryPhotos.length > 0 && (
        <div className="grid grid-cols-2 gap-2 border-t border-gray-800 p-2">
          {secondaryPhotos.map((photo) => (
            <div
              key={photo.id}
              className="aspect-square overflow-hidden rounded-lg border border-gray-800 bg-gray-800"
            >
              <GiftImage giftId={giftId} attachment={photo} />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-gray-800 px-3 py-2">
      <p className="text-xs font-medium text-gray-400">{label}</p>
      <p className="mt-1 break-words text-sm font-medium text-white">{value}</p>
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
      className={`border-gray-800 px-3 py-2 ${
        wide ? "col-span-2 border-t" : "border-r last:border-r-0"
      }`}
    >
      <p className="text-xs font-medium text-gray-400">{label}</p>
      <p className="mt-1 break-words font-medium text-white">{value}</p>
    </div>
  );
}
