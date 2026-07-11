"use client";

import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import type { ManualGift, MiniappParticipantOption } from "@/types";
import {
  isRecipientSelectionChanged,
  miniappGiftMutationErrorMessage,
} from "@/utils/miniappMyGifts";
import { matchesSearchQuery } from "@/utils/search";

interface MyGiftRecipientSelectProps {
  gift: ManualGift;
  participants: MiniappParticipantOption[];
  isSaving: boolean;
  onSave: (giftID: number, participantID: number | null) => Promise<void>;
}

const MAX_VISIBLE_OPTIONS = 8;

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
  const [selectedID, setSelectedID] = useState<number | null>(
    gift.recipient?.id ?? null
  );
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const [error, setError] = useState<string | null>(null);
  const pickerRef = useRef<HTMLDivElement>(null);
  const listboxId = useId();

  const selectedParticipant = useMemo(
    () => participants.find((participant) => participant.id === selectedID),
    [participants, selectedID]
  );

  const options = useMemo(() => {
    return participants.filter((participant) =>
      matchesSearchQuery(search, [
        participant.display_name,
        participant.username,
        participant.id,
      ])
    );
  }, [participants, search]);

  const visibleOptions = useMemo(
    () => options.slice(0, MAX_VISIBLE_OPTIONS),
    [options]
  );

  const inputValue = isDropdownOpen
    ? search
    : selectedParticipant
      ? recipientLabel(selectedParticipant)
      : search;

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

  useEffect(() => {
    const handlePointerDown = (event: PointerEvent) => {
      if (!pickerRef.current?.contains(event.target as Node)) {
        setIsDropdownOpen(false);
        setHighlightedIndex(-1);
      }
    };

    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, []);

  const selectRecipient = (participant: MiniappParticipantOption) => {
    setSelectedID(participant.id);
    setSearch("");
    setIsDropdownOpen(false);
    setHighlightedIndex(-1);
  };

  const handleInputFocus = () => {
    if (selectedParticipant) {
      setSearch("");
    }
    setIsDropdownOpen(true);
    setHighlightedIndex(-1);
  };

  const handleInputChange = (value: string) => {
    setSearch(value);
    setIsDropdownOpen(true);
    setHighlightedIndex(-1);
  };

  const handleInputKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      setIsDropdownOpen(false);
      setHighlightedIndex(-1);
      return;
    }

    if (!search.trim() || visibleOptions.length === 0) {
      return;
    }

    if (event.key === "ArrowDown") {
      event.preventDefault();
      setIsDropdownOpen(true);
      setHighlightedIndex((current) =>
        Math.min(current + 1, visibleOptions.length - 1)
      );
      return;
    }

    if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlightedIndex((current) => Math.max(current - 1, 0));
      return;
    }

    if (event.key === "Enter" && isDropdownOpen && highlightedIndex >= 0) {
      event.preventDefault();
      selectRecipient(visibleOptions[highlightedIndex]);
    }
  };

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
      <div ref={pickerRef} className="relative mt-2">
        <input
          id={`recipient-search-${gift.id}`}
          type="text"
          role="combobox"
          autoComplete="off"
          value={inputValue}
          onFocus={handleInputFocus}
          onChange={(event) => handleInputChange(event.target.value)}
          onKeyDown={handleInputKeyDown}
          placeholder="Поиск по имени или @нику"
          aria-autocomplete="list"
          aria-controls={listboxId}
          aria-expanded={isDropdownOpen}
          aria-activedescendant={
            highlightedIndex >= 0
              ? `${listboxId}-option-${visibleOptions[highlightedIndex]?.id}`
              : undefined
          }
          disabled={isSaving}
          className="tg-divider tg-title h-10 w-full rounded-lg border bg-transparent px-3 text-sm outline-none focus:border-[var(--tg-button-color)] disabled:cursor-not-allowed disabled:opacity-60"
        />
        {isDropdownOpen && (
          <div
            id={listboxId}
            role="listbox"
            aria-label="Результаты поиска участников"
            className="tg-card tg-divider absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border p-1 shadow-lg"
          >
            {!search.trim() ? (
              <p className="tg-muted px-2.5 py-2 text-xs">
                Введите имя или @ник участника
              </p>
            ) : visibleOptions.length === 0 ? (
              <p className="tg-muted px-2.5 py-2 text-xs">
                Участники по этому запросу не найдены.
              </p>
            ) : (
              visibleOptions.map((participant, index) => (
                <button
                  key={participant.id}
                  id={`${listboxId}-option-${participant.id}`}
                  type="button"
                  role="option"
                  aria-selected={selectedID === participant.id}
                  onMouseEnter={() => setHighlightedIndex(index)}
                  onClick={() => selectRecipient(participant)}
                  className={`block w-full rounded-md px-2.5 py-2 text-left text-xs transition-colors ${
                    highlightedIndex === index || selectedID === participant.id
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
        )}
      </div>
      {selectedID !== null && (
        <button
          type="button"
          onClick={() => setSelectedID(null)}
          disabled={isSaving}
          className="tg-muted mt-2 text-xs underline underline-offset-2 disabled:cursor-not-allowed disabled:opacity-60"
        >
          Снять выбор
        </button>
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
