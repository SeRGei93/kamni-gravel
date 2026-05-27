import React from "react";
import Script from "next/script";
import MiniappTheme from "@/components/miniapp/MiniappTheme";
import { defaultTelegramDarkThemeStyle } from "@/components/miniapp/telegramTheme";

export default function MiniappLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <Script src="https://telegram.org/js/telegram-web-app.js" strategy="afterInteractive" />
      <MiniappTheme />
      <div className="tg-miniapp min-h-screen antialiased" style={defaultTelegramDarkThemeStyle}>
        {children}
      </div>
    </>
  );
}
