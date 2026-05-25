import React from "react";
import Script from "next/script";

export default function MiniappLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <Script src="https://telegram.org/js/telegram-web-app.js" strategy="afterInteractive" />
      <div className="min-h-screen bg-white text-gray-900 antialiased">
        {children}
      </div>
    </>
  );
}
