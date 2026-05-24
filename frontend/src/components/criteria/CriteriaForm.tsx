'use client';

import React, { useState, useEffect } from 'react';
import Input from '../form/input/InputField';
import Label from '../form/Label';
import TextArea from '../form/input/TextArea';
import Button from '../ui/button/Button';
import Select from '../form/Select';
import type {
  Criteria,
  CreateCriteriaRequest,
  UpdateCriteriaRequest,
  CriteriaType,
} from '@/types';

interface CriteriaFormProps {
  criteria?: Criteria;
  onSubmit: (data: CreateCriteriaRequest | UpdateCriteriaRequest) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
}

const criteriaTypeOptions = [
  { value: 'speed', label: 'Скорость' },
  { value: 'photo', label: 'Фото' },
  { value: 'beer', label: 'Пиво' },
  { value: 'custom', label: 'Кастомный' },
];

export default function CriteriaForm({
  criteria,
  onSubmit,
  onCancel,
  isLoading = false,
}: CriteriaFormProps) {
  const [name, setName] = useState(criteria?.name || '');
  const [description, setDescription] = useState(criteria?.description || '');
  const [criteriaType, setCriteriaType] = useState<CriteriaType>(
    criteria?.criteria_type || 'custom'
  );

  // Обновляем состояние при изменении criteria
  useEffect(() => {
    if (criteria) {
      setName(criteria.name);
      setDescription(criteria.description);
      setCriteriaType(criteria.criteria_type);
    } else {
      setName('');
      setDescription('');
      setCriteriaType('custom');
    }
  }, [criteria]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (criteria) {
      // Обновление
      const data: UpdateCriteriaRequest = {
        name,
        description,
        criteria_type: criteriaType,
      };
      await onSubmit(data);
    } else {
      // Создание
      const data: CreateCriteriaRequest = {
        name,
        description,
        criteria_type: criteriaType,
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
          placeholder="Например: Максимальная скорость"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          disabled={isLoading}
        />
      </div>

      <div>
        <Label>
          Описание <span className="text-error-500">*</span>
        </Label>
        <TextArea
          placeholder="Описание критерия"
          value={description}
          onChange={setDescription}
          rows={3}
          disabled={isLoading}
        />
      </div>

      <div>
        <Label>
          Тип критерия <span className="text-error-500">*</span>
        </Label>
        <div className="relative">
          <Select
            key={`type-${criteriaType}`}
            options={criteriaTypeOptions}
            placeholder="Выберите тип"
            defaultValue={criteriaType}
            onChange={(value) => setCriteriaType(value as CriteriaType)}
          />
        </div>
      </div>

      <div className="flex items-center gap-3 justify-end">
        <Button type="button" variant="outline" onClick={onCancel} disabled={isLoading}>
          Отмена
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading
            ? 'Сохранение...'
            : criteria
              ? 'Сохранить изменения'
              : 'Создать критерий'}
        </Button>
      </div>
    </form>
  );
}
