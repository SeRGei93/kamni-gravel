export default function MiniappGiftsLoading() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-gray-950 px-5 py-8 text-gray-100" style={{ colorScheme: "dark" }}>
      <section className="w-full max-w-sm rounded-xl border border-gray-800 bg-gray-900 p-5 shadow-sm">
        <div className="mb-4 h-2 w-16 animate-pulse rounded-full bg-orange-500" />
        <h1 className="text-xl font-semibold leading-7 text-white">Каталог подарков</h1>
        <p className="mt-2 text-sm leading-5 text-gray-400">Загружаем подарки</p>
      </section>
    </main>
  );
}
