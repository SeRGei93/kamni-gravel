export default function MiniappGiftsLoading() {
  return (
    <main className="tg-screen flex min-h-screen items-center justify-center px-5 py-8">
      <section className="tg-card w-full max-w-sm rounded-xl border p-5">
        <div className="tg-accent-bar mb-4 h-2 w-16 animate-pulse rounded-full" />
        <h1 className="tg-title text-xl font-semibold leading-7">Каталог призов</h1>
        <p className="tg-muted mt-2 text-sm leading-5">Загружаем призы</p>
      </section>
    </main>
  );
}
