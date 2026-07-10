"use client";

import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { miniappApi } from "@/api/miniapp";
import type { MiniappSessionResponse } from "@/types";
import {
  expandTelegramWebApp,
  isTelegramWebAppAvailable,
  readyTelegramWebApp,
  waitForTelegramInitData,
} from "@/utils/telegramWebApp";

interface MiniappSessionContextValue {
  session: MiniappSessionResponse | null;
  isLoading: boolean;
  error: string | null;
}

const MiniappSessionContext = createContext<MiniappSessionContextValue | null>(null);

export function MiniappSessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<MiniappSessionResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let ignore = false;

    async function loadSession() {
      try {
        await waitForTelegramInitData();
        if (isTelegramWebAppAvailable()) {
          readyTelegramWebApp();
          expandTelegramWebApp();
        }

        const nextSession = await miniappApi.getSession();
        if (!ignore) {
          console.debug("[FIX] Mini App session loaded", {
            eventId: nextSession.event.id,
            hasMyResult: nextSession.my_result_participant_id !== undefined,
          });
          setSession(nextSession);
          setError(null);
        }
      } catch (loadError) {
        console.warn("[FIX] Mini App session load failed", {
          message: loadError instanceof Error ? loadError.message : "Unknown error",
        });
        if (!ignore) {
          setError("Не удалось загрузить сессию Mini App");
        }
      } finally {
        if (!ignore) {
          setIsLoading(false);
        }
      }
    }

    void loadSession();

    return () => {
      ignore = true;
    };
  }, []);

  return (
    <MiniappSessionContext.Provider value={{ session, isLoading, error }}>
      {children}
    </MiniappSessionContext.Provider>
  );
}

export function useMiniappSession(): MiniappSessionContextValue {
  const context = useContext(MiniappSessionContext);
  if (!context) {
    throw new Error("useMiniappSession must be used within MiniappSessionProvider");
  }
  return context;
}
