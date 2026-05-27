'use client';

import { useEffect, useState } from 'react';
import Label from '@/components/form/Label';
import TextArea from '@/components/form/input/TextArea';
import InputField from '@/components/form/input/InputField';
import Select from '@/components/form/Select';
import Button from '@/components/ui/button/Button';
import Badge from '@/components/ui/badge/Badge';
import { getCriteriaColor } from '@/utils/criteria';
import type {
  BikeTypeFilter,
  Criteria,
  GenderFilter,
  Gift,
  GiftReviewStatus,
  UpdateGiftRequest,
} from '@/types';
import {
  BIKE_TYPE_OPTIONS,
  GENDER_OPTIONS,
  GIFT_REVIEW_STATUS_OPTIONS,
} from '@/constants';

interface GiftEditFormProps {
  gift: Gift;
  criteria: Criteria[];
  onSubmit: (data: UpdateGiftRequest) => Promise<void>;
  onCancel: () => void;
}

function normalizePlace(value: string): number | null {
  const trimmed = value.trim();

  if (!trimmed) {
    return null;
  }

  if (!/^\d+$/.test(trimmed)) {
    throw new Error('place_must_be_positive_integer');
  }

  const place = Number(trimmed);
  if (!Number.isSafeInteger(place) || place <= 0) {
    throw new Error('place_must_be_positive_integer');
  }

  return place;
}

export default function GiftEditForm({
  gift,
  criteria,
  onSubmit,
  onCancel,
}: GiftEditFormProps) {
  const [description, setDescription] = useState(gift.description);
  const [genderFilter, setGenderFilter] = useState<GenderFilter>(
    gift.gender_filter || 'all'
  );
  const [bikeTypeFilter, setBikeTypeFilter] = useState<BikeTypeFilter>(
    gift.bike_type_filter || 'all'
  );
  const [reviewStatus, setReviewStatus] = useState<GiftReviewStatus>(
    gift.review_status
  );
  const [place, setPlace] = useState(gift.place?.toString() || '');
  const [selectedCriteriaIds, setSelectedCriteriaIds] = useState<number[]>(
    gift.criteria?.map((item) => item.id) || []
  );
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setDescription(gift.description);
    setGenderFilter(gift.gender_filter || 'all');
    setBikeTypeFilter(gift.bike_type_filter || 'all');
    setReviewStatus(gift.review_status);
    setPlace(gift.place?.toString() || '');
    setSelectedCriteriaIds(gift.criteria?.map((item) => item.id) || []);
    setError(null);
  }, [gift]);

  const toggleCriteria = (criteriaId: number) => {
    setSelectedCriteriaIds((current) =>
      current.includes(criteriaId)
        ? current.filter((id) => id !== criteriaId)
        : [...current, criteriaId]
    );
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();

    const trimmedDescription = description.trim();
    if (!trimmedDescription) {
      setError('Введите описание приза');
      return;
    }

    let normalizedPlace: number | null;
    try {
      normalizedPlace = normalizePlace(place);
    } catch {
      setError('Место должно быть положительным целым числом');
      return;
    }

    try {
      setIsSubmitting(true);
      setError(null);

      await onSubmit({
        description: trimmedDescription,
        gender_filter: genderFilter || 'all',
        bike_type_filter: bikeTypeFilter || 'all',
        review_status: reviewStatus,
        place: normalizedPlace,
        criteria_ids: selectedCriteriaIds,
      });
    } catch (err) {
      console.error('Failed to update gift:', {
        gift_id: gift.id,
        operation: 'update_gift',
        error: err,
      });
      setError('Ошибка обновления приза');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      <div>
        <Label>
          Описание приза <span className="text-error-500">*</span>
        </Label>
        <TextArea
          placeholder="Опишите приз..."
          value={description}
          onChange={setDescription}
          rows={5}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
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
            min="1"
            step={1}
            placeholder="Например: 1, 2, 3"
            value={place}
            onChange={(event) => setPlace(event.target.value)}
          />
        </div>
        <div>
          <Label>Статус проверки</Label>
          <Select
            options={GIFT_REVIEW_STATUS_OPTIONS}
            defaultValue={reviewStatus}
            onChange={(value) => setReviewStatus(value as GiftReviewStatus)}
          />
        </div>
      </div>

      <div>
        <Label>Критерии</Label>
        <p className="mb-3 text-xs text-gray-500 dark:text-gray-400">
          Выберите критерии, которым соответствует приз
        </p>
        {criteria.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {criteria.map((criterion) => {
              const isSelected = selectedCriteriaIds.includes(criterion.id);
              return (
                <button
                  key={criterion.id}
                  type="button"
                  aria-pressed={isSelected}
                  onClick={() => toggleCriteria(criterion.id)}
                  className={`rounded-full px-1 py-1 text-sm font-medium transition ${
                    isSelected
                      ? 'ring-2 ring-brand-500 ring-offset-2 dark:ring-offset-gray-900'
                      : 'opacity-60 hover:opacity-100'
                  }`}
                >
                  <Badge
                    color={getCriteriaColor(criterion.criteria_type)}
                    size="sm"
                  >
                    {criterion.name}
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

      <div className="flex flex-col-reverse gap-3 border-t border-gray-100 pt-5 dark:border-white/[0.05] sm:flex-row sm:justify-end">
        <Button
          type="button"
          variant="outline"
          onClick={onCancel}
          disabled={isSubmitting}
        >
          Отмена
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Сохранение...' : 'Сохранить'}
        </Button>
      </div>
    </form>
  );
}
