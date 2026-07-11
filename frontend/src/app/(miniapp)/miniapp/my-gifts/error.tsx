"use client";

export default function MiniappMyGiftsError({ reset }: { error: Error; reset: () => void }) {
  return (
    <main className="tg-screen flex min-h-screen items-center justify-center px-5 py-8">
      <section className="tg-card w-full max-w-sm rounded-xl border p-5 text-center">
        <h1 className="tg-title text-lg font-semibold">Мои призы недоступны</h1>
        <p className="tg-muted mt-2 text-sm">Произошла непредвиденная ошибка.</p>
        <button
          type="button"
          onClick={reset}
          className="tg-link-button mt-4 rounded-lg border px-3 py-2 text-sm font-semibold"
        >
          Повторить
        </button>
      </section>
    </main>
  );
}
