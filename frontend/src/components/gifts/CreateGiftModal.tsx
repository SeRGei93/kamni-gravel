'use client';

import React, { useState } from 'react';
import { Modal } from '@/components/ui/modal';
import Label from '@/components/form/Label';
import TextArea from '@/components/form/input/TextArea';
import InputField from '@/components/form/input/InputField';
import Button from '@/components/ui/button/Button';
import { giftsApi } from '@/api/gifts';
import type { CreateGiftRequest } from '@/types';

interface CreateGiftModalProps {
  isOpen: boolean;
  onClose: () => void;
  eventId: number;
  onSuccess?: () => void;
}

export default function CreateGiftModal({
  isOpen,
  onClose,
  eventId,
  onSuccess,
}: CreateGiftModalProps) {
  const [userId, setUserId] = useState('');
  const [description, setDescription] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleClose = () => {
    setUserId('');
    setDescription('');
    setError(null);
    onClose();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!userId.trim()) {
      setError('Укажите Telegram ID пользователя');
      return;
    }

    if (!description.trim()) {
      setError('Введите описание подарка');
      return;
    }

    const userIdNum = parseInt(userId, 10);
    if (isNaN(userIdNum)) {
      setError('Telegram ID должен быть числом');
      return;
    }

    try {
      setIsSubmitting(true);
      setError(null);

      const data: CreateGiftRequest = {
        user_id: userIdNum,
        description: description.trim(),
      };

      await giftsApi.create(eventId, data);

      if (onSuccess) {
        onSuccess();
      }

      handleClose();
    } catch (err) {
      console.error('Failed to create gift:', err);
      if (err instanceof Error && err.message.includes('not found')) {
        setError('Пользователь с таким Telegram ID не найден');
      } else {
        setError('Ошибка создания подарка');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} className="max-w-lg m-4">
      <div className="no-scrollbar relative w-full max-w-lg overflow-y-auto rounded-3xl bg-white p-4 dark:bg-gray-900 lg:p-11">
        <div className="px-2 pr-14">
          <h4 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white/90">
            Добавить подарок
          </h4>
          <p className="mb-6 text-sm text-gray-500 dark:text-gray-400 lg:mb-7">
            Добавьте новый подарок в призовой фонд
          </p>
        </div>

        {error && (
          <div className="mb-4 rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
            <p className="text-error-600 dark:text-error-400">{error}</p>
          </div>
        )}

        <form onSubmit={handleSubmit} className="px-2">
          <div className="space-y-6">
            <div>
              <Label>
                Telegram ID дарителя <span className="text-error-500">*</span>
              </Label>
              <InputField
                type="text"
                placeholder="Например: 123456789"
                value={userId}
                onChange={(e) => setUserId(e.target.value)}
              />
              <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                ID пользователя Telegram, который дарит подарок
              </p>
            </div>

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
                {isSubmitting ? 'Создание...' : 'Добавить подарок'}
              </Button>
            </div>
          </div>
        </form>
      </div>
    </Modal>
  );
}
