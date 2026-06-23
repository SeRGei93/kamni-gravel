'use client';

import React, { useState, useEffect } from 'react';
import { resultsApi } from '@/api/results';
import { criteriaApi } from '@/api/criteria';
import { getCriteriaColor } from '@/utils/criteria';
import Input from '@/components/form/input/InputField';
import type { Result, Criteria } from '@/types';

interface ResultCriteriaManagerProps {
  result: Result | null;
  onUpdate?: () => void;
  // disabled блокирует изменение критериев, когда запись редактирует другой
  // администратор (лок участника). Сервер дополнительно отклонит запись (409).
  disabled?: boolean;
}

export default function ResultCriteriaManager({
  result,
  onUpdate,
  disabled = false,
}: ResultCriteriaManagerProps) {
  const [allCriteria, setAllCriteria] = useState<Criteria[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [pendingIds, setPendingIds] = useState<Set<number>>(new Set());
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState('');

  useEffect(() => {
    loadCriteria();
  }, []);

  // Локальный стейт выбранных критериев — источник правды для чекбоксов.
  // Синхронизируем из пропа при смене результата (первичная загрузка / полный рефреш).
  useEffect(() => {
    setSelectedIds(new Set(result?.criteria?.map((c) => c.id) ?? []));
  }, [result]);

  const loadCriteria = async () => {
    try {
      setIsLoading(true);
      const response = await criteriaApi.getAll();
      setAllCriteria(response.criteria);
    } catch (err) {
      console.error('Failed to load criteria:', err);
      setError('Ошибка загрузки критериев');
    } finally {
      setIsLoading(false);
    }
  };

  const handleToggleCriteria = async (criteriaId: number, isChecked: boolean) => {
    // Гард против повторного клика по этому же критерию, пока запрос в полёте,
    // и против правок при чужом локе участника.
    if (!result || disabled || pendingIds.has(criteriaId)) return;

    // Оптимистично обновляем чекбокс сразу — без ожидания ответа сервера.
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (isChecked) next.add(criteriaId);
      else next.delete(criteriaId);
      return next;
    });
    setPendingIds((prev) => new Set(prev).add(criteriaId));
    setError(null);

    try {
      if (isChecked) {
        await resultsApi.addCriteria(result.id, criteriaId);
      } else {
        await resultsApi.removeCriteria(result.id, criteriaId);
      }

      // Успех — обновляем только блок «Подобранные призы» в родителе (без перезагрузки страницы).
      if (onUpdate) {
        onUpdate();
      }
    } catch (err) {
      console.error('Failed to toggle criteria:', err);
      // Откатываем именно это изменение (безопасно при параллельных тогглах).
      setSelectedIds((prev) => {
        const next = new Set(prev);
        if (isChecked) next.delete(criteriaId);
        else next.add(criteriaId);
        return next;
      });
      setError(isChecked ? 'Ошибка добавления критерия' : 'Ошибка удаления критерия');
    } finally {
      setPendingIds((prev) => {
        const next = new Set(prev);
        next.delete(criteriaId);
        return next;
      });
    }
  };

  if (!result) {
    return (
      <p className="text-sm text-gray-500 dark:text-gray-400">
        Нет результата для добавления критериев
      </p>
    );
  }

  if (isLoading) {
    return (
      <p className="text-sm text-gray-500 dark:text-gray-400">
        Загрузка критериев...
      </p>
    );
  }

  const isCriteriaSelected = (criteriaId: number) => selectedIds.has(criteriaId);

  const query = search.trim().toLowerCase();
  const filteredCriteria = query
    ? allCriteria.filter(
        (c) =>
          c.name.toLowerCase().includes(query) ||
          (c.description?.toLowerCase().includes(query) ?? false)
      )
    : allCriteria;

  return (
    <div className="space-y-4">
      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-3 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-sm text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {allCriteria.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Нет доступных критериев
        </p>
      ) : (
        <>
          <Input
            type="text"
            placeholder="Поиск по критериям..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />

          {filteredCriteria.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Ничего не найдено
            </p>
          ) : (
            <div className="space-y-2">
              {filteredCriteria.map((criteria) => {
            const isSelected = isCriteriaSelected(criteria.id);
            const isPending = pendingIds.has(criteria.id);
            const color = getCriteriaColor(criteria.criteria_type);

            return (
              <label
                key={criteria.id}
                className={`flex items-center gap-3 rounded-lg border p-3 transition-colors ${
                  disabled ? 'cursor-not-allowed' : 'cursor-pointer'
                } ${
                  isSelected
                    ? 'border-brand-500 bg-brand-50 dark:border-brand-600 dark:bg-brand-900/20'
                    : 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'
                } ${isPending ? 'opacity-60 cursor-progress' : ''} ${
                  disabled ? 'opacity-60' : ''
                }`}
              >
                <input
                  type="checkbox"
                  checked={isSelected}
                  onChange={(e) => handleToggleCriteria(criteria.id, e.target.checked)}
                  disabled={isPending || disabled}
                  className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500 dark:border-gray-600 dark:bg-gray-700"
                />
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-800 dark:text-white/90">
                      {criteria.name}
                    </span>
                    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                      color === 'success' ? 'bg-success-100 text-success-700 dark:bg-success-900/30 dark:text-success-400' :
                      color === 'warning' ? 'bg-warning-100 text-warning-700 dark:bg-warning-900/30 dark:text-warning-400' :
                      color === 'info' ? 'bg-info-100 text-info-700 dark:bg-info-900/30 dark:text-info-400' :
                      'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400'
                    }`}>
                      {criteria.criteria_type}
                    </span>
                  </div>
                  {criteria.description && (
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      {criteria.description}
                    </p>
                  )}
                </div>
              </label>
            );
              })}
            </div>
          )}
        </>
      )}
    </div>
  );
}
