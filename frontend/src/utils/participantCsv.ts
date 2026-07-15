import type { Participant } from '@/types';
import type { ParticipantColumn } from '@/components/participants/participantColumns';
import { buildCsv, downloadCsvFile } from './csv';

export { escapeSpreadsheetFormula } from './csv';

export interface ParticipantCsvExportInput {
  eventId: number;
  columns: ParticipantColumn[];
  participants: Participant[];
}

export function buildParticipantCsv(
  columns: ParticipantColumn[],
  participants: Participant[],
): string {
  return buildCsv(columns, participants);
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
  downloadCsvFile(buildParticipantCsv(columns, participants), participantCsvFileName(eventId));

  console.debug('[participants] export CSV prepared', {
    event_id: eventId,
    row_count: participants.length,
    column_keys: columns.map((column) => column.key),
  });
}
