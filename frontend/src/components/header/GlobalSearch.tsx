'use client';

import { useState, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { participantsApi } from '@/api/participants';
import { eventsApi } from '@/api/events';
import type { Participant, Event } from '@/types';

export default function GlobalSearch() {
  const [searchQuery, setSearchQuery] = useState('');
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [results, setResults] = useState<Participant[]>([]);
  const [activeEvent, setActiveEvent] = useState<Event | null>(null);
  
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const router = useRouter();

  // Загрузка активного события при монтировании
  useEffect(() => {
    loadActiveEvent();
  }, []);

  // Обработка горячих клавиш ⌘K / Ctrl+K
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault();
        inputRef.current?.focus();
      }
      // Закрытие по Escape
      if (event.key === 'Escape') {
        setIsOpen(false);
        setSearchQuery('');
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, []);

  // Закрытие при клике вне компонента
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node) &&
        !inputRef.current?.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, []);

  // Поиск участников при изменении запроса
  useEffect(() => {
    if (searchQuery.length >= 2 && activeEvent) {
      searchParticipants();
    } else {
      setResults([]);
      setIsOpen(false);
    }
  }, [searchQuery, activeEvent]);

  const loadActiveEvent = async () => {
    try {
      const response = await eventsApi.getAll();
      const active = response.events.find((e) => e.active);
      setActiveEvent(active || response.events[0] || null);
    } catch (err) {
      console.error('Failed to load active event:', err);
    }
  };

  const searchParticipants = async () => {
    if (!activeEvent) return;

    try {
      setIsLoading(true);
      const response = await participantsApi.getByEvent(activeEvent.id, {});
      
      // Фильтрация на клиенте
      const query = searchQuery.toLowerCase();
      const filtered = response.participants.filter((p) => {
        return (
          p.username?.toLowerCase().includes(query) ||
          p.first_name?.toLowerCase().includes(query) ||
          p.last_name?.toLowerCase().includes(query) ||
          String(p.user_id).includes(query)
        );
      });

      setResults(filtered.slice(0, 10)); // Ограничиваем 10 результатами
      setIsOpen(filtered.length > 0);
    } catch (err) {
      console.error('Failed to search participants:', err);
      setResults([]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSelectParticipant = (participantId: number) => {
    router.push(`/participants/${participantId}`);
    setIsOpen(false);
    setSearchQuery('');
  };

  const getParticipantDisplayName = (p: Participant) => {
    const parts = [];
    if (p.first_name) parts.push(p.first_name);
    if (p.last_name) parts.push(p.last_name);
    if (parts.length === 0 && p.username) parts.push(p.username);
    return parts.join(' ') || `ID: ${p.user_id}`;
  };

  const getBikeTypeLabel = (bikeType: string) => {
    const labels: Record<string, string> = {
      gravel: 'Гравийник',
      mtb: 'МТБ',
      road: 'Шоссе',
      single_speed: 'Фикс',
      tandem: 'Тандем',
    };
    return labels[bikeType] || bikeType;
  };

  const getGenderLabel = (gender: string) => {
    return gender === 'male' ? 'М' : 'Ж';
  };

  return (
    <div className="relative">
      <form onSubmit={(e) => e.preventDefault()}>
        <div className="relative">
          <span className="absolute -translate-y-1/2 left-4 top-1/2 pointer-events-none">
            <svg
              className="fill-gray-500 dark:fill-gray-400"
              width="20"
              height="20"
              viewBox="0 0 20 20"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                fillRule="evenodd"
                clipRule="evenodd"
                d="M3.04175 9.37363C3.04175 5.87693 5.87711 3.04199 9.37508 3.04199C12.8731 3.04199 15.7084 5.87693 15.7084 9.37363C15.7084 12.8703 12.8731 15.7053 9.37508 15.7053C5.87711 15.7053 3.04175 12.8703 3.04175 9.37363ZM9.37508 1.54199C5.04902 1.54199 1.54175 5.04817 1.54175 9.37363C1.54175 13.6991 5.04902 17.2053 9.37508 17.2053C11.2674 17.2053 13.003 16.5344 14.357 15.4176L17.177 18.238C17.4699 18.5309 17.9448 18.5309 18.2377 18.238C18.5306 17.9451 18.5306 17.4703 18.2377 17.1774L15.418 14.3573C16.5365 13.0033 17.2084 11.2669 17.2084 9.37363C17.2084 5.04817 13.7011 1.54199 9.37508 1.54199Z"
                fill=""
              />
            </svg>
          </span>
          <input
            ref={inputRef}
            type="text"
            placeholder="Поиск участника..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onFocus={() => {
              if (searchQuery.length >= 2 && results.length > 0) {
                setIsOpen(true);
              }
            }}
            className="dark:bg-dark-900 h-11 w-full rounded-lg border border-gray-200 bg-transparent py-2.5 pl-12 pr-14 text-sm text-gray-800 shadow-theme-xs placeholder:text-gray-400 focus:border-brand-300 focus:outline-hidden focus:ring-3 focus:ring-brand-500/10 dark:border-gray-800 dark:bg-gray-900 dark:bg-white/[0.03] dark:text-white/90 dark:placeholder:text-white/30 dark:focus:border-brand-800 xl:w-[430px]"
          />

          <button
            type="button"
            className="absolute right-2.5 top-1/2 inline-flex -translate-y-1/2 items-center gap-0.5 rounded-lg border border-gray-200 bg-gray-50 px-[7px] py-[4.5px] text-xs -tracking-[0.2px] text-gray-500 dark:border-gray-800 dark:bg-white/[0.03] dark:text-gray-400"
          >
            <span> ⌘ </span>
            <span> K </span>
          </button>
        </div>
      </form>

      {/* Выпадающий список результатов */}
      {isOpen && (
        <div
          ref={dropdownRef}
          className="absolute top-full left-0 mt-2 w-full xl:w-[430px] rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-800 dark:bg-gray-900 z-50 max-h-[400px] overflow-y-auto"
        >
          {isLoading ? (
            <div className="p-4 text-center text-sm text-gray-600 dark:text-gray-400">
              Поиск...
            </div>
          ) : results.length === 0 ? (
            <div className="p-4 text-center text-sm text-gray-600 dark:text-gray-400">
              Ничего не найдено
            </div>
          ) : (
            <div className="py-2">
              {results.map((participant) => (
                <button
                  key={participant.id}
                  onClick={() => handleSelectParticipant(participant.id)}
                  className="w-full px-4 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors border-b border-gray-100 dark:border-gray-800 last:border-b-0"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <div className="font-medium text-gray-900 dark:text-white">
                        {getParticipantDisplayName(participant)}
                      </div>
                      {participant.username && (
                        <div className="text-sm text-gray-500 dark:text-gray-400">
                          @{participant.username}
                        </div>
                      )}
                    </div>
                    <div className="flex items-center gap-2 text-xs">
                      <span className="px-2 py-1 rounded bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300">
                        {getGenderLabel(participant.gender)}
                      </span>
                      <span className="px-2 py-1 rounded bg-brand-50 dark:bg-brand-900/20 text-brand-700 dark:text-brand-400">
                        {getBikeTypeLabel(participant.bike_type)}
                      </span>
                      {participant.is_finished && (
                        <span className="px-2 py-1 rounded bg-success-50 dark:bg-success-900/20 text-success-700 dark:text-success-400">
                          ✓ Финиш
                        </span>
                      )}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
