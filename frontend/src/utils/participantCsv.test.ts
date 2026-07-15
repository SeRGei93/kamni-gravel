import { describe, expect, it } from 'vitest';
import type { Participant } from '@/types';
import type { ParticipantColumn } from '@/components/participants/participantColumns';
import {
  buildParticipantCsv,
  escapeSpreadsheetFormula,
  isCurrentParticipantExportRequest,
  participantCsvFileName,
  shouldSettleParticipantExportRequest,
} from './participantCsv';

const participant: Participant = {
  id: 1,
  user_id: 42,
  username: '=formula',
  first_name: 'Анна',
  last_name: 'Иванова',
  event_id: 77,
  bike_type: 'gravel',
  gender: 'female',
  status: 'active',
  is_finished: true,
  has_gift: false,
  prizes_count: 0,
  registered_at: '2026-07-15T08:00:00Z',
};

const columns: ParticipantColumn[] = [
  {
    key: 'username',
    label: 'Username',
    defaultVisible: true,
    render: () => null,
    exportValue: (item) => item.username,
  },
  {
    key: 'notes',
    label: 'Заметки',
    defaultVisible: true,
    render: () => null,
    exportValue: () => 'Старт; "финиш"\nзавершён',
  },
  {
    key: 'place',
    label: 'Место',
    defaultVisible: true,
    render: () => null,
    exportValue: () => 5,
  },
  {
    key: 'empty',
    label: 'Пусто',
    defaultVisible: true,
    render: () => null,
    exportValue: () => null,
  },
];

describe('participant CSV export', () => {
  it('keeps selected columns, values, CSV escaping and formula protection', () => {
    expect(buildParticipantCsv(columns, [participant])).toBe(
      '\uFEFFUsername;Заметки;Место;Пусто\r\n\'=formula;"Старт; ""финиш""\nзавершён";5;\r\n',
    );
  });

  it.each(['=SUM(A1:A2)', '+1+1', '-1+1', '@cmd'])(
    'escapes formula-like text %s',
    (value) => {
      expect(escapeSpreadsheetFormula(value)).toBe(`'${value}`);
    },
  );

  it('preserves numeric values and empty cells', () => {
    expect(escapeSpreadsheetFormula(12)).toBe(12);
    expect(escapeSpreadsheetFormula(null)).toBeNull();
  });

  it('uses the expected file name and rejects stale export versions', () => {
    expect(participantCsvFileName(77)).toBe('participants-event-77.csv');
    expect(isCurrentParticipantExportRequest(3, 4)).toBe(false);
    expect(isCurrentParticipantExportRequest(4, 4)).toBe(true);
  });

  it('does not settle export state after the page unmounts', () => {
    expect(shouldSettleParticipantExportRequest(4, 4, true)).toBe(true);
    expect(shouldSettleParticipantExportRequest(4, 4, false)).toBe(false);
    expect(shouldSettleParticipantExportRequest(4, 5, true)).toBe(false);
  });
});
