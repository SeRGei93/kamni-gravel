import { redirect } from "next/navigation";

// Каноничный вход в Mini App: по умолчанию открываем лидерборд.
// Кнопка бота может указывать на /miniapp — пользователь попадёт на лидерборд,
// а на каталог призов переключится табом.
export default function MiniappIndexPage() {
  redirect("/miniapp/leaderboard");
}
