export default function GiftEmptyState() {
  return (
    <section className="rounded-xl border border-dashed border-gray-700 bg-gray-900 p-6 text-center shadow-sm">
      <div className="mx-auto h-10 w-10 rounded-full bg-orange-950/60" />
      <h2 className="mt-4 text-lg font-semibold text-white">Подарков не найдено</h2>
      <p className="mt-2 text-sm leading-5 text-gray-400">
        Для выбранных фильтров нет проверенных подарков. Попробуйте другую категорию.
      </p>
    </section>
  );
}
