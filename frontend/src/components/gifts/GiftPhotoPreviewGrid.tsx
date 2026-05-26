'use client';

import { useMemo, useState } from 'react';
import Image from 'next/image';
import type { Gift } from '@/types';
import GiftPhotoLightbox from './GiftPhotoLightbox';
import { useGiftPhotoUrls } from './useGiftPhotoUrls';

interface GiftPhotoPreviewGridProps {
  gift: Gift;
}

export default function GiftPhotoPreviewGrid({ gift }: GiftPhotoPreviewGridProps) {
  const photos = useMemo(
    () =>
      gift.attachments?.filter(
        (attachment) => attachment.file_type === 'photo'
      ) || [],
    [gift.attachments]
  );
  const photoTargets = useMemo(
    () =>
      photos.map((attachment) => ({
        giftId: gift.id,
        attachment,
      })),
    [gift.id, photos]
  );
  const photoUrls = useGiftPhotoUrls(photoTargets);
  const [activeAttachmentId, setActiveAttachmentId] = useState<number | null>(
    null
  );
  const [failedImageIds, setFailedImageIds] = useState<Set<number>>(
    () => new Set()
  );

  const activePhoto = photos
    .map((attachment) => ({
      attachment,
      state: photoUrls[attachment.id],
    }))
    .find(
      (photo) => photo.attachment.id === activeAttachmentId && photo.state?.url
    );

  const handleImageError = (attachmentId: number) => {
    if (!failedImageIds.has(attachmentId)) {
      console.warn('Failed to render gift photo preview:', {
        gift_id: gift.id,
        attachment_id: attachmentId,
        operation: 'render_gift_photo_preview',
      });
    }

    setFailedImageIds((current) => {
      const next = new Set(current);
      next.add(attachmentId);
      return next;
    });
  };

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 dark:border-white/[0.05] dark:bg-white/[0.03]">
      <div className="mb-4 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-gray-800 dark:text-white/90">
          Фото подарка
        </h2>
        <span className="text-xs text-gray-500 dark:text-gray-400">
          {photos.length}
        </span>
      </div>

      {photos.length === 0 ? (
        <div className="rounded-lg border border-dashed border-gray-200 px-3 py-4 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400">
          Фото не прикреплены
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-3">
          {photos.map((attachment) => {
            const state = photoUrls[attachment.id];
            const imageFailed = failedImageIds.has(attachment.id);
            const canOpen = Boolean(state?.url && !state.hasFailed && !imageFailed);

            return (
              <button
                key={attachment.id}
                type="button"
                disabled={!canOpen}
                onClick={() => setActiveAttachmentId(attachment.id)}
                className="group relative aspect-square overflow-hidden rounded-lg border border-gray-200 bg-gray-100 text-xs text-gray-500 transition hover:border-brand-300 focus:outline-none focus:ring-3 focus:ring-brand-500/10 disabled:cursor-default dark:border-gray-800 dark:bg-gray-900 dark:text-gray-400"
              >
                {state?.url && !imageFailed ? (
                  <Image
                    src={state.url}
                    alt="Фото подарка"
                    fill
                    sizes="160px"
                    className="object-cover transition group-hover:scale-105"
                    onError={() => handleImageError(attachment.id)}
                  />
                ) : (
                  <span className="flex h-full w-full items-center justify-center px-2 text-center">
                    {state?.hasFailed || imageFailed
                      ? 'Не удалось загрузить'
                      : 'Загрузка'}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      )}

      {activePhoto?.state?.url && (
        <GiftPhotoLightbox
          giftId={gift.id}
          attachmentId={activePhoto.attachment.id}
          url={activePhoto.state.url}
          onClose={() => setActiveAttachmentId(null)}
        />
      )}
    </div>
  );
}
