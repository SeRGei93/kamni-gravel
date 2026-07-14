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
import type { ManualGift, MiniappParticipantOption } from "@/types";

export interface MyGiftsSnapshot {
  gifts: ManualGift[];
  participants: MiniappParticipantOption[];
}

interface MiniappMyGiftsContextValue {
  snapshot: MyGiftsSnapshot | null;
  setSnapshot: (snapshot: MyGiftsSnapshot) => void;
  updateSnapshot: (updater: (snapshot: MyGiftsSnapshot) => MyGiftsSnapshot) => void;
  // Позиция прокрутки вкладки «Призы от меня» при переключении меню.
  scrollYRef: MutableRefObject<number>;
}

const MiniappMyGiftsContext =
  createContext<MiniappMyGiftsContextValue | null>(null);

export function MiniappMyGiftsProvider({ children }: { children: ReactNode }) {
  const [snapshot, setSnapshot] = useState<MyGiftsSnapshot | null>(null);
  const scrollYRef = useRef(0);

  const setCachedSnapshot = useCallback((value: MyGiftsSnapshot) => {
    setSnapshot(value);
  }, []);

  const updateCachedSnapshot = useCallback((updater: (snapshot: MyGiftsSnapshot) => MyGiftsSnapshot) => {
    setSnapshot((current) => (current ? updater(current) : current));
  }, []);

  const value = useMemo<MiniappMyGiftsContextValue>(
    () => ({ snapshot, setSnapshot: setCachedSnapshot, updateSnapshot: updateCachedSnapshot, scrollYRef }),
    [snapshot, setCachedSnapshot, updateCachedSnapshot]
  );

  return (
    <MiniappMyGiftsContext.Provider value={value}>
      {children}
    </MiniappMyGiftsContext.Provider>
  );
}

export function useMiniappMyGifts(): MiniappMyGiftsContextValue {
  const context = useContext(MiniappMyGiftsContext);
  if (!context) {
    throw new Error("useMiniappMyGifts must be used within MiniappMyGiftsProvider");
  }
  return context;
}
