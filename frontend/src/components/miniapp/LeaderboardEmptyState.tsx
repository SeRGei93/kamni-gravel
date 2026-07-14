interface LeaderboardEmptyStateProps {
  isSearchActive?: boolean;
  onSearchClear?: () => void;
}

export default function LeaderboardEmptyState({
  isSearchActive = false,
  onSearchClear,
}: LeaderboardEmptyStateProps) {
  return (
    <section className="tg-card-dashed rounded-xl border border-dashed p-6 text-center">
      <div className="tg-soft-accent mx-auto h-10 w-10 rounded-full" />
      <h2 className="tg-title mt-4 text-lg font-semibold">
        {isSearchActive ? "Участники не найдены" : "Пока никого нет"}
      </h2>
      <p className="tg-muted mt-2 text-sm leading-5">
        {isSearchActive
          ? "Попробуйте другой запрос или очистите поиск."
          : "Для выбранных фильтров нет участников. Попробуйте другую категорию."}
      </p>
      {isSearchActive && onSearchClear && (
        <button
          type="button"
          onClick={onSearchClear}
          className="tg-link-button mt-4 rounded-lg border px-3 py-2 text-sm font-semibold"
        >
          Очистить поиск
        </button>
      )}
    </section>
  );
}
