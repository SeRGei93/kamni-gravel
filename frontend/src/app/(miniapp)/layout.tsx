import React from "react";
import Script from "next/script";
import { MiniappCatalogProvider } from "@/components/miniapp/MiniappCatalogContext";
import { MiniappLeaderboardProvider } from "@/components/miniapp/MiniappLeaderboardContext";
import { MiniappMyGiftsProvider } from "@/components/miniapp/MiniappMyGiftsContext";
import { MiniappSessionProvider } from "@/components/miniapp/MiniappSessionContext";
import MiniappTabs from "@/components/miniapp/MiniappTabs";
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
      <div
        className="tg-miniapp min-h-screen pb-[calc(env(safe-area-inset-bottom)+4.75rem)] antialiased"
        style={defaultTelegramDarkThemeStyle}
      >
        <MiniappSessionProvider>
          <MiniappCatalogProvider>
            <MiniappLeaderboardProvider>
              <MiniappMyGiftsProvider>
                {children}
                <MiniappTabs />
              </MiniappMyGiftsProvider>
            </MiniappLeaderboardProvider>
          </MiniappCatalogProvider>
        </MiniappSessionProvider>
      </div>
    </>
  );
}
