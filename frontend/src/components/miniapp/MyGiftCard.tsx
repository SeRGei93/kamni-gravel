"use client";

import type { ManualGift, MiniappParticipantOption } from "@/types";
import GiftDistributionConditions from "./GiftDistributionConditions";
import GiftPhotoGallery from "./GiftPhotoGallery";
import MyGiftRecipientSelect from "./MyGiftRecipientSelect";

interface MyGiftCardProps {
  gift: ManualGift;
  participants: MiniappParticipantOption[];
  savingGiftID: number | null;
  onSaveRecipient: (giftID: number, participantID: number | null) => Promise<void>;
}

export default function MyGiftCard({
  gift,
  participants,
  savingGiftID,
  onSaveRecipient,
}: MyGiftCardProps) {
  return (
    <article className="tg-card overflow-hidden rounded-xl border">
      <GiftPhotoGallery giftId={gift.id} attachments={gift.attachments} />

      <div className="space-y-4 p-3">
        <div>
          <p className="tg-accent text-xs font-medium">Описание</p>
          <h2 className="tg-title mt-2 whitespace-pre-wrap break-words text-base font-medium leading-6">
            {gift.description}
          </h2>
        </div>
        {gift.manual_distribution ? (
          <div>
            <p className="tg-muted text-xs font-medium">Ручное назначение</p>
            <MyGiftRecipientSelect
              gift={gift}
              participants={participants}
              isSaving={savingGiftID === gift.id}
              onSave={onSaveRecipient}
            />
          </div>
        ) : (
          <div>
            <p className="tg-muted mb-3 text-xs font-medium">Автоматическое распределение</p>
            <GiftDistributionConditions
              genderFilter={gift.gender_filter}
              bikeTypeFilter={gift.bike_type_filter}
              place={gift.place}
              placeRule={gift.place_rule}
              criteria={gift.criteria}
            />
          </div>
        )}
      </div>
    </article>
  );
}
