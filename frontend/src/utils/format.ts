// Утилиты форматирования скорости и дистанции (русская локаль, запятая-разделитель)

/**
 * Форматирует скорость в км/ч с одной цифрой после запятой: 13.89 → "13,9 км/ч".
 * Возвращает "" для undefined/null/нечисловых значений.
 */
export function formatSpeed(kmh?: number | null): string {
  if (kmh === undefined || kmh === null || Number.isNaN(kmh)) {
    return '';
  }
  return `${kmh.toLocaleString('ru-RU', {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })} км/ч`;
}

/**
 * Конвертирует метры в километры для отображения: 202000 → 202.
 * Возвращает undefined, если значение не задано.
 */
export function metersToKm(meters?: number | null): number | undefined {
  if (meters === undefined || meters === null || Number.isNaN(meters)) {
    return undefined;
  }
  return meters / 1000;
}

/**
 * Конвертирует километры в метры (округление) для отправки на бэкенд: 202.3 → 202300.
 * Возвращает undefined, если значение не задано.
 */
export function kmToMeters(km?: number | null): number | undefined {
  if (km === undefined || km === null || Number.isNaN(km)) {
    return undefined;
  }
  return Math.round(km * 1000);
}

/**
 * Форматирует дистанцию (в метрах) как километры: 202300 → "202,3 км".
 * Возвращает "" для undefined/null.
 */
export function formatDistanceKm(meters?: number | null): string {
  const km = metersToKm(meters);
  if (km === undefined) {
    return '';
  }
  return `${km.toLocaleString('ru-RU', {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })} км`;
}
