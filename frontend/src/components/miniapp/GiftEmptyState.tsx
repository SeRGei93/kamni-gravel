export default function GiftEmptyState() {
  return (
    <section className="rounded-lg border border-dashed border-[#252821]/25 bg-[#f4f0df]/90 p-6 text-center">
      <div className="mx-auto h-10 w-10 rounded-full bg-[#b85733]/25" />
      <h2 className="mt-4 text-lg font-semibold text-[#1f211c]">Подарков не найдено</h2>
      <p className="mt-2 text-sm leading-5 text-[#4c4a40]">
        Для выбранных фильтров нет проверенных подарков. Попробуйте другую категорию.
      </p>
    </section>
  );
}
