"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
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
];

const myGiftsNavigationItem: MiniappNavigationItem = {
  href: "/miniapp/my-gifts",
  label: "Призы от меня",
  match: "/miniapp/my-gifts",
  icon: "my-gifts",
};

const navigationItemClass =
  "flex min-w-0 flex-col items-center gap-0.5 rounded-[1.5rem] px-1.5 py-1.5 text-[10px] font-semibold leading-3 transition-[flex,background-color,color,box-shadow,transform] duration-300 ease-out motion-reduce:transition-none active:scale-[0.96]";

const scrollDirectionThreshold = 12;

export default function MiniappTabs() {
  const pathname = usePathname() ?? "";
  const { session } = useMiniappSession();
  const [isCompact, setIsCompact] = useState(false);
  const previousScrollY = useRef(0);
  const hasMyGifts = session?.has_my_gifts ?? false;
  const myResultParticipantID = session?.my_result_participant_id ?? null;
  const myResultNavigationItem: MiniappNavigationItem | null = myResultParticipantID
    ? {
        href: `/miniapp/leaderboard/${myResultParticipantID}`,
        label: "Мой результат",
        match: `/miniapp/leaderboard/${myResultParticipantID}`,
        icon: "result",
      }
    : null;

  const navigationItems: MiniappNavigationItem[] = [
    ...fixedNavigationItems,
    ...(hasMyGifts ? [myGiftsNavigationItem] : []),
    ...(myResultNavigationItem ? [myResultNavigationItem] : []),
  ];
  const myResultHref = myResultParticipantID
    ? `/miniapp/leaderboard/${myResultParticipantID}`
    : null;

  useEffect(() => {
    previousScrollY.current = window.scrollY;

    const handleScroll = () => {
      const currentScrollY = window.scrollY;
      const scrollDelta = currentScrollY - previousScrollY.current;

      if (currentScrollY <= 0) {
        setIsCompact(false);
        previousScrollY.current = currentScrollY;
        return;
      }
      if (Math.abs(scrollDelta) < scrollDirectionThreshold) {
        return;
      }

      setIsCompact(scrollDelta > 0);
      previousScrollY.current = currentScrollY;
    };

    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  return (
    <nav
      aria-label="Навигация Mini App"
      className={`fixed inset-x-0 bottom-[calc(env(safe-area-inset-bottom)+1.25rem)] z-30 transition-[padding] duration-300 ease-out motion-reduce:transition-none ${
        isCompact ? "px-10" : "px-5"
      }`}
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
                  ? "tg-liquid-glass-nav-active tg-title flex-[1.6]"
                  : "tg-muted flex-1 hover:bg-[var(--tg-hover-bg-color)]"
              }`}
            >
              <NavigationIcon icon={item.icon} />
              <span className={isActive ? "whitespace-nowrap" : "truncate"}>{item.label}</span>
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
        <path strokeLinecap="round" strokeLinejoin="round" d="M11 14h2a2 2 0 0 0 0-4h-3c-.6 0-1.1.2-1.4.6L3 16" />
        <path strokeLinecap="round" strokeLinejoin="round" d="m14.45 13.39 5.05-4.694C20.196 8 21 6.85 21 5.75a2.75 2.75 0 0 0-4.797-1.837.276.276 0 0 1-.406 0A2.75 2.75 0 0 0 11 5.75c0 1.2.802 2.248 1.5 2.946L16 11.95" />
        <path strokeLinecap="round" strokeLinejoin="round" d="m2 15 6 6" />
        <path strokeLinecap="round" strokeLinejoin="round" d="m7 20 1.6-1.4c.3-.4.8-.6 1.4-.6h4c1.1 0 2.1-.4 2.8-1.2l4.6-4.4a1 1 0 0 0-2.75-2.91" />
      </svg>
    );
  }

  return (
    <svg {...commonProps}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm7 8a7 7 0 0 0-14 0" />
    </svg>
  );
}
