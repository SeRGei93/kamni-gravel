const MINSK_OFFSET_MINUTES = 3 * 60;
const MINSK_OFFSET_LABEL = 'Минск UTC+3';

function pad(value: number): string {
  return value.toString().padStart(2, '0');
}

function toMinskShiftedDate(value: string): Date | null {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return new Date(date.getTime() + MINSK_OFFSET_MINUTES * 60 * 1000);
}

export function toMinskDateTimeInput(value?: string | null): string {
  if (!value) {
    return '';
  }

  const shifted = toMinskShiftedDate(value);
  if (!shifted) {
    return '';
  }

  return [
    shifted.getUTCFullYear(),
    pad(shifted.getUTCMonth() + 1),
    pad(shifted.getUTCDate()),
  ].join('-') + `T${pad(shifted.getUTCHours())}:${pad(shifted.getUTCMinutes())}`;
}

export function fromMinskDateTimeInput(value: string): string | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }

  if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) {
    return `${trimmed}T00:00:00+03:00`;
  }

  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(trimmed)) {
    return `${trimmed}:00+03:00`;
  }

  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}$/.test(trimmed)) {
    return `${trimmed}+03:00`;
  }

  return trimmed;
}

export function formatMinskDateTime(value?: string | null): string {
  const inputValue = toMinskDateTimeInput(value);
  if (!inputValue) {
    return '-';
  }

  const [datePart, timePart] = inputValue.split('T');
  const [year, month, day] = datePart.split('-');

  return `${day}.${month}.${year} ${timePart} (${MINSK_OFFSET_LABEL})`;
}

export { MINSK_OFFSET_LABEL };
