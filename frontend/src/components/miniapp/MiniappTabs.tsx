"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

// Постоянная панель табов Mini App. Рендерится в layout группы (miniapp), поэтому
// не размонтируется при переходах между экранами лидерборда и каталога призов.

const tabs: Array<{ href: string; label: string; match: string }> = [
  { href: "/miniapp/leaderboard", label: "Лидерборд", match: "/miniapp/leaderboard" },
  { href: "/miniapp/gifts", label: "Призы", match: "/miniapp/gifts" },
];

const tabButtonClass =
  "flex-1 rounded-md border px-3 py-1.5 text-center text-[13px] font-semibold transition active:scale-[0.98]";

export default function MiniappTabs() {
  const pathname = usePathname() ?? "";

  return (
    <nav className="tg-topbar border-b px-3 py-2">
      <div className="mx-auto flex w-full max-w-md gap-1.5">
        {tabs.map((tab) => {
          const isActive = pathname.startsWith(tab.match);

          return (
            <Link
              key={tab.href}
              href={tab.href}
              aria-current={isActive ? "page" : undefined}
              className={`${tabButtonClass} ${
                isActive ? "tg-filter-active shadow-sm" : "tg-filter-inactive"
              }`}
            >
              {tab.label}
            </Link>
          );
        })}
      </div>
    </nav>
  );
}
