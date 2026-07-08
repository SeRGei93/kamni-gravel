"use client";

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type MutableRefObject,
  type ReactNode,
} from "react";
import type {
  BikeTypeFilter,
  MiniappLeaderboardEntry,
  MiniappSessionResponse,
} from "@/types";

// Состояние лидерборда Mini App живёт в провайдере на уровне layout группы
// (miniapp), поэтому фильтры, загруженная сессия и уже подгруженный список
// участников сохраняются при переходе на детальную карточку и обратно — без
// повторной загрузки и без потери позиции прокрутки. Зеркалит подход
// MiniappCatalogContext (каталог призов).

export type LeaderboardGenderFilter = "all" | "male" | "female";

const DEFAULT_GENDER: LeaderboardGenderFilter = "all";
const DEFAULT_BIKE_TYPE: BikeTypeFilter = "all";

const GENDER_STORAGE_KEY = "miniapp:leaderboard:gender";
const BIKE_TYPE_STORAGE_KEY = "miniapp:leaderboard:bike_type";

interface MiniappLeaderboardContextValue {
  gender: LeaderboardGenderFilter;
  setGender: (value: LeaderboardGenderFilter) => void;
  bikeType: BikeTypeFilter;
  setBikeType: (value: BikeTypeFilter) => void;
  session: MiniappSessionResponse | null;
  setSession: (value: MiniappSessionResponse | null) => void;
  entries: MiniappLeaderboardEntry[] | null;
  setEntries: (value: MiniappLeaderboardEntry[]) => void;
  getEntry: (id: number) => MiniappLeaderboardEntry | undefined;
  // Позиция прокрутки списка, чтобы восстановить её при возврате с карточки.
  scrollYRef: MutableRefObject<number>;
}

const MiniappLeaderboardContext =
  createContext<MiniappLeaderboardContextValue | null>(null);

function readStoredFilter<T extends string>(key: string, fallback: T): T {
  if (typeof window === "undefined") {
    return fallback;
  }
  try {
    const stored = window.sessionStorage.getItem(key);
    return (stored as T) || fallback;
  } catch {
    return fallback;
  }
}

function writeStoredFilter(key: string, value: string): void {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.sessionStorage.setItem(key, value);
  } catch {
    // sessionStorage может быть недоступен в некоторых webview — молча игнорируем.
  }
}

export function MiniappLeaderboardProvider({ children }: { children: ReactNode }) {
  const [gender, setGenderState] = useState<LeaderboardGenderFilter>(() =>
    readStoredFilter(GENDER_STORAGE_KEY, DEFAULT_GENDER)
  );
  const [bikeType, setBikeTypeState] = useState<BikeTypeFilter>(() =>
    readStoredFilter(BIKE_TYPE_STORAGE_KEY, DEFAULT_BIKE_TYPE)
  );
  const [session, setSession] = useState<MiniappSessionResponse | null>(null);
  const [entries, setEntriesState] = useState<MiniappLeaderboardEntry[] | null>(null);
  const scrollYRef = useRef(0);

  const setGender = useCallback((value: LeaderboardGenderFilter) => {
    setGenderState(value);
    writeStoredFilter(GENDER_STORAGE_KEY, value);
  }, []);

  const setBikeType = useCallback((value: BikeTypeFilter) => {
    setBikeTypeState(value);
    writeStoredFilter(BIKE_TYPE_STORAGE_KEY, value);
  }, []);

  const setEntries = useCallback((value: MiniappLeaderboardEntry[]) => {
    setEntriesState(value);
  }, []);

  const getEntry = useCallback(
    (id: number) => entries?.find((entry) => entry.id === id),
    [entries]
  );

  const value = useMemo<MiniappLeaderboardContextValue>(
    () => ({
      gender,
      setGender,
      bikeType,
      setBikeType,
      session,
      setSession,
      entries,
      setEntries,
      getEntry,
      scrollYRef,
    }),
    [gender, setGender, bikeType, setBikeType, session, entries, setEntries, getEntry]
  );

  return (
    <MiniappLeaderboardContext.Provider value={value}>
      {children}
    </MiniappLeaderboardContext.Provider>
  );
}

export function useMiniappLeaderboard(): MiniappLeaderboardContextValue {
  const context = useContext(MiniappLeaderboardContext);
  if (!context) {
    throw new Error(
      "useMiniappLeaderboard must be used within MiniappLeaderboardProvider"
    );
  }
  return context;
}
