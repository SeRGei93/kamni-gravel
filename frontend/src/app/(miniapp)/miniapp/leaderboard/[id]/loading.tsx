import MiniappSpinner from "@/components/miniapp/MiniappSpinner";

export default function MiniappLeaderboardDetailLoading() {
  return (
    <main className="tg-screen flex min-h-[60vh] items-center justify-center px-5 py-8">
      <section className="tg-card flex w-full max-w-sm flex-col items-center gap-3 rounded-xl border p-6">
        <MiniappSpinner size={28} />
        <p className="tg-muted text-sm leading-5">Загружаем результат</p>
      </section>
    </main>
  );
}
