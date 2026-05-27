"use client";

export default function MiniappGiftsError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className="tg-screen flex min-h-screen items-center justify-center px-5 py-8">
      <section className="tg-card w-full max-w-sm rounded-xl border p-5">
        <div className="tg-error-bar mb-4 h-2 w-16 rounded-full" />
        <h1 className="tg-title text-xl font-semibold leading-7">Каталог недоступен</h1>
        <p className="tg-muted mt-2 text-sm leading-5">
          Не удалось загрузить экран призов.
        </p>
        <button
          type="button"
          onClick={reset}
          className="tg-primary-button mt-5 h-10 rounded-lg px-4 text-sm font-medium"
        >
          Повторить
        </button>
      </section>
    </main>
  );
}
