"use client";

import { useEffect, useState } from "react";
import { miniappApi } from "@/api/miniapp";
import type { GiftAttachment } from "@/types";

interface GiftImageProps {
  giftId: number;
  attachment?: GiftAttachment;
}

export default function GiftImage({ giftId, attachment }: GiftImageProps) {
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
        alt="Фото подарка"
        className="h-full w-full object-cover"
      />
    );
  }

  return (
    <div className="flex h-full w-full items-center justify-center bg-[linear-gradient(160deg,#7f9294_0%,#8f6c92_45%,#9a812a_68%,#b85733_100%)] text-center">
      <div className="flex h-6 w-6 items-center justify-center rounded-full border border-[#f4f0df]/50 bg-[#252821]/80 text-[9px] font-semibold uppercase text-[#f4f0df]">
        {failed ? "!" : "нет"}
      </div>
    </div>
  );
}
