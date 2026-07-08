import Link from "next/link";
import type { MiniappLeaderboardEntry } from "@/types";
import { formatDistanceKm, formatSpeed } from "@/utils/format";
import { bikeTypeLabel, genderFullLabel } from "./leaderboardFormat";

interface LeaderboardDetailViewProps {
  entry: MiniappLeaderboardEntry;
  place: number | null;
}

interface StatusPill {
  label: string;
  tone: "success" | "muted" | "danger";
}

function resolveStatus(entry: MiniappLeaderboardEntry): StatusPill {
  if (entry.status === "disqualified") {
    return { label: "Дисквалификация", tone: "danger" };
  }
  if (entry.status === "dnf") {
    return { label: "Сошёл с дистанции", tone: "danger" };
  }
  if (entry.is_finished) {
    return { label: "Проехал", tone: "success" };
  }
  return { label: "Нет результата", tone: "muted" };
}

const pillToneClass: Record<StatusPill["tone"], string> = {
  success: "tg-soft-accent",
  muted: "tg-divider tg-muted",
  danger: "tg-divider tg-muted",
};

export default function LeaderboardDetailView({
  entry,
  place,
}: LeaderboardDetailViewProps) {
  const status = resolveStatus(entry);
  const hasResult = entry.is_finished;
  const submittedAt = formatSubmittedAt(entry.submitted_at);

  return (
    <main className="tg-screen min-h-screen">
      <section className="tg-topbar border-b px-3 py-3">
        <div className="mx-auto flex w-full max-w-md items-center gap-3">
          <Link
            href="/miniapp/leaderboard"
            className="tg-link-button inline-flex rounded-lg border px-3 py-2 text-sm font-medium"
          >
            Назад
          </Link>
        </div>
      </section>

      <section className="mx-auto flex w-full max-w-md flex-col gap-3 px-3 py-3">
        <article className="tg-card overflow-hidden rounded-xl border">
          <div className="space-y-4 p-4">
            {/* Шапка: место, имя, категория, статус */}
            <div className="flex items-start gap-3">
              <div className="tg-soft-accent flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-base font-semibold tabular-nums">
                {place ?? "—"}
              </div>
              <div className="min-w-0 flex-1">
                <h1 className="tg-title break-words text-lg font-semibold leading-6">
                  {entry.name}
                </h1>
                <p className="tg-muted mt-1 text-xs font-medium">
                  {genderFullLabel(entry.gender)} · {bikeTypeLabel(entry.bike_type)}
                </p>
              </div>
              <span
                className={`shrink-0 rounded-md border px-2 py-1 text-[11px] font-semibold ${pillToneClass[status.tone]}`}
              >
                {status.label}
              </span>
            </div>

            {/* Ссылка на результат и дата отправки */}
            {(entry.result_link || submittedAt) && (
              <div className="tg-divider rounded-lg border px-3 py-2.5 text-sm">
                {entry.result_link && (
                  <div>
                    <p className="tg-muted text-xs font-medium">Ссылка на результат</p>
                    <a
                      href={entry.result_link}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="tg-accent mt-1 block break-all text-sm font-medium underline"
                    >
                      {entry.result_link}
                    </a>
                  </div>
                )}
                {submittedAt && (
                  <div className={entry.result_link ? "mt-3" : ""}>
                    <p className="tg-muted text-xs font-medium">Дата отправки результата</p>
                    <p className="tg-title mt-1 text-sm font-medium">{submittedAt}</p>
                  </div>
                )}
              </div>
            )}

            {/* Метрики заезда */}
            {hasResult ? (
              <div className="tg-divider grid grid-cols-2 overflow-hidden rounded-lg border sm:grid-cols-3">
                <Metric label="Общее время" value={entry.elapsed_time} />
                <Metric label="Время в движении" value={entry.moving_time} />
                <Metric label="Простой" value={entry.idle_time} />
                <Metric label="Ср. скорость" value={formatSpeed(entry.avg_speed_kmh)} />
                <Metric
                  label="Ср. скорость в движении"
                  value={formatSpeed(entry.avg_moving_speed_kmh)}
                />
                <Metric label="Дата проезда" value={entry.ride_date} />
                <Metric label="Дистанция" value={formatDistanceKm(entry.distance_meters)} />
                <Metric label="Пиковая скорость" value={formatSpeed(entry.peak_speed_kmh)} />
                <Metric label="Средний пульс" value={formatHeartRate(entry.avg_heart_rate)} />
                <Metric label="Максимальный пульс" value={formatHeartRate(entry.max_heart_rate)} />
                <Metric label="Средний каденс" value={formatCadence(entry.avg_cadence)} />
                <Metric label="Калории" value={formatCalories(entry.calories)} />
              </div>
            ) : (
              <p className="tg-divider tg-muted rounded-lg border px-3 py-2.5 text-sm font-medium">
                Участник ещё не отправил результат.
              </p>
            )}
          </div>
        </article>
      </section>
    </main>
  );
}

function Metric({ label, value }: { label: string; value?: string | null }) {
  const display = value && value.length > 0 ? value : "—";
  return (
    <div className="tg-divider border-b border-r px-3 py-2.5 last:border-r-0">
      <p className="tg-muted text-[11px] font-medium leading-4">{label}</p>
      <p className="tg-title mt-1 break-words text-sm font-semibold leading-5">{display}</p>
    </div>
  );
}

function formatHeartRate(value?: number): string {
  return value !== undefined && value !== null ? `${value} уд/мин` : "";
}

function formatCadence(value?: number): string {
  return value !== undefined && value !== null ? `${value} об/мин` : "";
}

function formatCalories(value?: number): string {
  return value !== undefined && value !== null ? `${value} ккал` : "";
}

function formatSubmittedAt(iso?: string): string {
  if (!iso) {
    return "";
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return date.toLocaleString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}
