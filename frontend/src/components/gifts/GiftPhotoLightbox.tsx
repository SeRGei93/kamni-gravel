'use client';

import { useEffect, useRef, useState } from 'react';
import Image from 'next/image';

interface GiftPhotoLightboxProps {
  giftId: number;
  attachmentId: number;
  url: string;
  onClose: () => void;
}

export default function GiftPhotoLightbox({
  giftId,
  attachmentId,
  url,
  onClose,
}: GiftPhotoLightboxProps) {
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const [hasFailed, setHasFailed] = useState(false);

  useEffect(() => {
    closeButtonRef.current?.focus();

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  const handleImageError = () => {
    if (!hasFailed) {
      console.warn('Failed to render enlarged gift photo:', {
        gift_id: giftId,
        attachment_id: attachmentId,
        operation: 'render_enlarged_gift_photo',
      });
    }
    setHasFailed(true);
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Просмотр фото приза"
      className="fixed inset-0 z-[99999] flex items-center justify-center bg-gray-950/85 p-4"
    >
      <button
        type="button"
        aria-label="Закрыть просмотр фото"
        className="absolute inset-0 cursor-default"
        onClick={onClose}
      />

      <div className="relative z-10 flex h-[80vh] w-full max-w-5xl flex-col gap-3">
        <div className="flex justify-end">
          <button
            ref={closeButtonRef}
            type="button"
            onClick={onClose}
            className="inline-flex items-center justify-center rounded-lg bg-white px-3 py-2 text-sm font-medium text-gray-700 ring-1 ring-inset ring-gray-300 transition hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:ring-gray-700 dark:hover:bg-gray-700"
          >
            Закрыть
          </button>
        </div>

        <div className="relative min-h-0 flex-1 overflow-hidden rounded-xl bg-gray-900 shadow-2xl">
          {hasFailed ? (
            <div className="flex h-full w-full items-center justify-center text-sm text-gray-300">
              Не удалось показать фото
            </div>
          ) : (
            <Image
              src={url}
              alt="Фото приза"
              fill
              sizes="100vw"
              className="object-contain"
              onError={handleImageError}
            />
          )}
        </div>
      </div>
    </div>
  );
}
