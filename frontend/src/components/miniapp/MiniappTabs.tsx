"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useMiniappSession } from "@/components/miniapp/MiniappSessionContext";

// Постоянная нижняя навигация Mini App. Рендерится в layout группы (miniapp),
// поэтому не размонтируется при переходах между экранами лидерборда и каталога.

type NavigationIcon = "leaderboard" | "gifts" | "my-gifts" | "result";

interface MiniappNavigationItem {
  href: string;
  label: string;
  match: string;
  icon: NavigationIcon;
}

const fixedNavigationItems: MiniappNavigationItem[] = [
  {
    href: "/miniapp/leaderboard",
    label: "Лидерборд",
    match: "/miniapp/leaderboard",
    icon: "leaderboard",
  },
  { href: "/miniapp/gifts", label: "Призы", match: "/miniapp/gifts", icon: "gifts" },
  {
    href: "/miniapp/my-gifts",
    label: "Мои призы",
    match: "/miniapp/my-gifts",
    icon: "my-gifts",
  },
];

const navigationItemClass =
  "flex min-w-0 flex-1 flex-col items-center gap-0.5 rounded-[1.5rem] px-1.5 py-1.5 text-[10px] font-semibold leading-3 transition active:scale-[0.96]";

export default function MiniappTabs() {
  const pathname = usePathname() ?? "";
  const { session } = useMiniappSession();
  const myResultParticipantID = session?.my_result_participant_id ?? null;

  const navigationItems = myResultParticipantID
    ? [
        ...fixedNavigationItems,
        {
          href: `/miniapp/leaderboard/${myResultParticipantID}`,
          label: "Мой результат",
          match: `/miniapp/leaderboard/${myResultParticipantID}`,
          icon: "result" as const,
        },
      ]
    : fixedNavigationItems;
  const myResultHref = myResultParticipantID
    ? `/miniapp/leaderboard/${myResultParticipantID}`
    : null;

  return (
    <nav
      aria-label="Навигация Mini App"
      className="fixed inset-x-0 bottom-[calc(env(safe-area-inset-bottom)+1.25rem)] z-30 px-4"
    >
      <div className="tg-liquid-glass-nav mx-auto flex w-full max-w-md items-stretch rounded-[1.75rem] p-1">
        {navigationItems.map((item) => {
          const isMyResultRoute = myResultHref !== null && pathname === myResultHref;
          const isActive =
            item.icon === "result"
              ? isMyResultRoute
              : pathname.startsWith(item.match) && !isMyResultRoute;

          return (
            <Link
              key={item.href}
              href={item.href}
              aria-current={isActive ? "page" : undefined}
              className={`${navigationItemClass} ${
                isActive
                  ? "tg-liquid-glass-nav-active tg-title"
                  : "tg-muted hover:bg-[var(--tg-hover-bg-color)]"
              }`}
            >
              <NavigationIcon icon={item.icon} />
              <span className="truncate">{item.label}</span>
            </Link>
          );
        })}
      </div>
    </nav>
  );
}

function NavigationIcon({ icon }: { icon: NavigationIcon }) {
  const commonProps = {
    className: "h-4 w-4 shrink-0",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: 1.8,
    viewBox: "0 0 24 24",
    "aria-hidden": true,
  };

  if (icon === "leaderboard") {
    return (
      <svg {...commonProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M7 20v-7m5 7V4m5 16v-4" />
      </svg>
    );
  }

  if (icon === "gifts") {
    return (
      <svg {...commonProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M20 12v8H4v-8m16-4H4v4h16V8Zm-8 0v12m0-12H8.5A2.5 2.5 0 1 1 12 4.5V8Zm0 0h3.5A2.5 2.5 0 1 0 12 4.5V8Z" />
      </svg>
    );
  }

  if (icon === "my-gifts") {
    return (
      <svg {...commonProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M8 3h8v4a4 4 0 0 1-8 0V3Zm-3 1H3v2a4 4 0 0 0 4 4m10-6h2v2a4 4 0 0 1-4 4m-5-1v5m-3 4h10m-5-5a4 4 0 0 0 4-4V7H8v2a4 4 0 0 0 4 4Z" />
      </svg>
    );
  }

  return (
    <svg {...commonProps}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm7 8a7 7 0 0 0-14 0" />
    </svg>
  );
}
