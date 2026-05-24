'use client';

import React from 'react';
import Badge from '../ui/badge/Badge';

interface BreakdownCardProps {
  title: string;
  data: Record<string, number>;
  total: number;
  labels: Record<string, string>;
}

export default function BreakdownCard({
  title,
  data,
  total,
  labels,
}: BreakdownCardProps) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] md:p-6">
      <h3 className="mb-4 text-lg font-semibold text-gray-800 dark:text-white">
        {title}
      </h3>
      <div className="space-y-3">
        {Object.entries(data).map(([key, value]) => {
          const percentage = total > 0 ? Math.round((value / total) * 100) : 0;
          return (
            <div key={key} className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {labels[key] || key}
                </span>
              </div>
              <div className="flex items-center gap-3">
                <div className="w-24 bg-gray-200 rounded-full h-2 dark:bg-gray-700">
                  <div
                    className="bg-brand-500 h-2 rounded-full"
                    style={{ width: `${percentage}%` }}
                  />
                </div>
                <span className="text-sm font-medium text-gray-800 dark:text-white w-12 text-right">
                  {value}
                </span>
                <Badge color="info" size="sm">
                  {percentage}%
                </Badge>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
