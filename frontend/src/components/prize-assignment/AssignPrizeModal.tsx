'use client';

import React, { useState, useEffect } from 'react';
import { Modal } from '@/components/ui/modal';
import Select from '@/components/form/Select';
import Label from '@/components/form/Label';
import TextArea from '@/components/form/input/TextArea';
import Button from '@/components/ui/button/Button';
import { giftsApi } from '@/api/gifts';
import { prizeAssignmentsApi } from '@/api/prizeAssignments';
import { getCriteriaColor } from '@/utils/criteria';
import Badge from '../ui/badge/Badge';
import type {
  Gift,
  ParticipantDetail,
  CreatePrizeAssignmentRequest,
} from '@/types';

interface AssignPrizeModalProps {
  isOpen: boolean;
  onClose: () => void;
  participant: ParticipantDetail;
  onSuccess?: () => void;
}


export default function AssignPrizeModal({
  isOpen,
  onClose,
  participant,
  onSuccess,
}: AssignPrizeModalProps) {
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [selectedGiftId, setSelectedGiftId] = useState<string>('');
  const [comment, setComment] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Загружаем призы при открытии модального окна
  useEffect(() => {
    if (isOpen && participant.event_id) {
      loadData();
    } else {
      // Сбрасываем состояние при закрытии
      setSelectedGiftId('');
      setComment('');
      setError(null);
    }
  }, [isOpen, participant.event_id]);

  const loadData = async () => {
    try {
      setIsLoading(true);
      setError(null);

      // Загружаем все призы события
      const giftsResponse = await giftsApi.getByEvent(participant.event_id);
      // Все призы доступны для автоматического распределения
      setGifts(giftsResponse.gifts);
    } catch (err) {
      setError('Ошибка загрузки данных');
      console.error('Failed to load data:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!selectedGiftId) {
      setError('Выберите приз');
      return;
    }

    try {
      setIsSubmitting(true);
      setError(null);

      const data: CreatePrizeAssignmentRequest = {
        participant_id: participant.id,
        gift_id: Number(selectedGiftId),
        comment: comment || undefined,
      };

      await prizeAssignmentsApi.create(data);

      // Вызываем callback успешного создания
      if (onSuccess) {
        onSuccess();
      }

      // Закрываем модальное окно
      onClose();
    } catch (err) {
      setError('Ошибка назначения приза');
      console.error('Failed to assign prize:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  const giftOptions = gifts.map((gift) => {
    const criteriaText =
      gift.criteria && gift.criteria.length > 0
        ? ` (${gift.criteria.map((c) => c.name).join(', ')})`
        : '';
    return {
      value: String(gift.id),
      label: `${gift.description}${criteriaText}`,
    };
  });

  return (
    <Modal isOpen={isOpen} onClose={onClose} className="max-w-2xl m-4">
      <div className="no-scrollbar relative w-full max-w-2xl overflow-y-auto rounded-3xl bg-white p-4 dark:bg-gray-900 lg:p-11">
        <div className="px-2 pr-14">
          <h4 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white/90">
            Назначить приз
          </h4>
          <p className="mb-6 text-sm text-gray-500 dark:text-gray-400 lg:mb-7">
            Выберите приз для участника{' '}
            <span className="font-medium">
              {participant.first_name} {participant.last_name}
            </span>
          </p>
        </div>

        {error && (
          <div className="mb-4 rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
            <p className="text-error-600 dark:text-error-400">{error}</p>
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="text-gray-500 dark:text-gray-400">Загрузка данных...</div>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="px-2">
            <div className="space-y-6">
              <div>
                <Label>
                  Приз <span className="text-error-500">*</span>
                </Label>
                {giftOptions.length > 0 ? (
                  <div className="relative">
                    <Select
                      options={giftOptions}
                      placeholder="Выберите приз"
                      defaultValue={selectedGiftId}
                      onChange={setSelectedGiftId}
                    />
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      Доступно призов: {giftOptions.length}
                    </p>
                  </div>
                ) : (
                  <div className="rounded-lg border border-warning-200 bg-warning-50 p-4 dark:border-warning-800 dark:bg-warning-900/20">
                    <p className="text-warning-600 dark:text-warning-400">
                      Нет доступных призов. Все призы уже назначены.
                    </p>
                  </div>
                )}
              </div>

              {selectedGiftId && (
                <div>
                  <Label>Критерии выбранного приза</Label>
                  {(() => {
                    const selectedGift = gifts.find(
                      (g) => String(g.id) === selectedGiftId
                    );
                    if (selectedGift?.criteria && selectedGift.criteria.length > 0) {
                      return (
                        <div className="mt-2 flex flex-wrap gap-2">
                          {selectedGift.criteria.map((c) => (
                            <Badge
                              key={c.id}
                              color={getCriteriaColor(c.criteria_type)}
                              size="sm"
                            >
                              {c.name}
                            </Badge>
                          ))}
                        </div>
                      );
                    }
                    return (
                      <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
                        У этого приза нет критериев
                      </p>
                    );
                  })()}
                </div>
              )}

              <div>
                <Label>Комментарий</Label>
                <TextArea
                  placeholder="Введите комментарий (необязательно)"
                  value={comment}
                  onChange={setComment}
                  rows={3}
                />
              </div>

              <div className="flex items-center gap-3 justify-end">
                <Button type="button" variant="outline" onClick={onClose} disabled={isSubmitting}>
                  Отмена
                </Button>
                <Button
                  type="submit"
                  disabled={isSubmitting || giftOptions.length === 0}
                >
                  {isSubmitting ? 'Назначение...' : 'Назначить приз'}
                </Button>
              </div>
            </div>
          </form>
        )}
      </div>
    </Modal>
  );
}
