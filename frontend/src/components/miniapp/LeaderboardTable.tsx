"use client";

import type { KeyboardEvent } from "react";
import { useRouter } from "next/navigation";
import type { RankedLeaderboardEntry } from "@/utils/leaderboard";
import { bikeTypeLabel, genderShortLabel } from "./leaderboardFormat";

interface LeaderboardTableProps {
  rows: RankedLeaderboardEntry[];
  isLoading?: boolean;
}

export default function LeaderboardTable({ rows, isLoading }: LeaderboardTableProps) {
  return (
    <section
      className={`tg-card overflow-hidden rounded-xl border ${
        isLoading ? "opacity-70" : ""
      }`}
      aria-busy={isLoading}
    >
      <table className="w-full table-fixed border-collapse">
        <colgroup>
          <col className="w-[40px]" />
          <col />
          <col className="w-[76px]" />
          <col className="w-[76px]" />
        </colgroup>
        <thead className="tg-topbar">
          <tr className="tg-divider tg-muted border-b text-left text-[10px] font-semibold uppercase">
            <th scope="col" className="px-1.5 py-2 text-center">
              #
            </th>
            <th scope="col" className="px-1.5 py-2">
              Участник
            </th>
            <th scope="col" className="px-1.5 py-2 text-right">
              Общее
            </th>
            <th scope="col" className="px-1.5 py-2 text-right">
              Чистое
            </th>
          </tr>
        </thead>
        <tbody className="tg-table-body">
          {rows.map(({ entry, place }) => (
            <LeaderboardRow key={entry.id} entry={entry} place={place} />
          ))}
        </tbody>
      </table>
    </section>
  );
}

function LeaderboardRow({
  entry,
  place,
}: RankedLeaderboardEntry) {
  const router = useRouter();
  const href = `/miniapp/leaderboard/${entry.id}`;

  const open = () => {
    router.push(href);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLTableRowElement>) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }
    event.preventDefault();
    open();
  };

  return (
    <tr
      role="link"
      tabIndex={0}
      aria-label={`Открыть результат: ${entry.name}`}
      onClick={open}
      onKeyDown={handleKeyDown}
      className="tg-row-hover cursor-pointer align-middle focus:outline-none focus:ring-2 focus:ring-[var(--tg-button-color)]"
    >
      <td className="px-1 py-2 text-center">
        <span className="tg-title text-[13px] font-semibold tabular-nums">
          {place ?? "—"}
        </span>
      </td>
      <td className="min-w-0 px-1.5 py-2">
        <p className="tg-title truncate text-[13px] font-medium leading-[17px]">
          {entry.name}
        </p>
        <div className="tg-muted mt-0.5 flex items-center gap-1.5 text-[10px] font-medium leading-4">
          <span>{genderShortLabel(entry.gender)}</span>
          <span aria-hidden>·</span>
          <span className="truncate">{bikeTypeLabel(entry.bike_type)}</span>
        </div>
      </td>
      <td className="px-1.5 py-2 text-right">
        <span className="tg-title text-[12px] font-medium tabular-nums">
          {entry.elapsed_time ?? "—"}
        </span>
      </td>
      <td className="px-1.5 py-2 text-right">
        <span className="tg-muted text-[12px] tabular-nums">
          {entry.moving_time ?? "—"}
        </span>
      </td>
    </tr>
  );
}
