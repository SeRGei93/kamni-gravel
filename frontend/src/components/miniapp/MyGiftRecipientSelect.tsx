"use client";

import {
  useEffect,
  useId,
  useMemo,
  useState,
  type KeyboardEvent,
} from "react";
import type { ManualGift, MiniappParticipantOption } from "@/types";
import { miniappGiftMutationErrorMessage } from "@/utils/miniappMyGifts";
import { matchesSearchQuery } from "@/utils/search";

interface MyGiftRecipientSelectProps {
  gift: ManualGift;
  participants: MiniappParticipantOption[];
  isSaving: boolean;
  onSave: (giftID: number, participantID: number | null) => Promise<void>;
}

function recipientLabel(participant: MiniappParticipantOption): string {
  const username = participant.username?.replace(/^@+/, "").trim();
  return username
    ? `${participant.display_name} (@${username})`
    : participant.display_name;
}

export default function MyGiftRecipientSelect({
  gift,
  participants,
  isSaving,
  onSave,
}: MyGiftRecipientSelectProps) {
  const [search, setSearch] = useState("");
  const [isPickerOpen, setIsPickerOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const [error, setError] = useState<string | null>(null);
  const listboxId = useId();
  const titleId = useId();

  const options = useMemo(() => {
    return participants.filter((participant) =>
      matchesSearchQuery(search, [
        participant.display_name,
        participant.username,
        participant.id,
      ])
    );
  }, [participants, search]);

  useEffect(() => {
    if (!search.trim()) {
      return;
    }
    console.debug("[FIX:recipient-search] miniapp filter completed", {
      gift_id: gift.id,
      result_count: options.length,
      has_username_prefix: search.trim().startsWith("@"),
    });
  }, [gift.id, options.length, search]);

  const saveRecipient = async (
    participantID: number | null,
    reopenPickerOnError: boolean,
  ) => {
    try {
      setError(null);
      await onSave(gift.id, participantID);
    } catch (saveError) {
      console.warn("[miniapp] manual recipient update failed", {
        giftId: gift.id,
        participantId: participantID,
        message: saveError instanceof Error ? saveError.message : "Unknown error",
      });
      setError(miniappGiftMutationErrorMessage(saveError));
      if (reopenPickerOnError) {
        setIsPickerOpen(true);
      }
    }
  };

  const selectRecipient = (participant: MiniappParticipantOption) => {
    setSearch(recipientLabel(participant));
    setIsPickerOpen(false);
    setHighlightedIndex(-1);
    void saveRecipient(participant.id, true);
  };

  const clearRecipient = () => {
    void saveRecipient(null, false);
  };

  const openPicker = () => {
    setError(null);
    setSearch("");
    setIsPickerOpen(true);
    setHighlightedIndex(-1);
  };

  const closePicker = () => {
    setIsPickerOpen(false);
    setHighlightedIndex(-1);
  };

  const handleInputChange = (value: string) => {
    setError(null);
    setSearch(value);
    setHighlightedIndex(-1);
  };

  const handleInputKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      closePicker();
      return;
    }

    if (options.length === 0) {
      return;
    }

    if (event.key === "ArrowDown") {
      event.preventDefault();
      setHighlightedIndex((current) =>
        Math.min(current + 1, options.length - 1)
      );
      return;
    }

    if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlightedIndex((current) => Math.max(current - 1, 0));
      return;
    }

    if (event.key === "Enter" && highlightedIndex >= 0) {
      event.preventDefault();
      selectRecipient(options[highlightedIndex]);
    }
  };

  return (
    <>
      {gift.recipient ? (
        <div className="mt-3">
          <p className="tg-muted text-sm leading-5">
            Получатель: <span className="tg-title font-semibold">{recipientLabel(gift.recipient)}</span>
          </p>
          <div className="mt-2 flex flex-wrap gap-2">
            <button
              type="button"
              onClick={openPicker}
              disabled={isSaving}
              className="tg-link-button inline-flex min-h-9 items-center justify-center rounded-lg border px-3 py-2 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-60"
            >
              Изменить
            </button>
            <button
              type="button"
              onClick={clearRecipient}
              disabled={isSaving}
              className="tg-link-button inline-flex min-h-9 items-center justify-center rounded-lg border px-3 py-2 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-60"
            >
              Отменить
            </button>
          </div>
        </div>
      ) : (
        <button
          type="button"
          onClick={openPicker}
          disabled={isSaving}
          className="tg-divider tg-title mt-3 h-10 w-full rounded-lg border bg-transparent px-3 text-left text-base disabled:cursor-not-allowed disabled:opacity-60"
        >
          Выберите получателя
        </button>
      )}
      {!isPickerOpen && error && <p className="tg-error mt-2 text-sm leading-5">{error}</p>}
      {isPickerOpen && (
        <section
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          className="tg-screen fixed inset-0 z-50 flex min-h-dvh flex-col px-3 pt-[calc(env(safe-area-inset-top)+0.75rem)] pb-[calc(env(safe-area-inset-bottom)+0.75rem)]"
        >
          <header className="flex items-center justify-between gap-3">
            <h2 id={titleId} className="tg-title text-lg font-semibold">
              {gift.recipient ? "Изменить получателя" : "Выберите получателя"}
            </h2>
            <button
              type="button"
              onClick={closePicker}
              className="tg-link-button rounded-lg border px-3 py-2 text-sm font-semibold"
            >
              Закрыть
            </button>
          </header>
          <input
            id={`recipient-search-${gift.id}`}
            type="text"
            role="combobox"
            autoFocus
            autoComplete="off"
            value={search}
            onChange={(event) => handleInputChange(event.target.value)}
            onKeyDown={handleInputKeyDown}
            placeholder="Поиск по имени или @нику"
            aria-label="Поиск получателя"
            aria-autocomplete="list"
            aria-controls={listboxId}
            aria-expanded
            aria-activedescendant={
              highlightedIndex >= 0
                ? `${listboxId}-option-${options[highlightedIndex]?.id}`
                : undefined
            }
            disabled={isSaving}
            className="tg-divider tg-title mt-4 h-11 w-full rounded-lg border bg-transparent px-3 text-base outline-none focus:border-[var(--tg-button-color)] disabled:cursor-not-allowed disabled:opacity-60"
          />
          {error && <p className="tg-error mt-2 text-sm leading-5">{error}</p>}
          <div
            id={listboxId}
            role="listbox"
            aria-label="Результаты поиска участников"
            className="tg-card tg-divider mt-3 min-h-0 flex-1 overflow-y-auto rounded-xl border p-1"
          >
            {options.length === 0 ? (
              <p className="tg-muted px-3 py-4 text-sm">
                Участники по этому запросу не найдены.
              </p>
            ) : (
              options.map((participant, index) => (
                <button
                  key={participant.id}
                  id={`${listboxId}-option-${participant.id}`}
                  type="button"
                  role="option"
                  aria-selected={false}
                  onMouseEnter={() => setHighlightedIndex(index)}
                  onClick={() => selectRecipient(participant)}
                  className={`block w-full rounded-lg px-3 py-3 text-left text-sm transition-colors ${
                    highlightedIndex === index
                      ? "bg-[var(--tg-secondary-bg-color)]"
                      : "hover:bg-[var(--tg-secondary-bg-color)]"
                  }`}
                >
                  <span className="tg-title block font-medium">
                    {participant.display_name}
                  </span>
                  {participant.username && (
                    <span className="tg-muted mt-0.5 block">
                      @{participant.username.replace(/^@+/, "")}
                    </span>
                  )}
                </button>
              ))
            )}
          </div>
        </section>
      )}
    </>
  );
}
