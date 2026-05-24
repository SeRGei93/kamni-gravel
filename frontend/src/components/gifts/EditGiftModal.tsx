'use client';

import React, { useState, useEffect } from 'react';
import { Modal } from '@/components/ui/modal';
import Label from '@/components/form/Label';
import TextArea from '@/components/form/input/TextArea';
import InputField from '@/components/form/input/InputField';
import Select from '@/components/form/Select';
import Button from '@/components/ui/button/Button';
import Badge from '@/components/ui/badge/Badge';
import { giftsApi } from '@/api/gifts';
import { criteriaApi } from '@/api/criteria';
import { getCriteriaColor } from '@/utils/criteria';
import type { Gift, Criteria, UpdateGiftRequest, GenderFilter, BikeTypeFilter } from '@/types';
import { GENDER_OPTIONS, BIKE_TYPE_OPTIONS } from '@/constants';

interface EditGiftModalProps {
  isOpen: boolean;
  onClose: () => void;
  gift: Gift;
  onSuccess?: () => void;
}

export default function EditGiftModal({
  isOpen,
  onClose,
  gift,
  onSuccess,
}: EditGiftModalProps) {
  const [description, setDescription] = useState('');
  const [genderFilter, setGenderFilter] = useState<GenderFilter>('all');
  const [bikeTypeFilter, setBikeTypeFilter] = useState<BikeTypeFilter>('all');
  const [place, setPlace] = useState('');
  const [allCriteria, setAllCriteria] = useState<Criteria[]>([]);
  const [selectedCriteriaIds, setSelectedCriteriaIds] = useState<number[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Загружаем данные при открытии
  useEffect(() => {
    if (isOpen && gift) {
      setDescription(gift.description);
      setGenderFilter(gift.gender_filter || 'all');
      setBikeTypeFilter(gift.bike_type_filter || 'all');
      setPlace(gift.place?.toString() || '');
      setSelectedCriteriaIds(gift.criteria?.map((c) => c.id) || []);
      loadCriteria();
    }
  }, [isOpen, gift]);

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

  const handleClose = () => {
    setError(null);
    onClose();
  };

  const toggleCriteria = (criteriaId: number) => {
    setSelectedCriteriaIds((prev) =>
      prev.includes(criteriaId)
        ? prev.filter((id) => id !== criteriaId)
        : [...prev, criteriaId]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!description.trim()) {
      setError('Введите описание подарка');
      return;
    }

    try {
      setIsSubmitting(true);
      setError(null);

      const data: UpdateGiftRequest = {
        description: description.trim(),
        gender_filter: genderFilter || 'all',
        bike_type_filter: bikeTypeFilter || 'all',
        place: place ? parseInt(place, 10) : null,
        criteria_ids: selectedCriteriaIds,
      };

      await giftsApi.update(gift.id, data);

      if (onSuccess) {
        onSuccess();
      }

      handleClose();
    } catch (err) {
      console.error('Failed to update gift:', err);
      setError('Ошибка обновления подарка');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} className="max-w-2xl m-4">
      <div className="no-scrollbar relative w-full max-w-2xl overflow-y-auto rounded-3xl bg-white p-4 dark:bg-gray-900 lg:p-11">
        <div className="px-2 pr-14">
          <h4 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white/90">
            Редактировать подарок
          </h4>
          <p className="mb-6 text-sm text-gray-500 dark:text-gray-400 lg:mb-7">
            От: {gift.first_name} {gift.last_name} (@{gift.username || `user${gift.user_id}`})
          </p>
        </div>

        {error && (
          <div className="mb-4 rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
            <p className="text-error-600 dark:text-error-400">{error}</p>
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="text-gray-500 dark:text-gray-400">Загрузка...</div>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="px-2">
            <div className="space-y-6">
              <div>
                <Label>
                  Описание подарка <span className="text-error-500">*</span>
                </Label>
                <TextArea
                  placeholder="Опишите подарок..."
                  value={description}
                  onChange={setDescription}
                  rows={4}
                />
              </div>

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div>
                  <Label>Фильтр по полу</Label>
                  <Select
                    options={GENDER_OPTIONS}
                    defaultValue={genderFilter}
                    onChange={(value) => setGenderFilter(value as GenderFilter)}
                  />
                </div>
                <div>
                  <Label>Тип велосипеда</Label>
                  <Select
                    options={BIKE_TYPE_OPTIONS}
                    defaultValue={bikeTypeFilter}
                    onChange={(value) => setBikeTypeFilter(value as BikeTypeFilter)}
                  />
                </div>
                <div>
                  <Label>Место (позиция)</Label>
                  <InputField
                    type="number"
                    placeholder="Например: 1, 2, 3"
                    value={place}
                    onChange={(e) => setPlace(e.target.value)}
                  />
                </div>
              </div>

              <div>
                <Label>Критерии</Label>
                <p className="mb-3 text-xs text-gray-500 dark:text-gray-400">
                  Выберите критерии, которым соответствует подарок
                </p>
                {allCriteria.length > 0 ? (
                  <div className="flex flex-wrap gap-2">
                    {allCriteria.map((criteria) => {
                      const isSelected = selectedCriteriaIds.includes(criteria.id);
                      return (
                        <button
                          key={criteria.id}
                          type="button"
                          onClick={() => toggleCriteria(criteria.id)}
                          className={`rounded-full px-3 py-1.5 text-sm font-medium transition-all ${
                            isSelected
                              ? 'ring-2 ring-brand-500 ring-offset-2 dark:ring-offset-gray-900'
                              : 'opacity-60 hover:opacity-100'
                          }`}
                        >
                          <Badge
                            color={getCriteriaColor(criteria.criteria_type)}
                            size="sm"
                          >
                            {criteria.name}
                          </Badge>
                        </button>
                      );
                    })}
                  </div>
                ) : (
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    Нет доступных критериев
                  </p>
                )}
              </div>

              {gift.attachments && gift.attachments.length > 0 && (
                <div>
                  <Label>Вложения</Label>
                  <p className="text-sm text-gray-500 dark:text-gray-400">
                    Прикреплено файлов: {gift.attachments.length}
                  </p>
                </div>
              )}

              <div className="flex items-center gap-3 justify-end">
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleClose}
                  disabled={isSubmitting}
                >
                  Отмена
                </Button>
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting ? 'Сохранение...' : 'Сохранить'}
                </Button>
              </div>
            </div>
          </form>
        )}
      </div>
    </Modal>
  );
}
