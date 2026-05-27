export default function GiftEmptyState() {
  return (
    <section className="tg-card-dashed rounded-xl border border-dashed p-6 text-center">
      <div className="tg-soft-accent mx-auto h-10 w-10 rounded-full" />
      <h2 className="tg-title mt-4 text-lg font-semibold">Призов не найдено</h2>
      <p className="tg-muted mt-2 text-sm leading-5">
        Для выбранных фильтров нет проверенных призов. Попробуйте другую категорию.
      </p>
    </section>
  );
}
