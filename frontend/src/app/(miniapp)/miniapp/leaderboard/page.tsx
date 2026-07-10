"use client";

import { useEffect, useMemo, useState } from "react";
import { miniappApi } from "@/api/miniapp";
import LeaderboardEmptyState from "@/components/miniapp/LeaderboardEmptyState";
import LeaderboardFilters from "@/components/miniapp/LeaderboardFilters";
import LeaderboardTable from "@/components/miniapp/LeaderboardTable";
import { useMiniappLeaderboard } from "@/components/miniapp/MiniappLeaderboardContext";
import { useMiniappSession } from "@/components/miniapp/MiniappSessionContext";
import MiniappSpinner from "@/components/miniapp/MiniappSpinner";
import { useIsomorphicLayoutEffect } from "@/hooks/useIsomorphicLayoutEffect";
import { rankAndFilterLeaderboard } from "@/utils/leaderboard";
import {
  expandTelegramWebApp,
  isTelegramWebAppAvailable,
  readyTelegramWebApp,
} from "@/utils/telegramWebApp";

export default function MiniappLeaderboardPage() {
  const {
    gender,
    setGender,
    bikeType,
    setBikeType,
    entries,
    setEntries,
    scrollYRef,
  } = useMiniappLeaderboard();
  const { session, isLoading: isSessionLoading, error: sessionError } = useMiniappSession();

  const [isListLoading, setIsListLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isTelegramWebAppAvailable()) {
      readyTelegramWebApp();
      expandTelegramWebApp();
    }
  }, []);

  useEffect(() => {
    if (!session) {
      return;
    }

    let ignore = false;

    async function loadLeaderboard() {
      // Показываем кеш сразу (мгновенный возврат с карточки), затем обновляем.
      setIsListLoading(true);

      try {
        const data = await miniappApi.getLeaderboard();
        if (!ignore) {
          setEntries(data.participants);
          setError(null);
        }
      } catch (loadError) {
        console.warn("[miniapp] Leaderboard load failed", {
          message: loadError instanceof Error ? loadError.message : "Unknown error",
        });
        // Если кеш есть — оставляем его показанным, ошибку не выводим.
        if (!ignore && entries === null) {
          setError("Не удалось загрузить лидерборд");
        }
      } finally {
        if (!ignore) {
          setIsListLoading(false);
        }
      }
    }

    loadLeaderboard();

    return () => {
      ignore = true;
    };
    // entries намеренно не в зависимостях: список грузим один раз на сессию,
    // фильтрация дальше идёт на клиенте без повторных запросов.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session, setEntries]);

  const rows = useMemo(
    () => rankAndFilterLeaderboard(entries ?? [], gender, bikeType),
    [entries, gender, bikeType]
  );

  // Восстанавливаем позицию прокрутки при возврате с карточки и сохраняем её при
  // уходе. Данные списка уже в кеше контекста, поэтому таблица отрисована
  // синхронно на маунте — высоты страницы хватает для точного восстановления.
  useIsomorphicLayoutEffect(() => {
    window.scrollTo(0, scrollYRef.current);
    return () => {
      scrollYRef.current = window.scrollY;
    };
  }, [scrollYRef]);

  if (isSessionLoading) {
    return <MiniappShellState title="Лидерборд" text="Загружаем активное событие" />;
  }

  if (sessionError || error) {
    return (
      <MiniappShellState
        title="Лидерборд недоступен"
        text={sessionError ?? error ?? "Не удалось открыть лидерборд"}
        tone="error"
      />
    );
  }

  if (!session) {
    return (
      <MiniappShellState
        title="Нет активного события"
        text="Лидерборд появится после открытия события"
      />
    );
  }

  return (
    <main className="tg-screen min-h-screen">
      <section className="tg-topbar sticky top-0 z-10 border-b px-3 py-2 backdrop-blur">
        <div className="mx-auto flex w-full max-w-md flex-col gap-2">
          <LeaderboardFilters
            gender={gender}
            bikeType={bikeType}
            onGenderChange={setGender}
            onBikeTypeChange={setBikeType}
          />
          {isListLoading && (
            <div className="tg-muted flex items-center justify-center gap-2 text-[11px] font-medium">
              <MiniappSpinner size={12} />
              <span>Обновляем лидерборд…</span>
            </div>
          )}
        </div>
      </section>

      <section className="mx-auto flex w-full max-w-md flex-col gap-3 px-3 py-3">
        {isListLoading && entries === null ? (
          <MiniappLeaderboardLoading />
        ) : rows.length > 0 ? (
          <LeaderboardTable rows={rows} isLoading={isListLoading} />
        ) : (
          <LeaderboardEmptyState />
        )}
      </section>
    </main>
  );
}

function MiniappShellState({
  title,
  text,
  tone = "default",
}: {
  title: string;
  text: string;
  tone?: "default" | "error";
}) {
  const isError = tone === "error";

  return (
    <main className="tg-screen flex min-h-[60vh] items-center justify-center px-5 py-8">
      <section className="tg-card w-full max-w-sm rounded-xl border p-5">
        <div
          className={`mb-4 h-2 w-16 rounded-full ${
            isError ? "tg-error-bar" : "tg-accent-bar"
          }`}
        />
        <h1 className="tg-title text-xl font-semibold leading-7">{title}</h1>
        <p className="tg-muted mt-2 text-sm leading-5">{text}</p>
      </section>
    </main>
  );
}

function MiniappLeaderboardLoading() {
  return (
    <div className="tg-card overflow-hidden rounded-xl border">
      <div className="tg-topbar grid grid-cols-[40px_minmax(0,1fr)_76px_76px] border-b px-2 py-2 text-[10px] font-semibold uppercase">
        <span className="text-center">#</span>
        <span>Участник</span>
        <span className="text-right">Общее</span>
        <span className="text-right">Чистое</span>
      </div>
      {[0, 1, 2, 3, 4, 5].map((item) => (
        <div
          key={item}
          className="tg-divider grid grid-cols-[40px_minmax(0,1fr)_76px_76px] items-center gap-0 border-b px-2 py-2.5 last:border-b-0"
        >
          <div className="tg-skeleton mx-auto h-4 w-4 animate-pulse rounded" />
          <div className="space-y-1.5 pr-2">
            <div className="tg-skeleton h-3.5 w-2/3 animate-pulse rounded" />
            <div className="tg-skeleton h-2.5 w-1/3 animate-pulse rounded" />
          </div>
          <div className="tg-skeleton ml-auto h-3 w-14 animate-pulse rounded" />
          <div className="tg-skeleton ml-auto h-3 w-14 animate-pulse rounded" />
        </div>
      ))}
    </div>
  );
}
