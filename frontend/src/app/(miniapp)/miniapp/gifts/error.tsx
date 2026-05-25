"use client";

export default function MiniappGiftsError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-[#f6f7ef] px-5 py-8 text-[#172016]">
      <section className="w-full max-w-sm rounded-lg border border-red-200 bg-white p-5 shadow-sm">
        <div className="mb-4 h-2 w-16 rounded-full bg-red-500" />
        <h1 className="text-xl font-semibold leading-7">Каталог недоступен</h1>
        <p className="mt-2 text-sm leading-5 text-gray-600">
          Не удалось загрузить экран подарков.
        </p>
        <button
          type="button"
          onClick={reset}
          className="mt-5 h-10 rounded-md bg-emerald-700 px-4 text-sm font-medium text-white"
        >
          Повторить
        </button>
      </section>
    </main>
  );
}
