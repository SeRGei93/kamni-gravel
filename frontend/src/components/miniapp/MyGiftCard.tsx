"use client";

import type { ManualGift, MiniappParticipantOption } from "@/types";
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
    <article className="tg-card rounded-xl border p-3">
      <h2 className="tg-title break-words text-sm font-semibold leading-5">
        {gift.description}
      </h2>
      {gift.manual_distribution && !gift.recipient && (
        <MyGiftRecipientSelect
          gift={gift}
          participants={participants}
          isSaving={savingGiftID === gift.id}
          onSave={onSaveRecipient}
        />
      )}
    </article>
  );
}
