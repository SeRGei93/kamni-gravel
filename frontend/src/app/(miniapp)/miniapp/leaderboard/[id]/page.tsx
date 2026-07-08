"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { miniappApi } from "@/api/miniapp";
import LeaderboardDetailView from "@/components/miniapp/LeaderboardDetailView";
import { useMiniappLeaderboard } from "@/components/miniapp/MiniappLeaderboardContext";
import MiniappSpinner from "@/components/miniapp/MiniappSpinner";
import { rankAndFilterLeaderboard } from "@/utils/leaderboard";
import {
  expandTelegramWebApp,
  isTelegramWebAppAvailable,
  readyTelegramWebApp,
  waitForTelegramInitData,
} from "@/utils/telegramWebApp";

export default function MiniappLeaderboardDetailPage() {
  const params = useParams();
  const participantId = useMemo(() => Number(params.id), [params.id]);

  const { entries, setEntries, gender, bikeType } = useMiniappLeaderboard();
  const [isLoading, setIsLoading] = useState(() => entries === null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isTelegramWebAppAvailable()) {
      readyTelegramWebApp();
      expandTelegramWebApp();
    }
  }, []);

  // Если список ещё не загружен (глубокий переход / перезагрузка) — подтягиваем
  // лидерборд один раз и кешируем в контексте, дальше берём участника из кеша.
  useEffect(() => {
    if (entries !== null) {
      setIsLoading(false);
      return;
    }

    let ignore = false;

    async function loadLeaderboard() {
      setIsLoading(true);
      setError(null);

      try {
        await waitForTelegramInitData();
        const data = await miniappApi.getLeaderboard();
        if (!ignore) {
          setEntries(data.participants);
        }
      } catch (loadError) {
        console.warn("[miniapp] Leaderboard detail load failed", {
          message: loadError instanceof Error ? loadError.message : "Unknown error",
        });
        if (!ignore) {
          setError("Не удалось загрузить результат");
        }
      } finally {
        if (!ignore) {
          setIsLoading(false);
        }
      }
    }

    loadLeaderboard();

    return () => {
      ignore = true;
    };
  }, [entries, setEntries]);

  const { entry, place } = useMemo(() => {
    if (!entries) {
      return { entry: undefined, place: null as number | null };
    }
    const found = entries.find((item) => item.id === participantId);
    if (!found) {
      return { entry: undefined, place: null as number | null };
    }
    // Место — как в текущем срезе фильтров (то, что видел пользователь в списке);
    // если участник не попадает в текущий фильтр — берём абсолютное место.
    const inView = rankAndFilterLeaderboard(entries, gender, bikeType).find(
      (row) => row.entry.id === participantId
    );
    const displayPlace =
      inView?.place ??
      rankAndFilterLeaderboard(entries, "all", "all").find(
        (row) => row.entry.id === participantId
      )?.place ??
      null;
    return { entry: found, place: displayPlace };
  }, [entries, participantId, gender, bikeType]);

  if (isLoading) {
    return (
      <main className="tg-screen flex min-h-[60vh] items-center justify-center px-5 py-8">
        <section className="tg-card flex w-full max-w-sm flex-col items-center gap-3 rounded-xl border p-6">
          <MiniappSpinner size={28} />
          <p className="tg-muted text-sm leading-5">Загружаем результат</p>
        </section>
      </main>
    );
  }

  if (error || !entry) {
    return (
      <MiniappDetailState
        title="Результат недоступен"
        text={error ?? "Участник не найден"}
      />
    );
  }

  return <LeaderboardDetailView entry={entry} place={place} />;
}

function MiniappDetailState({ title, text }: { title: string; text: string }) {
  return (
    <main className="tg-screen flex min-h-[60vh] items-center justify-center px-5 py-8">
      <section className="tg-card w-full max-w-sm rounded-xl border p-5">
        <div className="tg-accent-bar mb-4 h-2 w-16 rounded-full" />
        <h1 className="tg-title text-xl font-semibold leading-7">{title}</h1>
        <p className="tg-muted mt-2 text-sm leading-5">{text}</p>
        <Link
          href="/miniapp/leaderboard"
          className="tg-link-button mt-4 inline-flex rounded-lg border px-3 py-2 text-sm font-medium"
        >
          Вернуться
        </Link>
      </section>
    </main>
  );
}
