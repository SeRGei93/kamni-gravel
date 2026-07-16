"use client";

import type { MouseEvent } from "react";
import { getTelegramWebApp, openTelegramProfile } from "@/utils/telegramWebApp";

interface TelegramProfileLinkProps {
  label: string;
  username?: string;
}

// TelegramProfileLink открывает профиль по username через нативный Telegram API.
// Вне Mini App обычная ссылка остаётся рабочим фолбэком без JavaScript.
export default function TelegramProfileLink({ label, username }: TelegramProfileLinkProps) {
  const handle = username?.replace(/^@+/, "").trim();

  if (!handle) {
    return label;
  }

  const url = `https://t.me/${handle}`;

  const handleClick = (event: MouseEvent<HTMLAnchorElement>) => {
    if (getTelegramWebApp()?.openTelegramLink) {
      event.preventDefault();
      openTelegramProfile(handle);
    }
  };

  return (
    <a
      href={url}
      target="_blank"
      rel="noreferrer"
      onClick={handleClick}
      className="tg-accent break-words underline underline-offset-2"
    >
      {label}
    </a>
  );
}
