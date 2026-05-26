'use client';

import { useEffect, useMemo, useState } from 'react';
import { telegramApi } from '@/api/telegram';
import type { GiftAttachment } from '@/types';

export interface GiftPhotoUrlTarget {
  giftId: number;
  attachment: GiftAttachment;
}

export interface GiftPhotoUrlState {
  url: string | null;
  isLoading: boolean;
  hasFailed: boolean;
}

type LoadedPhotoUrl = {
  url?: string;
  hasFailed?: boolean;
};

export function useGiftPhotoUrls(
  targets: GiftPhotoUrlTarget[]
): Record<number, GiftPhotoUrlState> {
  const targetKey = useMemo(
    () =>
      targets
        .map(
          (target) =>
            `${target.giftId}:${target.attachment.id}:${target.attachment.gift_id}`
        )
        .join('|'),
    [targets]
  );
  const [loadedUrls, setLoadedUrls] = useState<Record<number, LoadedPhotoUrl>>(
    {}
  );

  useEffect(() => {
    let ignore = false;
    const pendingTargets = targets.filter((target) => {
      const existing = loadedUrls[target.attachment.id];
      return !existing?.url && !existing?.hasFailed;
    });

    if (pendingTargets.length === 0) {
      return;
    }

    async function loadPhotoUrls() {
      const updates: Record<number, LoadedPhotoUrl> = {};

      await Promise.all(
        pendingTargets.map(async (target) => {
          try {
            const url = await telegramApi.getFileURL(
              target.attachment.telegram_file_id
            );
            updates[target.attachment.id] = { url, hasFailed: false };
          } catch (err) {
            console.warn('Failed to load gift photo URL:', {
              gift_id: target.giftId,
              attachment_id: target.attachment.id,
              operation: 'load_gift_photo_url',
              error_message:
                err instanceof Error ? err.message : 'Unknown error',
            });
            updates[target.attachment.id] = { hasFailed: true };
          }
        })
      );

      if (!ignore) {
        setLoadedUrls((current) => ({ ...current, ...updates }));
      }
    }

    loadPhotoUrls();

    return () => {
      ignore = true;
    };
  }, [loadedUrls, targetKey, targets]);

  return targets.reduce<Record<number, GiftPhotoUrlState>>((result, target) => {
    const loaded = loadedUrls[target.attachment.id];
    result[target.attachment.id] = {
      url: loaded?.url || null,
      hasFailed: Boolean(loaded?.hasFailed),
      isLoading: !loaded?.url && !loaded?.hasFailed,
    };
    return result;
  }, {});
}
