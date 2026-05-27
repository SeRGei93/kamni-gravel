"use client";

import { useEffect, useState } from "react";
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

  useEffect(() => {
    let objectUrl: string | null = null;
    let ignore = false;

    async function loadImage() {
      if (!attachment) {
        setImageUrl(null);
        setFailed(false);
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
  }, [attachment, giftId]);

  if (imageUrl) {
    return (
      // Blob object URLs from the protected miniapp file endpoint cannot use next/image.
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={imageUrl}
        alt="Фото приза"
        className="h-full w-full object-cover"
      />
    );
  }

  return (
    <div
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
