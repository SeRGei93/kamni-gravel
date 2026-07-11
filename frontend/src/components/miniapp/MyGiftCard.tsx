"use client";

import type { ManualGift, MiniappParticipantOption } from "@/types";
import {
  miniappGiftModeLabel,
  miniappGiftRecipientLabel,
  miniappGiftReviewLabel,
} from "@/utils/miniappMyGifts";
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
  const isManual = gift.manual_distribution;

  return (
    <article className="tg-card rounded-xl border p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="tg-title break-words text-sm font-semibold leading-5">
            {gift.description}
          </h2>
          <p className="tg-muted mt-1 text-xs">{miniappGiftReviewLabel(gift)}</p>
        </div>
        <span className={`shrink-0 rounded-full px-2 py-1 text-[10px] font-semibold ${
          isManual ? "tg-soft-accent" : "tg-divider tg-muted border"
        }`}>
          {miniappGiftModeLabel(gift)}
        </span>
      </div>

      {isManual ? (
        <>
          <p className="tg-muted mt-3 text-xs leading-4">
            Сейчас: <span className="tg-title font-semibold">{miniappGiftRecipientLabel(gift)}</span>
          </p>
          <MyGiftRecipientSelect
            key={`${gift.id}:${gift.recipient?.id ?? "none"}`}
            gift={gift}
            participants={participants}
            isSaving={savingGiftID === gift.id}
            onSave={onSaveRecipient}
          />
        </>
      ) : (
        <p className="tg-muted tg-divider mt-3 rounded-lg border px-3 py-2 text-xs leading-4">
          Этот приз участвует в автоматическом распределении. Получателя для него выбирает система по условиям приза.
        </p>
      )}
    </article>
  );
}
