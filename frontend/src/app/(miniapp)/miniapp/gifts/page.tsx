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
    <main className="min-h-screen bg-gray-50 text-gray-900">
      <MiniappMasthead count={gifts.length} isLoading={isCatalogLoading} />

      <GiftFilters
        gender={gender}
        bikeType={bikeType}
        isLoading={isCatalogLoading}
        onGenderChange={setGender}
        onBikeTypeChange={setBikeType}
      />

      <section className="mx-auto flex w-full max-w-md flex-col gap-3 px-3 py-3">
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
    <section className="bg-gray-50 px-3 pb-3 pt-4">
      <div className="mx-auto w-full max-w-md rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
        <p className="text-xs font-medium text-orange-600">
          Gravel Bot
        </p>
        <div className="mt-1 flex items-end justify-between gap-3">
          <h1 className="text-2xl font-semibold leading-8 text-gray-900">
            Подарки
          </h1>
          <div className="mb-0.5 rounded-lg border border-orange-200 bg-orange-50 px-2.5 py-1.5 text-right">
            <p className="text-[10px] font-medium text-orange-600">
              Всего
            </p>
            <p className="text-lg font-semibold leading-none text-orange-600">{countLabel}</p>
          </div>
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
    <main className="flex min-h-screen items-center justify-center bg-gray-50 px-5 py-8 text-gray-900">
      <section className="w-full max-w-sm rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <div
          className={`mb-4 h-2 w-16 rounded-full ${
            isError ? "bg-error-500" : "bg-orange-500"
          }`}
        />
        <h1 className="text-xl font-semibold leading-7">{title}</h1>
        <p className="mt-2 text-sm leading-5 text-gray-500">{text}</p>
      </section>
    </main>
  );
}

function MiniappCatalogLoading() {
  return (
    <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
      <div className="grid grid-cols-[52px_minmax(0,1fr)_112px] border-b border-gray-200 bg-gray-50 px-2 py-2 text-[10px] font-semibold uppercase text-gray-500">
        <span>Фото</span>
        <span>Подарок</span>
        <span>Условия</span>
      </div>
      {[0, 1, 2, 3, 4].map((item) => (
        <div
          key={item}
          className="grid grid-cols-[52px_minmax(0,1fr)_112px] gap-0 border-b border-gray-200 px-2 py-1.5 last:border-b-0"
        >
          <div className="h-10 w-10 animate-pulse rounded-lg bg-orange-100" />
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
