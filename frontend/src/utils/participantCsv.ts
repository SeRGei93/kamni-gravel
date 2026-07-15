import type { Participant } from '@/types';
import type {
  ParticipantColumn,
  ParticipantExportValue,
} from '@/components/participants/participantColumns';

const CSV_BOM = '\uFEFF';
const CSV_DELIMITER = ';';
const FORMULA_PREFIX = /^[=+\-@]/;

export interface ParticipantCsvExportInput {
  eventId: number;
  columns: ParticipantColumn[];
  participants: Participant[];
}

/**
 * Keeps untrusted strings as text when the CSV is opened in Excel.
 */
export function escapeSpreadsheetFormula(
  value: ParticipantExportValue,
): ParticipantExportValue {
  if (typeof value === 'string' && FORMULA_PREFIX.test(value)) {
    return `'${value}`;
  }
  return value;
}

function encodeCsvValue(value: ParticipantExportValue): string {
  if (value === null) {
    return '';
  }

  const text = String(escapeSpreadsheetFormula(value));
  if (/[;"\r\n]/.test(text)) {
    return `"${text.replaceAll('"', '""')}"`;
  }
  return text;
}

export function buildParticipantCsv(
  columns: ParticipantColumn[],
  participants: Participant[],
): string {
  const rows: ParticipantExportValue[][] = [
    columns.map((column) => column.label),
    ...participants.map((participant) =>
      columns.map((column) => column.exportValue(participant)),
    ),
  ];

  return `${CSV_BOM}${rows
    .map((row) => row.map(encodeCsvValue).join(CSV_DELIMITER))
    .join('\r\n')}\r\n`;
}

export function participantCsvFileName(eventId: number): string {
  return `participants-event-${eventId}.csv`;
}

export function isCurrentParticipantExportRequest(
  requestVersion: number,
  latestRequestVersion: number,
): boolean {
  return requestVersion === latestRequestVersion;
}

export function shouldSettleParticipantExportRequest(
  requestVersion: number,
  pendingRequestVersion: number | null,
  isMounted: boolean,
): boolean {
  return isMounted && requestVersion === pendingRequestVersion;
}

export function downloadParticipantCsv({
  eventId,
  columns,
  participants,
}: ParticipantCsvExportInput): void {
  const csv = buildParticipantCsv(columns, participants);
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');

  link.href = url;
  link.download = participantCsvFileName(eventId);
  link.style.display = 'none';
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 0);

  console.debug('[participants] export CSV prepared', {
    event_id: eventId,
    row_count: participants.length,
    column_keys: columns.map((column) => column.key),
  });
}
