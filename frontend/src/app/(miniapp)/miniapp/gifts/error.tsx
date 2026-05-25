"use client";

export default function MiniappGiftsError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-gray-950 px-5 py-8 text-gray-100" style={{ colorScheme: "dark" }}>
      <section className="w-full max-w-sm rounded-xl border border-gray-800 bg-gray-900 p-5 shadow-sm">
        <div className="mb-4 h-2 w-16 rounded-full bg-red-500" />
        <h1 className="text-xl font-semibold leading-7 text-white">Каталог недоступен</h1>
        <p className="mt-2 text-sm leading-5 text-gray-400">
          Не удалось загрузить экран подарков.
        </p>
        <button
          type="button"
          onClick={reset}
          className="mt-5 h-10 rounded-lg bg-orange-500 px-4 text-sm font-medium text-white"
        >
          Повторить
        </button>
      </section>
    </main>
  );
}
