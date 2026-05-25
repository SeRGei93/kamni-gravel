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
        setError("Подарок не найден");
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
            setError("Подарок не найден");
          }
        }
      } catch (loadError) {
        console.warn("[miniapp] Gift detail load failed", {
          giftId,
          message: loadError instanceof Error ? loadError.message : "Unknown error",
        });
        if (!ignore) {
          setError("Не удалось загрузить подарок");
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
    return <MiniappDetailState title="Подарок" text="Загружаем описание и фото" />;
  }

  if (error || !gift) {
    return <MiniappDetailState title="Подарок недоступен" text={error ?? "Подарок не найден"} />;
  }

  return <GiftDetailView gift={gift} />;
}

function MiniappDetailState({ title, text }: { title: string; text: string }) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-gray-50 px-5 py-8 text-gray-900">
      <section className="w-full max-w-sm rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <div className="mb-4 h-2 w-16 rounded-full bg-orange-500" />
        <h1 className="text-xl font-semibold leading-7">{title}</h1>
        <p className="mt-2 text-sm leading-5 text-gray-500">{text}</p>
        <Link
          href="/miniapp/gifts"
          className="mt-4 inline-flex rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm font-medium text-gray-700"
        >
          Вернуться
        </Link>
      </section>
    </main>
  );
}
