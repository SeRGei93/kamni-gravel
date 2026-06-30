import MiniappSpinner from "@/components/miniapp/MiniappSpinner";

// Прелоадер показывается Next.js мгновенно при переходе на карточку приза
// (Suspense-fallback сегмента), пока грузится страница и её данные.
export default function MiniappGiftDetailLoading() {
  return (
    <main className="tg-screen flex min-h-screen items-center justify-center px-5 py-8">
      <section className="tg-card flex w-full max-w-sm flex-col items-center gap-3 rounded-xl border p-6">
        <MiniappSpinner size={28} />
        <p className="tg-muted text-sm leading-5">Загружаем приз</p>
      </section>
    </main>
  );
}
