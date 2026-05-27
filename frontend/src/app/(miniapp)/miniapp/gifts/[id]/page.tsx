"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { miniappApi } from "@/api/miniapp";
import GiftDetailView from "@/components/miniapp/GiftDetailView";
import type { GenderFilter, Gift } from "@/types";
import {
  expandTelegramWebApp,
  isTelegramWebAppAvailable,
  readyTelegramWebApp,
} from "@/utils/telegramWebApp";

export default function MiniappGiftDetailPage() {
  const params = useParams();
  const giftId = useMemo(() => Number(params.id), [params.id]);
  const [gift, setGift] = useState<Gift | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isTelegramWebAppAvailable()) {
      readyTelegramWebApp();
      expandTelegramWebApp();
    }
  }, []);

  useEffect(() => {
    let ignore = false;

    async function loadGift() {
      if (!Number.isFinite(giftId) || giftId <= 0) {
        setError("Приз не найден");
        setIsLoading(false);
        return;
      }

      setIsLoading(true);
      setError(null);

      try {
        const genderFilters: GenderFilter[] = ["all", "male", "female"];
        const giftLists = await Promise.all(
          genderFilters.map((gender) => miniappApi.getGifts({ gender, bike_type: "all" }))
        );
        const selectedGift =
          giftLists.flatMap((data) => data.gifts).find((item) => item.id === giftId) ?? null;

        if (!ignore) {
          setGift(selectedGift);
          if (!selectedGift) {
            setError("Приз не найден");
          }
        }
      } catch (loadError) {
        console.warn("[miniapp] Gift detail load failed", {
          giftId,
          message: loadError instanceof Error ? loadError.message : "Unknown error",
        });
        if (!ignore) {
          setError("Не удалось загрузить приз");
        }
      } finally {
        if (!ignore) {
          setIsLoading(false);
        }
      }
    }

    loadGift();

    return () => {
      ignore = true;
    };
  }, [giftId]);

  if (isLoading) {
    return <MiniappDetailState title="Приз" text="Загружаем описание и фото" />;
  }

  if (error || !gift) {
    return <MiniappDetailState title="Приз недоступен" text={error ?? "Приз не найден"} />;
  }

  return <GiftDetailView gift={gift} />;
}

function MiniappDetailState({ title, text }: { title: string; text: string }) {
  return (
    <main className="tg-screen flex min-h-screen items-center justify-center px-5 py-8">
      <section className="tg-card w-full max-w-sm rounded-xl border p-5">
        <div className="tg-accent-bar mb-4 h-2 w-16 rounded-full" />
        <h1 className="tg-title text-xl font-semibold leading-7">{title}</h1>
        <p className="tg-muted mt-2 text-sm leading-5">{text}</p>
        <Link
          href="/miniapp/gifts"
          className="tg-link-button mt-4 inline-flex rounded-lg border px-3 py-2 text-sm font-medium"
        >
          Вернуться
        </Link>
      </section>
    </main>
  );
}
