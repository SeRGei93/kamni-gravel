import type { Gift, GiftPlaceRule, PrizeGiftAssignment } from '@/types';

export type GiftPlaceRuleMode = 'none' | 'places' | 'last_n';

export interface GiftPlaceRange {
  start: number;
  end: number;
}

export interface GiftPlaceRuleFormState {
  mode: GiftPlaceRuleMode;
  placesInput: string;
  lastCount: string;
}

export function getGiftPlaceRuleFormState(gift: Gift): GiftPlaceRuleFormState {
  if (gift.place_rule?.type === 'places') {
    return {
      mode: 'places',
      placesInput: formatPlaceList(gift.place_rule.places ?? []),
      lastCount: '',
    };
  }

  if (gift.place_rule?.type === 'last_n') {
    return {
      mode: 'last_n',
      placesInput: '',
      lastCount: gift.place_rule.last_count ? String(gift.place_rule.last_count) : '',
    };
  }

  if (gift.place) {
    return {
      mode: 'places',
      placesInput: String(gift.place),
      lastCount: '',
    };
  }

  return {
    mode: 'none',
    placesInput: '',
    lastCount: '',
  };
}

export function buildGiftPlaceRuleFromForm(
  mode: GiftPlaceRuleMode,
  placesInput: string,
  lastCountInput: string
): GiftPlaceRule | null {
  if (mode === 'none') {
    return null;
  }

  if (mode === 'places') {
    return {
      type: 'places',
      places: parseGiftPlaceRulePlaces(placesInput),
    };
  }

  const lastCount = Number(lastCountInput);
  if (!Number.isSafeInteger(lastCount) || lastCount <= 0) {
    throw new Error('last_count_must_be_positive_integer');
  }

  return {
    type: 'last_n',
    last_count: lastCount,
  };
}

export function parseGiftPlaceRulePlaces(input: string): number[] {
  const trimmed = input.trim();
  if (!trimmed) {
    throw new Error('Введите места');
  }

  const places: number[] = [];
  for (const rawToken of trimmed.split(',')) {
    const token = rawToken.trim();
    if (!token) {
      throw new Error('Пустое значение места');
    }

    if (token.includes('-')) {
      const parts = token.split('-').map((part) => part.trim());
      if (parts.length !== 2 || !parts[0] || !parts[1]) {
        throw new Error('Некорректный диапазон мест');
      }
      const start = parsePositivePlace(parts[0]);
      const end = parsePositivePlace(parts[1]);
      if (start > end) {
        throw new Error('Начало диапазона больше конца');
      }
      for (let place = start; place <= end; place += 1) {
        places.push(place);
      }
      continue;
    }

    places.push(parsePositivePlace(token));
  }

  return Array.from(new Set(places)).sort((left, right) => left - right);
}

export function formatGiftPlaceRule(rule?: GiftPlaceRule | null): string {
  if (!rule) {
    return 'Без привязки';
  }

  if (rule.type === 'last_n') {
    return `${rule.last_count ?? 0} последних`;
  }

  const places = rule.places ?? [];
  if (places.length === 0) {
    return 'Без привязки';
  }

  return `Места ${formatPlaceList(places)}`;
}

export function formatPrizeAssignment(assignment: PrizeGiftAssignment): string {
  const target = assignment.target_rank
    ? `место ${assignment.target_rank}`
    : 'без привязки';
  const assigned = `выдано месту ${assignment.assigned_rank}`;

  if (assignment.is_fallback) {
    return `${target} -> ${assigned}`;
  }

  return assignment.target_rank ? target : assigned;
}

export function buildMissingGiftPlaceRanges(
  gifts: Gift[],
  targetPlace: number
): GiftPlaceRange[] {
  if (!Number.isSafeInteger(targetPlace) || targetPlace <= 0) {
    return [];
  }

  const occupiedPlaces = collectFixedGiftPlaces(gifts, targetPlace);
  const missingPlaces: number[] = [];
  for (let place = 1; place <= targetPlace; place += 1) {
    if (!occupiedPlaces.has(place)) {
      missingPlaces.push(place);
    }
  }

  return compactPlacesToRanges(missingPlaces);
}

export function getGiftFirstFixedPlace(gift: Gift): number | undefined {
  const places = getGiftFixedPlaces(gift);
  if (places.length === 0) {
    return undefined;
  }

  return Math.min(...places);
}

export function formatPlaceList(places: number[]): string {
  if (places.length === 0) {
    return '';
  }

  return compactPlacesToRanges(places)
    .map(formatPlaceRange)
    .join(', ');
}

export function formatPlaceRange(range: GiftPlaceRange): string {
  return range.start === range.end ? String(range.start) : `${range.start}-${range.end}`;
}

function collectFixedGiftPlaces(
  gifts: Gift[],
  targetPlace: number
): Set<number> {
  const occupiedPlaces = new Set<number>();

  for (const gift of gifts) {
    if (!isFixedPlaceGift(gift)) {
      continue;
    }

    for (const place of getGiftFixedPlaces(gift)) {
      if (place >= 1 && place <= targetPlace) {
        occupiedPlaces.add(place);
      }
    }
  }

  return occupiedPlaces;
}

function isFixedPlaceGift(gift: Gift): boolean {
  return gift.review_status === 'approved' && getGiftFixedPlaces(gift).length > 0;
}

function getGiftFixedPlaces(gift: Gift): number[] {
  if (gift.place_rule?.type === 'places') {
    return gift.place_rule.places ?? [];
  }

  if (gift.place) {
    return [gift.place];
  }

  return [];
}

function compactPlacesToRanges(places: number[]): GiftPlaceRange[] {
  const normalized = Array.from(new Set(places)).sort((left, right) => left - right);
  if (normalized.length === 0) {
    return [];
  }

  const ranges: GiftPlaceRange[] = [];
  let start = normalized[0];
  let previous = normalized[0];

  for (let index = 1; index <= normalized.length; index += 1) {
    const current = normalized[index];
    if (current === previous + 1) {
      previous = current;
      continue;
    }

    ranges.push({ start, end: previous });
    start = current;
    previous = current;
  }

  return ranges;
}

function parsePositivePlace(value: string): number {
  if (!/^\d+$/.test(value)) {
    throw new Error('Место должно быть числом');
  }

  const place = Number(value);
  if (!Number.isInteger(place) || place <= 0) {
    throw new Error('Место должно быть больше нуля');
  }

  return place;
}
