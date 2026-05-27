"use client";

import { useEffect, useState } from "react";
import { miniappApi } from "@/api/miniapp";
import GiftCatalogTable from "@/components/miniapp/GiftCatalogTable";
import GiftEmptyState from "@/components/miniapp/GiftEmptyState";
import GiftFilters, { type MiniappGenderFilter } from "@/components/miniapp/GiftFilters";
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

const ALL_GENDER_CATALOG_FILTERS: GenderFilter[] = ["all", "male", "female"];

export default function MiniappGiftsPage() {
  const [session, setSession] = useState<MiniappSessionResponse | null>(null);
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [gender, setGender] = useState<MiniappGenderFilter>("all_genders");
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
          setError("Не удалось открыть каталог призов");
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
        const catalogResponses =
          gender === "all_genders"
            ? await Promise.all(
                ALL_GENDER_CATALOG_FILTERS.map((genderFilter) =>
                  miniappApi.getGifts({
                    gender: genderFilter,
                    bike_type: bikeType,
                  })
                )
              )
            : [
                await miniappApi.getGifts({
                  gender,
                  bike_type: bikeType,
                }),
              ];

        if (!ignore) {
          setGifts(mergeUniqueGifts(catalogResponses.flatMap((response) => response.gifts)));
        }
      } catch (loadError) {
        console.warn("[miniapp] Gift catalog load failed", {
          gender,
          bikeType,
          message: loadError instanceof Error ? loadError.message : "Unknown error",
        });
        if (!ignore) {
          setError("Не удалось загрузить призы");
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
    return <MiniappShellState title="Каталог призов" text="Загружаем активное событие" />;
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
    <main className="tg-screen min-h-screen">
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
    <main className="tg-screen flex min-h-screen items-center justify-center px-5 py-8">
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

function mergeUniqueGifts(gifts: Gift[]): Gift[] {
  const seenGiftIds = new Set<number>();

  const mergedGifts = gifts.filter((gift) => {
    if (seenGiftIds.has(gift.id)) {
      return false;
    }

    seenGiftIds.add(gift.id);
    return true;
  });

  return sortGiftsByPlace(mergedGifts);
}

function sortGiftsByPlace(gifts: Gift[]): Gift[] {
  return [...gifts].sort((left, right) => {
    if (left.place === undefined && right.place === undefined) {
      return 0;
    }
    if (left.place === undefined) {
      return 1;
    }
    if (right.place === undefined) {
      return -1;
    }

    return left.place - right.place;
  });
}

function MiniappCatalogLoading() {
  return (
    <div className="tg-card overflow-hidden rounded-xl border">
      <div className="tg-topbar grid grid-cols-[52px_minmax(0,1fr)_112px] border-b px-2 py-2 text-[10px] font-semibold uppercase">
        <span>Фото</span>
        <span>Приз</span>
        <span>Условия</span>
      </div>
      {[0, 1, 2, 3, 4].map((item) => (
        <div
          key={item}
          className="tg-divider grid grid-cols-[52px_minmax(0,1fr)_112px] gap-0 border-b px-2 py-1.5 last:border-b-0"
        >
          <div className="tg-soft-accent h-10 w-10 animate-pulse rounded-lg" />
          <div className="space-y-1.5 py-1 pr-2">
            <div className="tg-skeleton h-3.5 w-full animate-pulse rounded" />
            <div className="tg-skeleton h-3.5 w-2/3 animate-pulse rounded" />
          </div>
          <div className="space-y-1 py-0.5">
            <div className="tg-skeleton h-3 w-16 animate-pulse rounded" />
            <div className="tg-skeleton h-3 w-20 animate-pulse rounded" />
          </div>
        </div>
      ))}
    </div>
  );
}
