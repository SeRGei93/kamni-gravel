export default function GiftEmptyState() {
  return (
    <section className="rounded-lg border border-dashed border-[#2a2720]/30 bg-[#fff0d0]/92 p-6 text-center">
      <div className="mx-auto h-10 w-10 rounded-full bg-[#dd7a3c]/35" />
      <h2 className="mt-4 text-lg font-semibold text-[#211c16]">Подарков не найдено</h2>
      <p className="mt-2 text-sm leading-5 text-[#624226]">
        Для выбранных фильтров нет проверенных подарков. Попробуйте другую категорию.
      </p>
    </section>
  );
}
