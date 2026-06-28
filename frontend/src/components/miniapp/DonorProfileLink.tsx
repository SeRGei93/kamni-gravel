"use client";

import type { MouseEvent } from "react";
import { getTelegramWebApp, openTelegramProfile } from "@/utils/telegramWebApp";

interface DonorProfileLinkProps {
  label: string;
  username: string;
}

// DonorProfileLink показывает имя дарителя ссылкой на его профиль в Telegram.
// В Telegram открываем профиль через нативный openTelegramLink, вне Telegram —
// обычным переходом по ссылке (href остаётся рабочим фолбэком без JS).
export default function DonorProfileLink({ label, username }: DonorProfileLinkProps) {
  const handle = username.replace(/^@+/, "").trim();
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
