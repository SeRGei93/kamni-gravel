'use client';

import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from 'react';
import type { Participant } from '@/types';
import { formatManualRecipientSearchLabel } from '@/utils/manualGiftAssignment';
import { matchesSearchQuery } from '@/utils/search';

interface RecipientAutocompleteProps {
  giftID: number;
  participants: Participant[];
  selectedID: number | null;
  onSelect: (participantID: number | null) => void;
  disabled?: boolean;
}

const MAX_VISIBLE_OPTIONS = 8;

function participantLabel(participant: Participant): string {
  const displayName = [participant.first_name, participant.last_name]
    .filter(Boolean)
    .join(' ') || `Участник #${participant.id}`;
  return formatManualRecipientSearchLabel(displayName, participant.username);
}

export default function RecipientAutocomplete({
  giftID,
  participants,
  selectedID,
  onSelect,
  disabled = false,
}: RecipientAutocompleteProps) {
  const [search, setSearch] = useState('');
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [highlightedIndex, setHighlightedIndex] = useState(-1);
  const pickerRef = useRef<HTMLDivElement>(null);
  const listboxID = useId();

  const selectedParticipant = useMemo(
    () => participants.find((participant) => participant.id === selectedID),
    [participants, selectedID]
  );
  const matchedParticipants = useMemo(
    () => participants.filter((participant) =>
      matchesSearchQuery(search, [
        participant.first_name,
        participant.last_name,
        participant.username,
        participant.id,
        participant.user_id,
      ])
    ),
    [participants, search]
  );
  const visibleParticipants = useMemo(
    () => matchedParticipants.slice(0, MAX_VISIBLE_OPTIONS),
    [matchedParticipants]
  );
  const highlightedParticipant = visibleParticipants[highlightedIndex];
  const inputValue = isDropdownOpen
    ? search
    : selectedParticipant
      ? participantLabel(selectedParticipant)
      : search;

  useEffect(() => {
    if (!search.trim()) {
      return;
    }
    console.debug('[FIX:recipient-search] admin autocomplete completed', {
      gift_id: giftID,
      result_count: matchedParticipants.length,
      has_username_prefix: search.trim().startsWith('@'),
    });
  }, [giftID, matchedParticipants.length, search]);

  useEffect(() => {
    const handlePointerDown = (event: PointerEvent) => {
      if (!pickerRef.current?.contains(event.target as Node)) {
        setIsDropdownOpen(false);
        setHighlightedIndex(-1);
      }
    };

    document.addEventListener('pointerdown', handlePointerDown);
    return () => document.removeEventListener('pointerdown', handlePointerDown);
  }, []);

  const selectParticipant = (participant: Participant) => {
    onSelect(participant.id);
    setSearch('');
    setIsDropdownOpen(false);
    setHighlightedIndex(-1);
  };

  const handleFocus = () => {
    if (selectedParticipant) {
      setSearch('');
    }
    setIsDropdownOpen(true);
    setHighlightedIndex(-1);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Escape') {
      setIsDropdownOpen(false);
      setHighlightedIndex(-1);
      return;
    }

    if (!search.trim() || visibleParticipants.length === 0) {
      return;
    }

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setIsDropdownOpen(true);
      setHighlightedIndex((current) =>
        Math.min(current + 1, visibleParticipants.length - 1)
      );
      return;
    }

    if (event.key === 'ArrowUp') {
      event.preventDefault();
      setHighlightedIndex((current) => Math.max(current - 1, 0));
      return;
    }

    if (event.key === 'Enter' && isDropdownOpen && highlightedParticipant) {
      event.preventDefault();
      selectParticipant(highlightedParticipant);
    }
  };

  return (
    <div ref={pickerRef} className="relative">
      <input
        type="text"
        role="combobox"
        autoComplete="off"
        value={inputValue}
        onFocus={handleFocus}
        onChange={(event) => {
          setSearch(event.target.value);
          setIsDropdownOpen(true);
          setHighlightedIndex(-1);
        }}
        onKeyDown={handleKeyDown}
        placeholder="Поиск по имени или @нику"
        aria-label="Поиск получателя"
        aria-autocomplete="list"
        aria-controls={listboxID}
        aria-expanded={isDropdownOpen}
        aria-activedescendant={
          highlightedParticipant
            ? `${listboxID}-option-${highlightedParticipant.id}`
            : undefined
        }
        disabled={disabled}
        className="h-10 w-full rounded-lg border border-gray-300 bg-transparent px-3 text-sm text-gray-800 outline-none focus:border-brand-500 disabled:cursor-not-allowed disabled:opacity-60 dark:border-gray-700 dark:text-white/90"
      />
      {isDropdownOpen && (
        <div
          id={listboxID}
          role="listbox"
          aria-label="Результаты поиска участников"
          className="absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-lg border border-gray-200 bg-white p-1 shadow-lg dark:border-gray-700 dark:bg-gray-900"
        >
          {participants.length === 0 ? (
            <p className="px-2.5 py-2 text-xs text-warning-600 dark:text-warning-400">
              В этом событии пока нет доступных участников.
            </p>
          ) : !search.trim() ? (
            <p className="px-2.5 py-2 text-xs text-gray-500 dark:text-gray-400">
              Введите имя или @ник участника
            </p>
          ) : visibleParticipants.length === 0 ? (
            <p className="px-2.5 py-2 text-xs text-gray-500 dark:text-gray-400">
              Участники по этому запросу не найдены.
            </p>
          ) : (
            visibleParticipants.map((participant, index) => (
              <button
                key={participant.id}
                id={`${listboxID}-option-${participant.id}`}
                type="button"
                role="option"
                aria-selected={selectedID === participant.id}
                onMouseEnter={() => setHighlightedIndex(index)}
                onClick={() => selectParticipant(participant)}
                className={`block w-full rounded-md px-2.5 py-2 text-left text-sm transition-colors ${
                  highlightedIndex === index || selectedID === participant.id
                    ? 'bg-brand-50 text-gray-900 dark:bg-brand-500/10 dark:text-white'
                    : 'text-gray-800 hover:bg-gray-50 dark:text-white/90 dark:hover:bg-white/[0.04]'
                }`}
              >
                <span className="block font-medium">
                  {[participant.first_name, participant.last_name]
                    .filter(Boolean)
                    .join(' ') || `Участник #${participant.id}`}
                </span>
                {participant.username && (
                  <span className="mt-0.5 block text-xs text-gray-500 dark:text-gray-400">
                    @{participant.username.replace(/^@+/, '')}
                  </span>
                )}
              </button>
            ))
          )}
        </div>
      )}
      {selectedID !== null && (
        <button
          type="button"
          onClick={() => onSelect(null)}
          disabled={disabled}
          className="mt-2 text-xs text-gray-500 underline underline-offset-2 disabled:cursor-not-allowed disabled:opacity-60 dark:text-gray-400"
        >
          Снять выбор
        </button>
      )}
    </div>
  );
}
