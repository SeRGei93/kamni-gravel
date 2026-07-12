import type { Gift } from "@/types";
import DonorProfileLink from "./DonorProfileLink";
import GiftDistributionConditions from "./GiftDistributionConditions";
import GiftPhotoGallery from "./GiftPhotoGallery";

interface GiftDetailViewProps {
  gift: Gift;
}

export default function GiftDetailView({ gift }: GiftDetailViewProps) {
  const donorName = [gift.first_name, gift.last_name].filter(Boolean).join(" ");
  const donorUsername = (gift.username ?? "").replace(/^@+/, "").trim();
  const donor = donorName || (donorUsername ? `@${donorUsername}` : `Участник ${gift.user_id}`);

  return (
    <main className="tg-screen min-h-screen">
      <section className="mx-auto flex w-full max-w-md flex-col gap-3 px-3 py-3">
        <article className="tg-card overflow-hidden rounded-xl border">
          <GiftPhotoGallery giftId={gift.id} attachments={gift.attachments} />

          <div className="space-y-4 p-3">
            <div>
              <p className="tg-accent text-xs font-medium">
                Описание
              </p>
              <p className="tg-title mt-2 whitespace-pre-wrap break-words text-base font-medium leading-6">
                {gift.description}
              </p>
            </div>

            <DetailRow label="От кого" value={donor} username={donorUsername || undefined} />

            {gift.manual_distribution ? (
              <div className="tg-soft-accent rounded-lg border px-3 py-2 text-sm">
                <p className="tg-title font-medium">Ручное распределение</p>
                <p className="tg-muted mt-1 text-xs leading-4">
                  Получателя этого приза выбирает его даритель. Условия по полу, велосипеду, местам и критериям не применяются.
                </p>
              </div>
            ) : (
              <GiftDistributionConditions
                genderFilter={gift.gender_filter}
                bikeTypeFilter={gift.bike_type_filter}
                place={gift.place}
                placeRule={gift.place_rule}
                criteria={gift.criteria}
              />
            )}
          </div>
        </article>
      </section>
    </main>
  );
}

function DetailRow({
  label,
  value,
  username,
}: {
  label: string;
  value: string;
  username?: string;
}) {
  return (
    <div className="tg-divider rounded-lg border px-3 py-2">
      <p className="tg-muted text-xs font-medium">{label}</p>
      <p className="tg-title mt-1 break-words text-sm font-medium">
        {username ? <DonorProfileLink label={value} username={username} /> : value}
      </p>
    </div>
  );
}
