"use client";

import type { ManualGift, ManualGiftRecipient, MiniappParticipantOption } from "@/types";
import GiftDistributionConditions from "./GiftDistributionConditions";
import GiftPhotoGallery from "./GiftPhotoGallery";
import MyGiftRecipientSelect from "./MyGiftRecipientSelect";
import TelegramProfileLink from "./TelegramProfileLink";

interface MyGiftCardProps {
  gift: ManualGift;
  participants: MiniappParticipantOption[];
  isSaving: boolean;
  showGiftRecipients: boolean;
  onSaveRecipient: (giftID: number, participantID: number | null) => Promise<void>;
  onAssignRandomRecipient: (giftID: number) => Promise<void>;
  onAssignRandomRecipientIncludingAwarded: (giftID: number) => Promise<void>;
}

export default function MyGiftCard({
  gift,
  participants,
  isSaving,
  showGiftRecipients,
  onSaveRecipient,
  onAssignRandomRecipient,
  onAssignRandomRecipientIncludingAwarded,
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
              isSaving={isSaving}
              onSave={onSaveRecipient}
              onAssignRandom={onAssignRandomRecipient}
              onAssignRandomIncludingAwarded={onAssignRandomRecipientIncludingAwarded}
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
            {showGiftRecipients && gift.recipients && gift.recipients.length > 0 && (
              <div className="mt-3">
                <p className="tg-muted text-sm leading-5">
                  {gift.recipients.length === 1 ? "Получатель:" : "Получатели:"}
                </p>
                <ul className="mt-1 space-y-1">
                  {gift.recipients.map((recipient) => (
                    <li key={recipient.id} className="tg-title text-sm font-semibold leading-5">
                      <TelegramProfileLink
                        label={recipientLabel(recipient)}
                        username={recipient.username}
                      />
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        )}
      </div>
    </article>
  );
}

function recipientLabel(recipient: ManualGiftRecipient): string {
  const username = recipient.username?.replace(/^@+/, "").trim();
  return username
    ? `${recipient.display_name} (@${username})`
    : recipient.display_name;
}
