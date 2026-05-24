'use client';

import React from 'react';
import Badge from '../ui/badge/Badge';
import { ArrowUpIcon } from '@/icons';

interface StatCardProps {
  title: string;
  value: number;
  icon: React.ReactNode;
  trend?: {
    value: number;
    isPositive: boolean;
  };
  color?: 'primary' | 'success' | 'error' | 'warning' | 'info';
}

export default function StatCard({
  title,
  value,
  icon,
  trend,
  color = 'primary',
}: StatCardProps) {
  return (
    <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] md:p-6">
      <div className={`flex items-center justify-center w-12 h-12 bg-gray-100 rounded-xl dark:bg-gray-800`}>
        <div className="text-gray-800 dark:text-white/90">{icon}</div>
      </div>

      <div className="flex items-end justify-between mt-5">
        <div>
          <span className="text-sm text-gray-500 dark:text-gray-400">
            {title}
          </span>
          <h4 className="mt-2 font-bold text-gray-800 text-title-sm dark:text-white/90">
            {value.toLocaleString('ru-RU')}
          </h4>
        </div>
        {trend && (
          <Badge color={trend.isPositive ? 'success' : 'error'}>
            <ArrowUpIcon className={trend.isPositive ? '' : 'rotate-180'} />
            {Math.abs(trend.value).toFixed(2)}%
          </Badge>
        )}
      </div>
    </div>
  );
}
