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
      <div className="min-h-screen bg-gray-950 text-gray-100 antialiased" style={{ colorScheme: "dark" }}>
        {children}
      </div>
    </>
  );
}
