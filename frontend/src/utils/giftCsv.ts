import type { Gift } from '@/types';
import { formatGiftPlaceRule } from './giftPlaceRule';
import {
  formatManualRecipientSearchLabel,
  getManualGiftStatus,
} from './manualGiftAssignment';
import {
  buildCsv,
  downloadCsvFile,
  type CsvCellValue,
  type CsvColumn,
} from './csv';

export interface GiftCsvColumn extends CsvColumn<Gift> {
  key: string;
}

export interface GiftCsvExportInput {
  eventId: number;
  gifts: Gift[];
  assignedGiftIds: Set<number>;
}

function giftAuthor(gift: Gift): string {
  const name = `${gift.first_name ?? ''} ${gift.last_name ?? ''}`.trim();
  const username = `@${gift.username || `user${gift.user_id}`}`;
  return name ? `${name} (${username})` : username;
}

function giftCriteria(gift: Gift): CsvCellValue {
  if (!gift.criteria || gift.criteria.length === 0) {
    return 'Без критериев';
  }
  return gift.criteria.map((criteria) => criteria.name).join(', ');
}

function manualRecipient(gift: Gift): CsvCellValue {
  const recipient = gift.manual_assignment?.recipient;
  if (!recipient) {
    return null;
  }
  return formatManualRecipientSearchLabel(recipient.display_name, recipient.username);
}

export function getGiftCsvColumns(assignedGiftIds: Set<number>): GiftCsvColumn[] {
  return [
    { key: 'id', label: 'ID', exportValue: (gift) => gift.id },
    { key: 'description', label: 'Описание', exportValue: (gift) => gift.description },
    { key: 'author', label: 'От кого', exportValue: giftAuthor },
    {
      key: 'review_status',
      label: 'Статус',
      exportValue: (gift) =>
        gift.review_status === 'pending_review' ? 'Новый / на проверке' : 'Проверен',
    },
    {
      key: 'place_rule',
      label: 'Правило',
      exportValue: (gift) =>
        formatGiftPlaceRule(
          gift.place_rule ?? (gift.place ? { type: 'places', places: [gift.place] } : null)
        ),
    },
    { key: 'criteria', label: 'Критерии', exportValue: giftCriteria },
    {
      key: 'distribution',
      label: 'Распределение',
      exportValue: (gift) => getManualGiftStatus(gift, assignedGiftIds).label,
    },
    { key: 'recipient', label: 'Получатель', exportValue: manualRecipient },
    {
      key: 'created_at',
      label: 'Дата',
      exportValue: (gift) => new Date(gift.created_at).toLocaleDateString('ru-RU'),
    },
  ];
}

export function buildGiftCsv(gifts: Gift[], assignedGiftIds: Set<number>): string {
  return buildCsv(getGiftCsvColumns(assignedGiftIds), gifts);
}

export function giftCsvFileName(eventId: number): string {
  return `gifts-event-${eventId}.csv`;
}

export function isCurrentGiftExportRequest(
  requestVersion: number,
  latestRequestVersion: number
): boolean {
  return requestVersion === latestRequestVersion;
}

export function shouldSettleGiftExportRequest(
  requestVersion: number,
  latestRequestVersion: number,
  isMounted: boolean
): boolean {
  return isMounted && requestVersion === latestRequestVersion;
}

export function downloadGiftCsv({
  eventId,
  gifts,
  assignedGiftIds,
}: GiftCsvExportInput): void {
  const columns = getGiftCsvColumns(assignedGiftIds);
  downloadCsvFile(buildCsv(columns, gifts), giftCsvFileName(eventId));

  console.debug('[gifts] export CSV prepared', {
    event_id: eventId,
    row_count: gifts.length,
    column_keys: columns.map((column) => column.key),
  });
}
