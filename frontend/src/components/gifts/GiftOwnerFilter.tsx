'use client';

import {
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
} from 'react';
import { ChevronDownIcon } from '@/icons';
import type { Participant } from '@/types';
import { matchesSearchQuery } from '@/utils/search';

interface GiftOwnerFilterProps {
  owners: Participant[];
  value?: number;
  onChange: (ownerUserID: number | undefined) => void;
}

function giftOwnerLabel(participant: Participant): string {
  const name = `${participant.first_name} ${participant.last_name}`.trim();
  if (name && participant.username) {
    return `${name} (@${participant.username.replace(/^@+/, '')})`;
  }

  return name || participant.username || `Участник #${participant.user_id}`;
}

export default function GiftOwnerFilter({
  owners,
  value,
  onChange,
}: GiftOwnerFilterProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const pickerRef = useRef<HTMLDivElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);
  const listboxID = useId();

  const sortedOwners = useMemo(
    () =>
      owners
        .slice()
        .sort((left, right) =>
          giftOwnerLabel(left).localeCompare(giftOwnerLabel(right), 'ru')
        ),
    [owners]
  );
  const selectedOwner = useMemo(
    () => sortedOwners.find((owner) => owner.user_id === value),
    [sortedOwners, value]
  );
  const matchedOwners = useMemo(
    () =>
      sortedOwners.filter((owner) =>
        matchesSearchQuery(search, [
          owner.first_name,
          owner.last_name,
          owner.username,
        ])
      ),
    [search, sortedOwners]
  );
  const selectedLabel = selectedOwner
    ? giftOwnerLabel(selectedOwner)
    : value
      ? `Участник #${value}`
      : 'Все авторы';

  useEffect(() => {
    const handlePointerDown = (event: PointerEvent) => {
      if (!pickerRef.current?.contains(event.target as Node)) {
        setIsOpen(false);
        setSearch('');
      }
    };

    document.addEventListener('pointerdown', handlePointerDown);
    return () => document.removeEventListener('pointerdown', handlePointerDown);
  }, []);

  useEffect(() => {
    if (isOpen) {
      searchInputRef.current?.focus();
    }
  }, [isOpen]);

  const openPicker = () => {
    setSearch('');
    setIsOpen(true);
  };

  const selectOwner = (ownerUserID: number | undefined) => {
    onChange(ownerUserID);
    setSearch('');
    setIsOpen(false);
  };

  const handleSearchKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Escape') {
      event.preventDefault();
      setIsOpen(false);
      setSearch('');
    }
  };

  return (
    <div ref={pickerRef} className="relative">
      <button
        type="button"
        onClick={() => (isOpen ? setIsOpen(false) : openPicker())}
        aria-haspopup="listbox"
        aria-expanded={isOpen}
        aria-controls={listboxID}
        className="flex h-11 w-full items-center justify-between gap-3 rounded-lg border border-gray-300 bg-transparent px-4 py-2.5 text-left text-sm text-gray-800 shadow-theme-xs outline-none transition focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
      >
        <span className="truncate">{selectedLabel}</span>
        <ChevronDownIcon
          aria-hidden="true"
          className={`size-4 shrink-0 transition-transform ${isOpen ? 'rotate-180' : ''}`}
        />
      </button>

      {isOpen && (
        <div className="absolute left-0 z-40 mt-1 w-full overflow-hidden rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-900">
          <div className="border-b border-gray-100 p-2 dark:border-gray-800">
            <input
              ref={searchInputRef}
              type="search"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              onKeyDown={handleSearchKeyDown}
              placeholder="Поиск участника"
              aria-label="Поиск автора приза"
              className="h-10 w-full rounded-md border border-gray-300 bg-transparent px-3 text-sm text-gray-800 outline-none focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 dark:border-gray-700 dark:text-white/90 dark:focus:border-brand-800"
            />
          </div>
          <div
            id={listboxID}
            role="listbox"
            aria-label="Авторы призов"
            className="max-h-72 overflow-y-auto p-1"
          >
            <button
              type="button"
              role="option"
              aria-selected={!value}
              onClick={() => selectOwner(undefined)}
              className={`block w-full rounded-md px-3 py-2 text-left text-sm transition-colors ${
                !value
                  ? 'bg-brand-50 text-gray-900 dark:bg-brand-500/10 dark:text-white'
                  : 'text-gray-800 hover:bg-gray-50 dark:text-white/90 dark:hover:bg-white/[0.04]'
              }`}
            >
              Все авторы
            </button>

            {matchedOwners.map((owner) => (
              <button
                key={owner.user_id}
                type="button"
                role="option"
                aria-selected={owner.user_id === value}
                onClick={() => selectOwner(owner.user_id)}
                className={`block w-full rounded-md px-3 py-2 text-left text-sm transition-colors ${
                  owner.user_id === value
                    ? 'bg-brand-50 text-gray-900 dark:bg-brand-500/10 dark:text-white'
                    : 'text-gray-800 hover:bg-gray-50 dark:text-white/90 dark:hover:bg-white/[0.04]'
                }`}
              >
                <span className="block truncate">{giftOwnerLabel(owner)}</span>
              </button>
            ))}

            {matchedOwners.length === 0 && (
              <p className="px-3 py-2 text-sm text-gray-500 dark:text-gray-400">
                {owners.length === 0
                  ? 'В этом событии пока нет авторов призов.'
                  : 'Участники по этому запросу не найдены.'}
              </p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
