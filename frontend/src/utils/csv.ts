export type CsvCellValue = string | number | boolean | null;

export interface CsvColumn<Item> {
  label: string;
  exportValue: (item: Item) => CsvCellValue;
}

const CSV_BOM = '\uFEFF';
const CSV_DELIMITER = ';';
const FORMULA_PREFIX = /^[=+\-@]/;

/** Keeps untrusted strings as text when the CSV is opened in Excel. */
export function escapeSpreadsheetFormula(value: CsvCellValue): CsvCellValue {
  if (typeof value === 'string' && FORMULA_PREFIX.test(value)) {
    return `'${value}`;
  }
  return value;
}

function encodeCsvValue(value: CsvCellValue): string {
  if (value === null) {
    return '';
  }

  const text = String(escapeSpreadsheetFormula(value));
  if (/[;"\r\n]/.test(text)) {
    return `"${text.replaceAll('"', '""')}"`;
  }
  return text;
}

export function buildCsv<Item>(
  columns: readonly CsvColumn<Item>[],
  items: readonly Item[]
): string {
  const rows: CsvCellValue[][] = [
    columns.map((column) => column.label),
    ...items.map((item) => columns.map((column) => column.exportValue(item))),
  ];

  return `${CSV_BOM}${rows
    .map((row) => row.map(encodeCsvValue).join(CSV_DELIMITER))
    .join('\r\n')}\r\n`;
}

export function downloadCsvFile(csv: string, fileName: string): void {
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');

  link.href = url;
  link.download = fileName;
  link.style.display = 'none';
  document.body.appendChild(link);
  link.click();
  link.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 0);
}
