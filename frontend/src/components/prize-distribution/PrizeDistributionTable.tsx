'use client';

import { Table, TableBody, TableCell, TableHeader, TableRow } from '@/components/ui/table';
import type { PrizeDistribution } from '@/types';

import type { PrizeDistributionColumn } from './prizeDistributionColumns';

interface PrizeDistributionTableProps {
  distribution: PrizeDistribution[];
  columns: PrizeDistributionColumn[];
  isLoading?: boolean;
}

function alignClass(align?: PrizeDistributionColumn['align']): string {
  if (align === 'end') return 'text-end';
  if (align === 'center') return 'text-center';
  return 'text-start';
}

export default function PrizeDistributionTable({
  distribution,
  columns,
  isLoading,
}: PrizeDistributionTableProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">Загрузка распределения...</div>
      </div>
    );
  }

  if (distribution.length === 0) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-500 dark:text-gray-400">
          В выбранном срезе нет участников
        </div>
      </div>
    );
  }

  return (
    <div className="w-full max-w-full overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-white/[0.05] dark:bg-white/[0.03]">
      <div className="w-full max-w-full overflow-x-auto">
        <Table>
          <TableHeader className="border-b border-gray-100 dark:border-white/[0.05]">
            <TableRow>
              {columns.map((column) => (
                <TableCell
                  key={column.key}
                  isHeader
                  className={`whitespace-nowrap px-5 py-3 font-medium text-gray-500 ${alignClass(column.align)} text-theme-xs dark:text-gray-400`}
                >
                  {column.label}
                </TableCell>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody className="divide-y divide-gray-100 dark:divide-white/[0.05]">
            {distribution.map((row) => (
              <TableRow
                key={row.participant_id}
                className="hover:bg-gray-50 dark:hover:bg-white/5"
              >
                {columns.map((column) => (
                  <TableCell
                    key={column.key}
                    className={`whitespace-nowrap px-5 py-4 ${alignClass(column.align)}`}
                  >
                    {column.render(row)}
                  </TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
