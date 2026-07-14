"use client";

import { useCallback, useEffect, useState } from "react";
import { miniappApi, MiniappApiError } from "@/api/miniapp";
import MyGiftCard from "@/components/miniapp/MyGiftCard";
import { useMiniappMyGifts } from "@/components/miniapp/MiniappMyGiftsContext";
import MiniappSpinner from "@/components/miniapp/MiniappSpinner";
import { useMiniappSession } from "@/components/miniapp/MiniappSessionContext";
import { useIsomorphicLayoutEffect } from "@/hooks/useIsomorphicLayoutEffect";
import type { ManualGift, MiniappParticipantOption } from "@/types";
import {
  MiniappMyGiftsRefreshError,
  updateManualGiftRecipient,
} from "@/utils/miniappMyGifts";

export default function MiniappMyGiftsPage() {
  const { session, isLoading: isSessionLoading, error: sessionError } = useMiniappSession();
  const { snapshot, setSnapshot, updateSnapshot, scrollYRef } = useMiniappMyGifts();
  const [isLoading, setIsLoading] = useState(false);
  const [savingGiftID, setSavingGiftID] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      const [giftResponse, participantResponse] = await Promise.all([
        miniappApi.getMyGifts(),
        miniappApi.getParticipants(),
      ]);
      setSnapshot({
        gifts: giftResponse.gifts,
        participants: participantResponse.participants,
      });
      console.debug("[miniapp] my gifts loaded", {
        giftCount: giftResponse.gifts.length,
        participantCount: participantResponse.total,
      });
      return true;
    } catch (loadError) {
      console.warn("[miniapp] my gifts load failed", {
        message: loadError instanceof Error ? loadError.message : "Unknown error",
      });
      if (snapshot) {
        return false;
      }
      if (loadError instanceof MiniappApiError && loadError.status === 404) {
        setError("Нет активного события");
      } else {
        setError("Не удалось загрузить мои призы");
      }
      return false;
    } finally {
      setIsLoading(false);
    }
  }, [setSnapshot, snapshot]);

  useEffect(() => {
    if (session && snapshot === null) {
      void load();
    }
  }, [load, session, snapshot]);

  const saveRecipient = useCallback(async (giftID: number, participantID: number | null) => {
    try {
      setSavingGiftID(giftID);
      await miniappApi.updateMyGiftRecipient(giftID, participantID);
      console.info("[miniapp] manual recipient updated", { giftId: giftID, participantId: participantID });
      updateSnapshot((current) => ({
        ...current,
        gifts: updateManualGiftRecipient(current.gifts, current.participants, giftID, participantID),
      }));
      // The server remains the source of truth after every mutation.
      const refreshed = await load();
      if (!refreshed) {
        console.warn("[FIX:my-gifts-recipient-cache] recipient cache updated without server refresh", {
          gift_id: giftID,
          mutation: participantID === null ? "clear" : "select",
        });
      }
    } finally {
      setSavingGiftID(null);
    }
  }, [load, updateSnapshot]);

  const assignRandomRecipient = useCallback(async (giftID: number) => {
    try {
      setSavingGiftID(giftID);
      await miniappApi.assignRandomMyGiftRecipient(giftID);
      console.info("[miniapp] random manual recipient assigned", { giftId: giftID });
      const refreshed = await load();
      if (!refreshed) {
        console.warn("[FIX:my-gifts-recipient-cache] random recipient assigned without server refresh", {
          gift_id: giftID,
          mutation: "random",
        });
        throw new MiniappMyGiftsRefreshError();
      }
    } finally {
      setSavingGiftID(null);
    }
  }, [load]);

  const gifts: ManualGift[] = snapshot?.gifts ?? [];
  const participants: MiniappParticipantOption[] = snapshot?.participants ?? [];

  // Кеш provider-а сохраняется между вкладками, поэтому при возврате через
  // нижнее меню прокрутка восстанавливается сразу и без нового запроса.
  useIsomorphicLayoutEffect(() => {
    window.scrollTo(0, scrollYRef.current);
    return () => {
      scrollYRef.current = window.scrollY;
    };
  }, [scrollYRef]);

  if (isSessionLoading) {
    return <MyGiftsState title="Призы от меня" text="Загружаем активное событие" loading />;
  }
  if (sessionError || !session) {
    return <MyGiftsState title="Призы от меня недоступны" text={sessionError ?? "Нет активного события"} />;
  }
  if (error) {
    return <MyGiftsState title="Призы от меня недоступны" text={error} onRetry={load} />;
  }
  if (isLoading && gifts.length === 0) {
    return <MyGiftsState title="Призы от меня" text="Загружаем добавленные призы" loading />;
  }

  return (
    <main className="tg-screen min-h-screen">
      <section className="mx-auto flex w-full max-w-md flex-col gap-3 px-3 py-3">
        {gifts.length === 0 ? (
          <section className="tg-card rounded-xl border p-5 text-center">
            <p className="tg-title text-sm font-semibold">Вы пока не добавили призы</p>
            <p className="tg-muted mt-2 text-xs leading-4">Добавленные вами призы появятся здесь для активного события.</p>
          </section>
        ) : (
          gifts.map((gift) => (
            <MyGiftCard
              key={gift.id}
              gift={gift}
              participants={participants}
              savingGiftID={savingGiftID}
              showGiftRecipients={session.event.show_gift_recipients}
              onSaveRecipient={saveRecipient}
              onAssignRandomRecipient={assignRandomRecipient}
            />
          ))
        )}
      </section>
    </main>
  );
}

function MyGiftsState({
  title,
  text,
  loading = false,
  onRetry,
}: {
  title: string;
  text: string;
  loading?: boolean;
  onRetry?: () => Promise<unknown>;
}) {
  return (
    <main className="tg-screen flex min-h-screen items-center justify-center px-5 py-8">
      <section className="tg-card flex w-full max-w-sm flex-col items-center rounded-xl border p-5 text-center">
        {loading && <MiniappSpinner size={24} />}
        <h1 className="tg-title mt-3 text-lg font-semibold">{title}</h1>
        <p className="tg-muted mt-2 text-sm leading-5">{text}</p>
        {onRetry && (
          <button
            type="button"
            onClick={() => void onRetry()}
            className="tg-link-button mt-4 rounded-lg border px-3 py-2 text-sm font-semibold"
          >
            Повторить
          </button>
        )}
      </section>
    </main>
  );
}
