"use client";

import { useEffect, useRef, useState } from "react";
import { miniappApi } from "@/api/miniapp";
import { BoxCubeIcon } from "@/icons";
import type { GiftAttachment } from "@/types";

interface GiftImageProps {
  giftId: number;
  attachment?: GiftAttachment;
  variant?: "thumbnail" | "detail";
}

export default function GiftImage({ giftId, attachment, variant = "detail" }: GiftImageProps) {
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  // shouldLoad включается, когда плейсхолдер приближается к вьюпорту — до этого
  // момента блоб не запрашивается (lazy-load). Без этого все картинки каталога
  // начинали грузиться одновременно при открытии списка.
  const [shouldLoad, setShouldLoad] = useState(false);
  const placeholderRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!attachment || shouldLoad) {
      return;
    }

    const node = placeholderRef.current;
    if (!node) {
      return;
    }

    // Фолбэк: если IntersectionObserver недоступен — грузим сразу.
    if (typeof IntersectionObserver === "undefined") {
      setShouldLoad(true);
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setShouldLoad(true);
          observer.disconnect();
        }
      },
      // Запас, чтобы картинка успела подгрузиться до попадания в кадр.
      { rootMargin: "200px" }
    );
    observer.observe(node);

    return () => {
      observer.disconnect();
    };
  }, [attachment, shouldLoad]);

  useEffect(() => {
    let objectUrl: string | null = null;
    let ignore = false;

    async function loadImage() {
      if (!attachment) {
        setImageUrl(null);
        setFailed(false);
        return;
      }

      if (!shouldLoad) {
        return;
      }

      setFailed(false);

      try {
        const blob = await miniappApi.getTelegramFile(attachment.telegram_file_id);
        if (ignore) {
          return;
        }

        objectUrl = URL.createObjectURL(blob);
        setImageUrl(objectUrl);
      } catch {
        console.warn("[miniapp] Gift image fetch failed", {
          giftId,
          attachmentId: attachment.id,
        });
        if (!ignore) {
          setFailed(true);
          setImageUrl(null);
        }
      }
    }

    loadImage();

    return () => {
      ignore = true;
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl);
      }
    };
  }, [attachment, giftId, shouldLoad]);

  if (imageUrl) {
    return (
      // Blob object URLs from the protected miniapp file endpoint cannot use next/image.
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={imageUrl}
        alt="Фото приза"
        className={
          variant === "detail"
            ? "block max-h-[80vh] w-full object-contain"
            : "h-full w-full object-cover"
        }
      />
    );
  }

  return (
    <div
      ref={placeholderRef}
      className="tg-photo-placeholder tg-placeholder flex h-full w-full items-center justify-center text-center"
      data-variant={variant}
    >
      <div className="tg-photo-placeholder-content flex flex-col items-center">
        <BoxCubeIcon className="tg-photo-placeholder-svg tg-accent" aria-hidden="true" />
        {variant === "detail" && (
          <span className="tg-photo-placeholder-label tg-muted font-medium">
            {failed ? "Фото недоступно" : "Без фото"}
          </span>
        )}
      </div>
    </div>
  );
}
