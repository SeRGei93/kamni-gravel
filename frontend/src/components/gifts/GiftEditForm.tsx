'use client';

import { useEffect, useMemo, useState } from 'react';
import Label from '@/components/form/Label';
import TextArea from '@/components/form/input/TextArea';
import InputField from '@/components/form/input/InputField';
import Select from '@/components/form/Select';
import Button from '@/components/ui/button/Button';
import Badge from '@/components/ui/badge/Badge';
import { Modal } from '@/components/ui/modal';
import { useModal } from '@/hooks/useModal';
import CriteriaForm from '@/components/criteria/CriteriaForm';
import { PlusIcon, TrashBinIcon } from '@/icons';
import { addSelectedCriterionId, getCriteriaColor } from '@/utils/criteria';
import type {
  BikeTypeFilter,
  CreateCriteriaRequest,
  Criteria,
  GenderFilter,
  Gift,
  GiftReviewStatus,
  ManualGift,
  Participant,
  UpdateCriteriaRequest,
  UpdateGiftRequest,
} from '@/types';
import {
  buildGiftPlaceRuleFromForm,
  formatGiftPlaceRule,
  getGiftPlaceRuleFormState,
  type GiftPlaceRuleMode,
} from '@/utils/giftPlaceRule';
import {
  BIKE_TYPE_OPTIONS,
  GENDER_OPTIONS,
  GIFT_PLACE_RULE_OPTIONS,
  GIFT_REVIEW_STATUS_OPTIONS,
} from '@/constants';
import {
  buildManualGiftUpdate,
  formatManualRecipientSearchLabel,
} from '@/utils/manualGiftAssignment';
import { matchesSearchQuery } from '@/utils/search';

interface GiftEditFormProps {
  gift: Gift;
  criteria: Criteria[];
  manualGift?: ManualGift;
  participants?: Participant[];
  onSubmit: (data: UpdateGiftRequest) => Promise<void>;
  onCancel: () => void;
  onDelete?: () => Promise<void>;
  onCreateCriteria?: (data: CreateCriteriaRequest) => Promise<Criteria>;
}

