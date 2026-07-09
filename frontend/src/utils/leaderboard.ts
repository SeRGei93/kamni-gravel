import type { LeaderboardGenderFilter } from "@/components/miniapp/MiniappLeaderboardContext";
import type { BikeTypeFilter, MiniappLeaderboardEntry } from "@/types";

// Запись лидерборда с отображаемым местом. place === null означает, что участник
// не в зачёте текущего представления (не финишировал / сошёл / дисквалифицирован).
export interface RankedLeaderboardEntry {
  entry: MiniappLeaderboardEntry;
  place: number | null;
}

// Участник в зачёте: активный статус, финишировал и есть положительное общее
// время. Повторяет серверную логику entity.Participant.IsRanked + расчёт мест.
export function isRankedEntry(entry: MiniappLeaderboardEntry): boolean {
  return (
    entry.status === "active" &&
    entry.is_finished &&
    typeof entry.elapsed_time_sec === "number" &&
    entry.elapsed_time_sec > 0
  );
}

// filterLeaderboard оставляет участников, подходящих под фильтры пола и типа
// велосипеда. "all" — без ограничения.
export function filterLeaderboard(
  entries: MiniappLeaderboardEntry[],
  gender: LeaderboardGenderFilter,
  bikeType: BikeTypeFilter
): MiniappLeaderboardEntry[] {
  return entries.filter((entry) => {
    if (gender !== "all" && entry.gender !== gender) {
      return false;
    }
    if (bikeType !== "all" && entry.bike_type !== bikeType) {
      return false;
    }
    return true;
  });
}

// rankAndFilterLeaderboard фильтрует список и назначает места В ПРЕДЕЛАХ текущего
// представления: финишировавшие в зачёте идут первыми, отсортированы по общему
// времени и получают последовательные места (1..N), затем — остальные без места.
// Такой пересчёт делает любой набор фильтров (пол, велосипед, оба, никакой)
// согласованным без отдельных серверных колонок мест по каждому срезу.
export function rankAndFilterLeaderboard(
  entries: MiniappLeaderboardEntry[],
  gender: LeaderboardGenderFilter,
  bikeType: BikeTypeFilter
): RankedLeaderboardEntry[] {
  const filtered = filterLeaderboard(entries, gender, bikeType);

  const ranked = filtered
    .filter(isRankedEntry)
    .sort((left, right) => {
      const byElapsed = (left.elapsed_time_sec ?? 0) - (right.elapsed_time_sec ?? 0);
      if (byElapsed !== 0) {
        return byElapsed;
      }
      // Тай-брейк: при равном общем времени выше тот, у кого меньше чистое время
      // (время в движении); отсутствующее чистое время уходит в конец.
      return (
        (left.moving_time_sec ?? Number.MAX_SAFE_INTEGER) -
        (right.moving_time_sec ?? Number.MAX_SAFE_INTEGER)
      );
    });

  const others = filtered
    .filter((entry) => !isRankedEntry(entry))
    .sort((left, right) => left.name.localeCompare(right.name, "ru"));

  return [
    ...ranked.map((entry, index) => ({ entry, place: index + 1 })),
    ...others.map((entry) => ({ entry, place: null })),
  ];
}
