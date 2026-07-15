import { describe, expect, it } from 'vitest';
import type { Gift } from '@/types';
import {
  buildGiftCsv,
  giftCsvFileName,
  isCurrentGiftExportRequest,
  shouldSettleGiftExportRequest,
} from './giftCsv';

const manualGift: Gift = {
  id: 7,
  user_id: 11,
  username: 'donor',
  first_name: 'Анна',
  last_name: 'Иванова',
  event_id: 77,
  description: '=Шлем; "карбон"',
  review_status: 'approved',
  place_rule: { type: 'places', places: [1, 2] },
  criteria: [
    {
      id: 1,
      name: 'Скорость',
      description: 'Быстрый заезд',
      criteria_type: 'speed',
      created_at: '2026-07-16T12:00:00Z',
    },
  ],
  manual_distribution: true,
  manual_assignment: {
    id: 7,
    event_id: 77,
    description: 'Шлем',
    review_status: 'approved',
    manual_distribution: true,
    recipient: { id: 22, display_name: 'Alex', username: 'alex', status: 'active' },
    created_at: '2026-07-16T12:00:00Z',
  },
  created_at: '2026-07-16T12:00:00Z',
};

describe('gift CSV export', () => {
  it('exports table fields with a protected manual recipient and CSV escaping', () => {
    const createdAt = new Date(manualGift.created_at).toLocaleDateString('ru-RU');

    expect(buildGiftCsv([manualGift], new Set())).toBe(
      `\uFEFFID;Описание;От кого;Статус;Правило;Критерии;Распределение;Получатель;Дата\r\n7;"'=Шлем; ""карбон""";Анна Иванова (@donor);Проверен;Места 1-2;Скорость;Ручной: получатель назначен;Alex (@alex);${createdAt}\r\n`
    );
  });

  it('leaves the recipient cell empty for an unassigned manual gift', () => {
    const unassignedGift: Gift = {
      ...manualGift,
      id: 8,
      manual_assignment: undefined,
    };

    expect(buildGiftCsv([unassignedGift], new Set())).toContain(
      'Ручной: ожидает назначения;;'
    );
  });

  it('uses the expected file name and protects export state from stale requests', () => {
    expect(giftCsvFileName(77)).toBe('gifts-event-77.csv');
    expect(isCurrentGiftExportRequest(3, 4)).toBe(false);
    expect(isCurrentGiftExportRequest(4, 4)).toBe(true);
    expect(shouldSettleGiftExportRequest(4, 4, true)).toBe(true);
    expect(shouldSettleGiftExportRequest(4, 4, false)).toBe(false);
    expect(shouldSettleGiftExportRequest(4, 5, true)).toBe(false);
  });
});
