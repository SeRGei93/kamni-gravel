"use client";

import { useEffect, useState } from "react";
import { miniappApi } from "@/api/miniapp";
import GiftCatalogTable from "@/components/miniapp/GiftCatalogTable";
import GiftEmptyState from "@/components/miniapp/GiftEmptyState";
import GiftFilters from "@/components/miniapp/GiftFilters";
import type {
  BikeTypeFilter,
  GenderFilter,
  Gift,
  MiniappSessionResponse,
} from "@/types";
import {
  expandTelegramWebApp,
  isTelegramWebAppAvailable,
  readyTelegramWebApp,
} from "@/utils/telegramWebApp";

export default function MiniappGiftsPage() {
  const [session, setSession] = useState<MiniappSessionResponse | null>(null);
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [gender, setGender] = useState<GenderFilter>("all");
  const [bikeType, setBikeType] = useState<BikeTypeFilter>("all");
  const [isSessionLoading, setIsSessionLoading] = useState(true);
  const [isCatalogLoading, setIsCatalogLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isTelegramWebAppAvailable()) {
      readyTelegramWebApp();
      expandTelegramWebApp();
    }

    let ignore = false;

    async function loadSession() {
      setIsSessionLoading(true);
      setError(null);

      try {
        const data = await miniappApi.getSession();
        if (!ignore) {
          setSession(data);
        }
      } catch (loadError) {
        console.warn("[miniapp] Session load failed", {
          message: loadError instanceof Error ? loadError.message : "Unknown error",
        });
        if (!ignore) {
          setError("Не удалось открыть каталог подарков");
        }
      } finally {
        if (!ignore) {
          setIsSessionLoading(false);
        }
      }
    }

    loadSession();

    return () => {
      ignore = true;
    };
  }, []);

  useEffect(() => {
    let ignore = false;

    async function loadCatalog() {
      setIsCatalogLoading(true);

      try {
        const data = await miniappApi.getGifts({
          gender,
          bike_type: bikeType,
        });
        if (!ignore) {
          setGifts(data.gifts);
        }
      } catch (loadError) {
        console.warn("[miniapp] Gift catalog load failed", {
          gender,
          bikeType,
          message: loadError instanceof Error ? loadError.message : "Unknown error",
        });
        if (!ignore) {
          setError("Не удалось загрузить подарки");
        }
      } finally {
        if (!ignore) {
          setIsCatalogLoading(false);
        }
      }
    }

    if (session) {
      loadCatalog();
    }

    return () => {
      ignore = true;
    };
  }, [bikeType, gender, session]);

  if (isSessionLoading) {
    return <MiniappShellState title="Каталог подарков" text="Загружаем активное событие" />;
  }

  if (error) {
    return <MiniappShellState title="Каталог недоступен" text={error} tone="error" />;
  }

  if (!session) {
    return (
      <MiniappShellState
        title="Нет активного события"
        text="Каталог появится после открытия события"
      />
    );
  }

  return (
    <main
      className="min-h-screen bg-[#070707] text-[#111111]"
      style={{
        color: "#111111",
      }}
    >
      <MiniappMasthead count={gifts.length} isLoading={isCatalogLoading} />

      <GiftFilters
        gender={gender}
        bikeType={bikeType}
        isLoading={isCatalogLoading}
        onGenderChange={setGender}
        onBikeTypeChange={setBikeType}
      />

      <section className="mx-auto flex w-full max-w-md flex-col gap-3 bg-white px-3 py-3">
        {isCatalogLoading && gifts.length === 0 ? (
          <MiniappCatalogLoading />
        ) : gifts.length > 0 ? (
          <GiftCatalogTable gifts={gifts} isLoading={isCatalogLoading} />
        ) : (
          <GiftEmptyState />
        )}
      </section>
    </main>
  );
}

function MiniappMasthead({
  count,
  isLoading,
}: {
  count: number;
  isLoading: boolean;
}) {
  const countLabel = isLoading ? "--" : String(count).padStart(2, "0");

  return (
    <section className="bg-[#070707] px-3 pb-3 pt-4 text-white">
      <div className="mx-auto w-full max-w-md border border-white/15 px-3 py-3">
        <p className="text-[10px] font-semibold uppercase text-[#b7a87f]">
          Gravel Bot
        </p>
        <div className="mt-1 flex items-end justify-between gap-3">
          <h1 className="text-[42px] font-black uppercase leading-none text-white">
            Gifts
          </h1>
          <div className="mb-1 border border-[#f97316] px-2 py-1 text-right">
            <p className="text-[10px] font-semibold uppercase text-[#b7a87f]">
              Total
            </p>
            <p className="text-xl font-black leading-none text-[#f97316]">{countLabel}</p>
          </div>
        </div>
        <div className="mt-3 grid grid-cols-3 border border-white/15 text-center text-[10px] font-semibold uppercase text-white">
          <span className="border-r border-white/15 py-1.5">Catalog</span>
          <span className="border-r border-white/15 py-1.5 text-[#b7a87f]">Approved</span>
          <span className="py-1.5">Mini App</span>
        </div>
      </div>
    </section>
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
    <main className="flex min-h-screen items-center justify-center bg-[#070707] px-5 py-8 text-[#111111]">
      <section className="w-full max-w-sm border border-white/15 bg-white p-5 shadow-sm">
        <div
          className={`mb-4 h-2 w-16 rounded-full ${
            isError ? "bg-[#ef4444]" : "bg-[#f97316]"
          }`}
        />
        <h1 className="text-xl font-semibold leading-7">{title}</h1>
        <p className="mt-2 text-sm leading-5 text-[#525252]">{text}</p>
      </section>
    </main>
  );
}

function MiniappCatalogLoading() {
  return (
    <div className="overflow-hidden border border-[#d4d4d4] bg-white shadow-sm">
      <div className="grid grid-cols-[52px_minmax(0,1fr)_112px] border-b border-[#262626] bg-[#111111] px-2 py-2 text-[10px] font-semibold uppercase text-white">
        <span>Фото</span>
        <span>Подарок</span>
        <span>Условия</span>
      </div>
      {[0, 1, 2, 3, 4].map((item) => (
        <div
          key={item}
          className="grid grid-cols-[52px_minmax(0,1fr)_112px] gap-0 border-b border-[#e5e5e5] px-2 py-1.5 last:border-b-0"
        >
          <div className="h-10 w-10 animate-pulse rounded-md bg-[#fed7aa]" />
          <div className="space-y-1.5 py-1 pr-2">
            <div className="h-3.5 w-full animate-pulse rounded bg-[#e5e5e5]" />
            <div className="h-3.5 w-2/3 animate-pulse rounded bg-[#f3f4f6]" />
          </div>
          <div className="space-y-1 py-0.5">
            <div className="h-3 w-16 animate-pulse rounded bg-[#e5e5e5]" />
            <div className="h-3 w-20 animate-pulse rounded bg-[#f3f4f6]" />
          </div>
        </div>
      ))}
    </div>
  );
}
