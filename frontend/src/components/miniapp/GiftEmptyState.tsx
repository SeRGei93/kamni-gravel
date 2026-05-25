export default function GiftEmptyState() {
  return (
    <section className="border border-dashed border-[#d4d4d4] bg-white p-6 text-center">
      <div className="mx-auto h-10 w-10 bg-[#fed7aa]" />
      <h2 className="mt-4 text-lg font-semibold text-[#111111]">Подарков не найдено</h2>
      <p className="mt-2 text-sm leading-5 text-[#525252]">
        Для выбранных фильтров нет проверенных подарков. Попробуйте другую категорию.
      </p>
    </section>
  );
}
