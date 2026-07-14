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
import type { MiniappGenderFilter } from "@/components/miniapp/GiftFilters";
import type { BikeTypeFilter, Gift } from "@/types";

// Состояние каталога Mini App живёт в провайдере, который рендерится в layout
// группы (miniapp). Layout НЕ размонтируется при переходе на карточку приза и
// обратно, поэтому выбранные фильтры, загруженная сессия и уже подгруженные
// списки призов сохраняются между навигациями (раньше всё сбрасывалось, т.к.
// состояние держалось в useState самой страницы каталога).

const DEFAULT_GENDER: MiniappGenderFilter = "all_genders";
const DEFAULT_BIKE_TYPE: BikeTypeFilter = "all";

// Фильтры дополнительно кладём в sessionStorage, чтобы они переживали и полную
// перезагрузку Mini App (повторное открытие из бота), а не только навигацию.
const GENDER_STORAGE_KEY = "miniapp:gifts:gender";
const BIKE_TYPE_STORAGE_KEY = "miniapp:gifts:bike_type";

export interface CatalogSnapshot {
  gifts: Gift[];
  participantCount?: number;
}

interface MiniappCatalogContextValue {
  gender: MiniappGenderFilter;
  setGender: (value: MiniappGenderFilter) => void;
  bikeType: BikeTypeFilter;
  setBikeType: (value: BikeTypeFilter) => void;
  searchQuery: string;
  setSearchQuery: (value: string) => void;
  // Позиция прокрутки каталога, чтобы восстановить её при возврате с карточки.
  scrollYRef: MutableRefObject<number>;
  getCatalogSnapshot: (key: string) => CatalogSnapshot | undefined;
  setCatalogSnapshot: (key: string, snapshot: CatalogSnapshot) => void;
}

const MiniappCatalogContext = createContext<MiniappCatalogContextValue | null>(null);

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

export function MiniappCatalogProvider({ children }: { children: ReactNode }) {
  const [gender, setGenderState] = useState<MiniappGenderFilter>(() =>
    readStoredFilter(GENDER_STORAGE_KEY, DEFAULT_GENDER)
  );
  const [bikeType, setBikeTypeState] = useState<BikeTypeFilter>(() =>
    readStoredFilter(BIKE_TYPE_STORAGE_KEY, DEFAULT_BIKE_TYPE)
  );
  const [searchQuery, setSearchQuery] = useState("");
  const scrollYRef = useRef(0);
  // Кеш списков призов по ключу фильтра держим в ref: его изменение не должно
  // вызывать ререндер провайдера, читаем синхронно при маунте страницы.
  const catalogCacheRef = useRef<Map<string, CatalogSnapshot>>(new Map());

  const setGender = useCallback((value: MiniappGenderFilter) => {
    setGenderState(value);
    writeStoredFilter(GENDER_STORAGE_KEY, value);
  }, []);

  const setBikeType = useCallback((value: BikeTypeFilter) => {
    setBikeTypeState(value);
    writeStoredFilter(BIKE_TYPE_STORAGE_KEY, value);
  }, []);

  const getCatalogSnapshot = useCallback(
    (key: string) => catalogCacheRef.current.get(key),
    []
  );

  const setCatalogSnapshot = useCallback((key: string, snapshot: CatalogSnapshot) => {
    catalogCacheRef.current.set(key, snapshot);
  }, []);

  const value = useMemo<MiniappCatalogContextValue>(
    () => ({
      gender,
      setGender,
      bikeType,
      setBikeType,
      searchQuery,
      setSearchQuery,
      scrollYRef,
      getCatalogSnapshot,
      setCatalogSnapshot,
    }),
    [
      gender,
      setGender,
      bikeType,
      setBikeType,
      searchQuery,
      getCatalogSnapshot,
      setCatalogSnapshot,
    ]
  );

  return (
    <MiniappCatalogContext.Provider value={value}>
      {children}
    </MiniappCatalogContext.Provider>
  );
}

export function useMiniappCatalog(): MiniappCatalogContextValue {
  const context = useContext(MiniappCatalogContext);
  if (!context) {
    throw new Error("useMiniappCatalog must be used within MiniappCatalogProvider");
  }
  return context;
}
