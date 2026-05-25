export default function MiniappGiftsLoading() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-[#f6f7ef] px-5 py-8 text-[#172016]">
      <section className="w-full max-w-sm rounded-lg border border-black/10 bg-white p-5 shadow-sm">
        <div className="mb-4 h-2 w-16 animate-pulse rounded-full bg-emerald-600" />
        <h1 className="text-xl font-semibold leading-7">Каталог подарков</h1>
        <p className="mt-2 text-sm leading-5 text-gray-600">Загружаем подарки</p>
      </section>
    </main>
  );
}
