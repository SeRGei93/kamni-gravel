import Link from "next/link";
import { BIKE_TYPE_OPTIONS } from "@/constants";
import type { BikeTypeFilter, GenderFilter, Gift, GiftAttachment } from "@/types";
import { getCriteriaTypeLabel } from "@/utils/criteria";
import { formatGiftPlaceRule } from "@/utils/giftPlaceRule";
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
    <main className="tg-screen min-h-screen">
      <section className="tg-topbar border-b px-3 py-3">
        <div className="mx-auto flex w-full max-w-md items-center gap-3">
          <Link
            href="/miniapp/gifts"
            className="tg-link-button inline-flex rounded-lg border px-3 py-2 text-sm font-medium"
          >
            Назад
          </Link>
        </div>
      </section>

      <section className="mx-auto flex w-full max-w-md flex-col gap-3 px-3 py-3">
        <article className="tg-card overflow-hidden rounded-xl border">
          <GiftPhotoGallery giftId={gift.id} photos={photos} />

          <div className="space-y-4 p-3">
            <div>
              <p className="tg-accent text-xs font-medium">
                Описание
              </p>
              <p className="tg-title mt-2 whitespace-pre-wrap break-words text-base font-medium leading-6">
                {gift.description}
              </p>
            </div>

            <DetailRow label="От кого" value={donor} />

            <div className="tg-divider grid grid-cols-2 rounded-lg border text-sm">
              <DetailCell label="Пол" value={genderText[gender] ?? gender} />
              <DetailCell label="Велосипед" value={bikeText[bikeType] ?? bikeType} />
              <DetailCell
                label="Места"
                value={formatGiftPlaceRule(gift.place_rule ?? (gift.place ? { type: "places", places: [gift.place] } : null))}
                wide
              />
            </div>

            {criteria.length > 0 ? (
              <div>
                <p className="tg-muted text-[10px] font-semibold uppercase">
                  Критерии
                </p>
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
  const hasPrimaryPhoto = Boolean(primaryPhoto);

  return (
    <div className="tg-placeholder tg-divider border-b">
      <div className={hasPrimaryPhoto ? "aspect-[4/3]" : "h-36"}>
        <GiftImage giftId={giftId} attachment={primaryPhoto} variant="detail" />
      </div>

      {secondaryPhotos.length > 0 && (
        <div className="tg-divider grid grid-cols-2 gap-2 border-t p-2">
          {secondaryPhotos.map((photo) => (
            <div
              key={photo.id}
              className="tg-divider tg-placeholder aspect-square overflow-hidden rounded-lg border"
            >
              <GiftImage giftId={giftId} attachment={photo} variant="thumbnail" />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="tg-divider rounded-lg border px-3 py-2">
      <p className="tg-muted text-xs font-medium">{label}</p>
      <p className="tg-title mt-1 break-words text-sm font-medium">{value}</p>
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
      className={`tg-divider px-3 py-2 ${
        wide ? "col-span-2 border-t" : "border-r last:border-r-0"
      }`}
    >
      <p className="tg-muted text-xs font-medium">{label}</p>
      <p className="tg-title mt-1 break-words font-medium">{value}</p>
    </div>
  );
}
