"use client";

import type { GiftAttachment } from "@/types";
import GiftImage from "./GiftImage";

interface GiftPhotoGalleryProps {
  giftId: number;
  attachments?: GiftAttachment[];
}

export default function GiftPhotoGallery({
  giftId,
  attachments,
}: GiftPhotoGalleryProps) {
  const photos = attachments?.filter((attachment) => attachment.file_type === "photo") ?? [];
  const [primaryPhoto, ...secondaryPhotos] = photos;
  const hasPrimaryPhoto = Boolean(primaryPhoto);

  return (
    <div className="tg-placeholder tg-divider border-b">
      <div className={hasPrimaryPhoto ? "flex justify-center" : "h-36"}>
        <GiftImage giftId={giftId} attachment={primaryPhoto} variant="detail" />
      </div>

      {secondaryPhotos.length > 0 && (
        <div className="tg-divider grid grid-cols-2 gap-2 border-t p-2">
          {secondaryPhotos.map((photo) => (
            <div
              key={photo.id}
              className="tg-divider tg-placeholder aspect-square overflow-hidden rounded-lg border"
            >
              <GiftImage giftId={giftId} attachment={photo} variant="thumbnail" />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
