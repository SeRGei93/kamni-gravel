"use client";

import { useMemo, useState } from "react";
import type { ManualGift, MiniappParticipantOption } from "@/types";
import {
  isRecipientSelectionChanged,
  miniappGiftMutationErrorMessage,
} from "@/utils/miniappMyGifts";

interface MyGiftRecipientSelectProps {
  gift: ManualGift;
  participants: MiniappParticipantOption[];
  isSaving: boolean;
  onSave: (giftID: number, participantID: number | null) => Promise<void>;
}

export default function MyGiftRecipientSelect({
  gift,
  participants,
  isSaving,
  onSave,
}: MyGiftRecipientSelectProps) {
  const [search, setSearch] = useState("");
  const [selectedID, setSelectedID] = useState<number | null>(
    gift.recipient?.id ?? null
  );
  const [error, setError] = useState<string | null>(null);

  const options = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) {
      return participants;
    }
    return participants.filter((participant) =>
      [participant.display_name, participant.username ?? ""]
        .join(" ")
        .toLowerCase()
        .includes(needle)
    );
  }, [participants, search]);

  const save = async () => {
    try {
      setError(null);
      await onSave(gift.id, selectedID);
    } catch (saveError) {
      console.warn("[miniapp] manual recipient update failed", {
        giftId: gift.id,
        participantId: selectedID,
        message: saveError instanceof Error ? saveError.message : "Unknown error",
      });
      setError(miniappGiftMutationErrorMessage(saveError));
    }
  };

  return (
    <div className="tg-divider mt-3 rounded-lg border p-3">
      <label className="tg-title block text-xs font-semibold" htmlFor={`recipient-search-${gift.id}`}>
        Получатель
      </label>
      <input
        id={`recipient-search-${gift.id}`}
        type="search"
        value={search}
        onChange={(event) => setSearch(event.target.value)}
        placeholder="Поиск участника"
        className="tg-divider tg-title mt-2 h-9 w-full rounded-lg border bg-transparent px-2.5 text-xs outline-none focus:border-[var(--tg-button-color)]"
      />
      <select
        value={selectedID ?? ""}
        onChange={(event) => setSelectedID(event.target.value ? Number(event.target.value) : null)}
        className="tg-divider tg-title mt-2 h-9 w-full rounded-lg border bg-transparent px-2.5 text-xs outline-none focus:border-[var(--tg-button-color)]"
        disabled={isSaving}
      >
        <option value="">Получатель пока не выбран</option>
        {options.map((participant) => (
          <option key={participant.id} value={participant.id}>
            {participant.display_name}
            {participant.username ? ` (@${participant.username.replace(/^@+/, "")})` : ""}
          </option>
        ))}
      </select>
      {options.length === 0 && (
        <p className="tg-muted mt-2 text-xs">Участники по этому запросу не найдены.</p>
      )}
      {error && <p className="tg-error mt-2 text-xs leading-4">{error}</p>}
      <button
        type="button"
        onClick={() => void save()}
        disabled={isSaving || !isRecipientSelectionChanged(gift, selectedID)}
        className="tg-link-button mt-3 inline-flex min-h-9 items-center justify-center rounded-lg border px-3 py-2 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-50"
      >
        {isSaving ? "Сохраняем…" : selectedID === null ? "Очистить получателя" : "Сохранить получателя"}
      </button>
    </div>
  );
}
