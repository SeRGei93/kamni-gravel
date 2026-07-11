type SearchValue = string | number | null | undefined;

function normalizeSearchValue(value: SearchValue): string {
  return String(value ?? '')
    .trim()
    .toLowerCase()
    .replace(/^@+/, '');
}

/**
 * Сопоставляет запрос с отображаемыми полями участника.
 *
 * Username в интерфейсе показывается как `@username`, а API возвращает его
 * без префикса. Нормализация делает оба варианта запроса эквивалентными.
 */
export function matchesSearchQuery(
  query: string,
  values: readonly SearchValue[]
): boolean {
  const normalizedQuery = normalizeSearchValue(query);
  if (!normalizedQuery) {
    return true;
  }

  return values.some((value) =>
    normalizeSearchValue(value).includes(normalizedQuery)
  );
}
