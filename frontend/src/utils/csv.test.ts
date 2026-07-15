import { describe, expect, it } from 'vitest';
import { buildCsv, escapeSpreadsheetFormula, type CsvColumn } from './csv';

type Row = {
  text: string;
  count: number | null;
};

const columns: CsvColumn<Row>[] = [
  { label: 'Текст', exportValue: (row) => row.text },
  { label: 'Количество', exportValue: (row) => row.count },
];

describe('CSV helpers', () => {
  it('serializes UTF-8 CSV with Excel-safe values and escaped content', () => {
    expect(buildCsv(columns, [{ text: '=SUM(A1:A2); "готово"\nстрока', count: 3 }])).toBe(
      '\uFEFFТекст;Количество\r\n"\'=SUM(A1:A2); ""готово""\nстрока";3\r\n'
    );
  });

  it.each(['=SUM(A1:A2)', '+1+1', '-1+1', '@cmd'])(
    'escapes formula-like text %s',
    (value) => {
      expect(escapeSpreadsheetFormula(value)).toBe(`'${value}`);
    }
  );

  it('preserves numeric values and empty cells', () => {
    expect(escapeSpreadsheetFormula(12)).toBe(12);
    expect(escapeSpreadsheetFormula(null)).toBeNull();
  });
});
