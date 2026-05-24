'use client';

import React, { useState, useEffect } from 'react';
import Input from '../form/input/InputField';
import Label from '../form/Label';
import TextArea from '../form/input/TextArea';
import Switch from '../form/switch/Switch';
import Button from '../ui/button/Button';
import Select from '../form/Select';
import type {
  Nomination,
  CreateNominationRequest,
  UpdateNominationRequest,
  GenderFilter,
  BikeTypeFilter,
} from '@/types';

interface NominationFormProps {
  eventId: number;
  nomination?: Nomination;
  onSubmit: (data: CreateNominationRequest | UpdateNominationRequest) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
  participantsCount?: number; // Количество подходящих участников
  onFilterChange?: (genderFilter: GenderFilter, bikeTypeFilter: BikeTypeFilter) => void;
}

const genderFilterOptions = [
  { value: 'all', label: 'Любой' },
  { value: 'male', label: 'Мужской' },
  { value: 'female', label: 'Женский' },
];

const bikeTypeFilterOptions = [
  { value: 'all', label: 'Любой' },
  { value: 'gravel', label: 'Гравийник' },
  { value: 'mtb', label: 'МТБ' },
  { value: 'road', label: 'Шоссе' },
  { value: 'single_speed', label: 'Фикс' },
  { value: 'tandem', label: 'Тандем' },
];

export default function NominationForm({
  eventId,
  nomination,
  onSubmit,
  onCancel,
  isLoading = false,
  participantsCount,
  onFilterChange,
}: NominationFormProps) {
  const [name, setName] = useState(nomination?.name || '');
  const [description, setDescription] = useState(nomination?.description || '');
  const [genderFilter, setGenderFilter] = useState<GenderFilter>(
    nomination?.gender_filter || 'all'
  );
  const [bikeTypeFilter, setBikeTypeFilter] = useState<BikeTypeFilter>(
    nomination?.bike_type_filter || 'all'
  );
  const [sortOrder, setSortOrder] = useState(
    nomination?.sort_order?.toString() || '0'
  );
  const [isActive, setIsActive] = useState(nomination?.is_active ?? true);

  // Обновляем состояние при изменении nomination
  useEffect(() => {
    if (nomination) {
      setName(nomination.name);
      setDescription(nomination.description);
      setGenderFilter(nomination.gender_filter);
      setBikeTypeFilter(nomination.bike_type_filter);
      setSortOrder(nomination.sort_order.toString());
      setIsActive(nomination.is_active);
    } else {
      setName('');
      setDescription('');
      setGenderFilter('all');
      setBikeTypeFilter('all');
      setSortOrder('0');
      setIsActive(true);
    }
  }, [nomination]);

  // Загружаем количество участников при монтировании или изменении фильтров
  useEffect(() => {
    if (onFilterChange) {
      onFilterChange(genderFilter, bikeTypeFilter);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [genderFilter, bikeTypeFilter]); // При изменении фильтров

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (nomination) {
      // Обновление
      const data: UpdateNominationRequest = {
        name,
        description,
        gender_filter: genderFilter,
        bike_type_filter: bikeTypeFilter,
        sort_order: Number(sortOrder),
        is_active: isActive,
      };
      await onSubmit(data);
    } else {
      // Создание
      const data: CreateNominationRequest = {
        event_id: eventId,
        name,
        description,
        gender_filter: genderFilter,
        bike_type_filter: bikeTypeFilter,
        sort_order: Number(sortOrder),
        is_active: isActive,
      };
      await onSubmit(data);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div>
        <Label>
          Название <span className="text-error-500">*</span>
        </Label>
        <Input
          type="text"
          placeholder="Например: 1 место (абсолют)"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          disabled={isLoading}
        />
      </div>

      <div>
        <Label>Описание</Label>
        <TextArea
          placeholder="Описание номинации"
          value={description}
          onChange={setDescription}
          rows={3}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <Label>
            Фильтр по полу <span className="text-error-500">*</span>
          </Label>
          <div className="relative">
            <Select
              key={`gender-${genderFilter}`}
              options={genderFilterOptions}
              placeholder="Выберите фильтр"
              defaultValue={genderFilter}
              onChange={(value) => {
                const newFilter = value as GenderFilter;
                setGenderFilter(newFilter);
                if (onFilterChange) {
                  onFilterChange(newFilter, bikeTypeFilter);
                }
              }}
            />
          </div>
        </div>

        <div>
          <Label>
            Фильтр по типу велосипеда <span className="text-error-500">*</span>
          </Label>
          <div className="relative">
            <Select
              key={`bike-${bikeTypeFilter}`}
              options={bikeTypeFilterOptions}
              placeholder="Выберите фильтр"
              defaultValue={bikeTypeFilter}
              onChange={(value) => {
                const newFilter = value as BikeTypeFilter;
                setBikeTypeFilter(newFilter);
                if (onFilterChange) {
                  onFilterChange(genderFilter, newFilter);
                }
              }}
            />
          </div>
        </div>
      </div>

      {/* Предпросмотр количества подходящих участников */}
      {participantsCount !== undefined && (
        <div className="rounded-lg border border-brand-200 bg-brand-50 p-4 dark:border-brand-800 dark:bg-brand-900/20">
          <p className="text-sm text-brand-700 dark:text-brand-300">
            <span className="font-medium">Подходящих участников:</span>{' '}
            {participantsCount}
          </p>
        </div>
      )}

      <div>
        <Label>Порядок сортировки</Label>
        <Input
          type="number"
          placeholder="0"
          value={sortOrder}
          onChange={(e) => setSortOrder(e.target.value)}
          disabled={isLoading}
          min="0"
        />
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
          Меньшее значение = выше в списке
        </p>
      </div>

      <div>
        <Switch
          label="Активная номинация"
          defaultChecked={isActive}
          onChange={setIsActive}
        />
      </div>

      <div className="flex items-center gap-3 justify-end">
        <Button type="button" variant="outline" onClick={onCancel} disabled={isLoading}>
          Отмена
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading
            ? 'Сохранение...'
            : nomination
              ? 'Сохранить изменения'
              : 'Создать номинацию'}
        </Button>
      </div>
    </form>
  );
}
