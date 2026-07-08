export default function LeaderboardEmptyState() {
  return (
    <section className="tg-card-dashed rounded-xl border border-dashed p-6 text-center">
      <div className="tg-soft-accent mx-auto h-10 w-10 rounded-full" />
      <h2 className="tg-title mt-4 text-lg font-semibold">Пока никого нет</h2>
      <p className="tg-muted mt-2 text-sm leading-5">
        Для выбранных фильтров нет участников. Попробуйте другую категорию.
      </p>
    </section>
  );
}
