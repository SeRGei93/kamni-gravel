// Маленький круговой прелоадер в цвете Telegram-темы. Используется при загрузке
// каталога (смена фильтров) и при переходе на детальную карточку приза.
export default function MiniappSpinner({
  size = 20,
  className = "",
}: {
  size?: number;
  className?: string;
}) {
  return (
    <span
      role="status"
      aria-label="Загрузка"
      className={`inline-block animate-spin rounded-full border-2 ${className}`}
      style={{
        width: size,
        height: size,
        borderColor: "var(--tg-button-color, #3390ec)",
        borderTopColor: "transparent",
      }}
    />
  );
}