export default function GiftEditForm({
  gift,
  criteria,
  manualGift,
  participants = [],
  onSubmit,
  onCancel,
  onDelete,
  onCreateCriteria,
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
  const initialPlaceRule = getGiftPlaceRuleFormState(gift);
  const [placeRuleMode, setPlaceRuleMode] = useState<GiftPlaceRuleMode>(
    initialPlaceRule.mode
  );
  const [placeRuleInput, setPlaceRuleInput] = useState(
    initialPlaceRule.placesInput
  );
  const [lastCountInput, setLastCountInput] = useState(
    initialPlaceRule.lastCount
  );
  const [selectedCriteriaIds, setSelectedCriteriaIds] = useState<number[]>(
    gift.criteria?.map((item) => item.id) || []
  );
  const [manualDistribution, setManualDistribution] = useState(
    gift.manual_distribution ?? manualGift?.manual_distribution ?? false
  );
  const [manualRecipientParticipantID, setManualRecipientParticipantID] =
    useState<number | null>(manualGift?.recipient?.id ?? null);
  const [recipientSearch, setRecipientSearch] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const {
    isOpen: isCriteriaModalOpen,
    openModal: openCriteriaModal,
    closeModal: closeCriteriaModal,
  } = useModal();
  const [isCreatingCriteria, setIsCreatingCriteria] = useState(false);
  const [criteriaError, setCriteriaError] = useState<string | null>(null);

  const handleOpenCriteriaModal = () => {
    setCriteriaError(null);
    openCriteriaModal();
  };

  // Создание критерия обновляет состояние через onCreateCriteria и авто-выбирает
  // новый критерий — страница не перезагружается.
  const handleCreateCriteria = async (
    data: CreateCriteriaRequest | UpdateCriteriaRequest
  ) => {
    if (!onCreateCriteria) {
      return;
    }
    setCriteriaError(null);
    setIsCreatingCriteria(true);
    try {
      const created = await onCreateCriteria(data as CreateCriteriaRequest);
      setSelectedCriteriaIds((current) =>
        addSelectedCriterionId(current, created.id)
      );
      closeCriteriaModal();
    } catch {
      setCriteriaError('Не удалось создать критерий. Попробуйте ещё раз.');
    } finally {
      setIsCreatingCriteria(false);
    }
  };

  useEffect(() => {
    setDescription(gift.description);
    setGenderFilter(gift.gender_filter || 'all');
    setBikeTypeFilter(gift.bike_type_filter || 'all');
    setReviewStatus(gift.review_status);
    const nextPlaceRule = getGiftPlaceRuleFormState(gift);
    setPlaceRuleMode(nextPlaceRule.mode);
    setPlaceRuleInput(nextPlaceRule.placesInput);
    setLastCountInput(nextPlaceRule.lastCount);
    setSelectedCriteriaIds(gift.criteria?.map((item) => item.id) || []);
    setManualDistribution(
      gift.manual_distribution ?? manualGift?.manual_distribution ?? false
    );
    setManualRecipientParticipantID(manualGift?.recipient?.id ?? null);
    setRecipientSearch('');
    setError(null);
  }, [gift, manualGift]);

  const placeRulePreview = useMemo(() => {
    try {
      return formatGiftPlaceRule(
        buildGiftPlaceRuleFromForm(placeRuleMode, placeRuleInput, lastCountInput)
      );
    } catch {
      return 'Проверьте формат правила';
    }
  }, [lastCountInput, placeRuleInput, placeRuleMode]);

  const toggleCriteria = (criteriaId: number) => {
    setSelectedCriteriaIds((current) =>
      current.includes(criteriaId)
        ? current.filter((id) => id !== criteriaId)
        : [...current, criteriaId]
    );
  };

  const filteredRecipientOptions = useMemo(() => {
    return participants.filter((participant) =>
      matchesSearchQuery(recipientSearch, [
        participant.first_name,
        participant.last_name,
        participant.username,
        participant.id,
        participant.user_id,
      ])
    );
  }, [participants, recipientSearch]);

  useEffect(() => {
    if (!recipientSearch.trim()) {
      return;
    }
    console.debug('[FIX:recipient-search] admin filter completed', {
      gift_id: gift.id,
      result_count: filteredRecipientOptions.length,
      has_username_prefix: recipientSearch.trim().startsWith('@'),
    });
  }, [filteredRecipientOptions.length, gift.id, recipientSearch]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();

    const trimmedDescription = description.trim();
    if (!trimmedDescription) {
      setError('Введите описание приза');
      return;
    }

    let placeRule: UpdateGiftRequest['place_rule'];
    try {
      placeRule = buildGiftPlaceRuleFromForm(placeRuleMode, placeRuleInput, lastCountInput);
    } catch {
      setError('Проверьте правило мест: используйте числа, запятые и диапазоны вроде 10-15');
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
        place_rule: placeRule,
        criteria_ids: selectedCriteriaIds,
        ...buildManualGiftUpdate(
          manualDistribution,
          manualRecipientParticipantID
        ),
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

  const handleDelete = async () => {
    if (!onDelete) {
      return;
    }
    if (!window.confirm('Вы уверены, что хотите удалить этот приз?')) {
      return;
    }

    try {
      setIsDeleting(true);
      setError(null);
      // При успехе родитель уводит со страницы — состояние сбрасывать не нужно.
      await onDelete();
    } catch (err) {
      console.error('Failed to delete gift:', {
        gift_id: gift.id,
        operation: 'delete_gift',
        error: err,
      });
      setError('Ошибка удаления приза');
      setIsDeleting(false);
    }
  };

  return (
    <>
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
          <Label>Статус проверки</Label>
          <Select
            options={GIFT_REVIEW_STATUS_OPTIONS}
            defaultValue={reviewStatus}
            onChange={(value) => setReviewStatus(value as GiftReviewStatus)}
          />
        </div>
      </div>

      <div className="space-y-3 border-t border-gray-100 pt-5 dark:border-white/[0.05]">
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[220px_1fr_160px]">
          <div>
            <Label>Правило мест</Label>
            <Select
              options={GIFT_PLACE_RULE_OPTIONS}
              defaultValue={placeRuleMode}
              onChange={(value) => setPlaceRuleMode(value as GiftPlaceRuleMode)}
            />
          </div>
          <div>
            <Label>Места или диапазоны</Label>
            <InputField
              type="text"
              placeholder="Например: 1, 3, 10-15"
              value={placeRuleInput}
              onChange={(event) => setPlaceRuleInput(event.target.value)}
              disabled={placeRuleMode !== 'places'}
            />
          </div>
          <div>
            <Label>Последние N</Label>
            <InputField
              type="number"
              min="1"
              step={1}
              placeholder="5"
              value={lastCountInput}
              onChange={(event) => setLastCountInput(event.target.value)}
              disabled={placeRuleMode !== 'last_n'}
            />
          </div>
        </div>
        <p className="text-xs font-medium text-gray-600 dark:text-gray-300">
          {placeRulePreview}
        </p>
      </div>

      <div className="space-y-3 border-t border-gray-100 pt-5 dark:border-white/[0.05]">
        <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-gray-200 p-4 dark:border-white/[0.08]">
          <input
            type="checkbox"
            checked={manualDistribution}
            onChange={(event) => {
              const enabled = event.target.checked;
              setManualDistribution(enabled);
              if (!enabled) {
                setManualRecipientParticipantID(null);
              }
            }}
            className="mt-0.5 h-4 w-4 rounded border-gray-300 text-brand-500 focus:ring-brand-500"
          />
          <span>
            <span className="block text-sm font-medium text-gray-800 dark:text-white/90">
              Ручное распределение
            </span>
            <span className="mt-1 block text-xs text-gray-500 dark:text-gray-400">
              Приз не участвует в автоматическом распределении.
            </span>
          </span>
        </label>

        {manualDistribution && (
          <div className="rounded-lg border border-brand-100 bg-brand-50/40 p-4 dark:border-brand-500/20 dark:bg-brand-500/5">
            <Label>Получатель</Label>
            <p className="mb-3 text-xs text-gray-500 dark:text-gray-400">
              Можно назначить любого участника этого события, включая участника без результата.
            </p>
            <input
              type="search"
              value={recipientSearch}
              onChange={(event) => setRecipientSearch(event.target.value)}
              placeholder="Поиск по имени или username"
              className="mb-3 h-10 w-full rounded-lg border border-gray-300 bg-transparent px-3 text-sm text-gray-800 outline-none focus:border-brand-500 dark:border-gray-700 dark:text-white/90"
            />
            <select
              value={manualRecipientParticipantID ?? ''}
              onChange={(event) => {
                const value = event.target.value;
                setManualRecipientParticipantID(value ? Number(value) : null);
              }}
              className="h-10 w-full rounded-lg border border-gray-300 bg-transparent px-3 text-sm text-gray-800 outline-none focus:border-brand-500 dark:border-gray-700 dark:text-white/90"
            >
              <option value="">Получатель пока не выбран</option>
              {filteredRecipientOptions.map((participant) => {
                const displayName = [participant.first_name, participant.last_name]
                  .filter(Boolean)
                  .join(' ') || `Участник #${participant.id}`;
                return (
                  <option key={participant.id} value={participant.id}>
                    {formatManualRecipientSearchLabel(displayName, participant.username)}
                  </option>
                );
              })}
            </select>
            {participants.length === 0 && (
              <p className="mt-2 text-xs text-warning-600 dark:text-warning-400">
                В этом событии пока нет доступных участников.
              </p>
            )}
          </div>
        )}
      </div>

      <div>
        <div className="mb-1 flex items-center justify-between gap-3">
          <Label>Критерии</Label>
          {onCreateCriteria && (
            <Button
              type="button"
              size="sm"
              variant="outline"
              startIcon={<PlusIcon />}
              onClick={handleOpenCriteriaModal}
            >
              Создать критерий
            </Button>
          )}
        </div>
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

      <div className="flex flex-col-reverse gap-3 border-t border-gray-100 pt-5 dark:border-white/[0.05] sm:flex-row sm:items-center sm:justify-between">
        {onDelete ? (
          <Button
            type="button"
            variant="outline"
            startIcon={<TrashBinIcon />}
            onClick={handleDelete}
            disabled={isSubmitting || isDeleting}
          >
            {isDeleting ? 'Удаление...' : 'Удалить приз'}
          </Button>
        ) : (
          <span />
        )}
        <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <Button
            type="button"
            variant="outline"
            onClick={onCancel}
            disabled={isSubmitting || isDeleting}
          >
            Отмена
          </Button>
          <Button type="submit" disabled={isSubmitting || isDeleting}>
            {isSubmitting ? 'Сохранение...' : 'Сохранить'}
          </Button>
        </div>
      </div>
    </form>

    {onCreateCriteria && (
      <Modal
        isOpen={isCriteriaModalOpen}
        onClose={closeCriteriaModal}
        className="max-w-2xl m-4"
      >
        <div className="no-scrollbar relative w-full max-w-2xl overflow-y-auto rounded-3xl bg-white p-4 dark:bg-gray-900 lg:p-11">
          <div className="px-2 pr-14">
            <h4 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white/90">
              Создать критерий
            </h4>
            <p className="mb-6 text-sm text-gray-500 dark:text-gray-400 lg:mb-7">
              Новый критерий сразу станет доступен и будет выбран для этого приза
            </p>
          </div>
          {criteriaError && (
            <div className="mx-2 mb-4 rounded-lg border border-error-200 bg-error-50 p-3 dark:border-error-800 dark:bg-error-900/20">
              <p className="text-sm text-error-600 dark:text-error-400">
                {criteriaError}
              </p>
            </div>
          )}
          <div className="px-2">
            <CriteriaForm
              onSubmit={handleCreateCriteria}
              onCancel={closeCriteriaModal}
              isLoading={isCreatingCriteria}
            />
          </div>
        </div>
      </Modal>
    )}
    </>
  );
}
