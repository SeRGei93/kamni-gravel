'use client';

import { FormEvent, useState } from 'react';
import { useRouter } from 'next/navigation';
import { authApi } from '@/api/auth';
import { ApiError } from '@/api/client';
import Button from '@/components/ui/button/Button';
import Input from '@/components/form/input/InputField';
import Label from '@/components/form/Label';
import { useAuth } from '@/hooks/useAuth';
import { LockIcon } from '@/icons';

const MIN_PASSWORD_LENGTH = 8;

export default function ChangePasswordPage() {
  const router = useRouter();
  const { logout } = useAuth();
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!currentPassword) {
      setError('Укажите текущий пароль');
      return;
    }
    if (!newPassword) {
      setError('Укажите новый пароль');
      return;
    }
    if (newPassword.length < MIN_PASSWORD_LENGTH) {
      setError(`Новый пароль должен быть не короче ${MIN_PASSWORD_LENGTH} символов`);
      return;
    }
    if (newPassword !== confirmPassword) {
      setError('Новый пароль и подтверждение не совпадают');
      return;
    }

    try {
      setIsSaving(true);
      setError(null);
      await authApi.changePassword({
        current_password: currentPassword,
        new_password: newPassword,
      });
      logout();
      router.replace('/login');
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError('Текущий пароль указан неверно');
      } else {
        setError('Ошибка смены пароля');
      }
      console.error('Auth API failed:', {
        operation: 'change_password',
        error: err,
      });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div>
        <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
          Сменить пароль
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          После сохранения потребуется войти заново
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      <form
        onSubmit={handleSubmit}
        className="rounded-xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03]"
      >
        <div className="mb-6 flex items-center gap-3">
          <span className="inline-flex h-11 w-11 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-white/[0.06] dark:text-gray-300">
            <LockIcon />
          </span>
          <div>
            <h2 className="text-base font-semibold text-gray-800 dark:text-white/90">
              Пароль текущего аккаунта
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Используйте пароль длиной минимум 8 символов
            </p>
          </div>
        </div>

        <div className="space-y-5">
          <div>
            <Label>Текущий пароль</Label>
            <Input
              type="password"
              value={currentPassword}
              onChange={(event) => setCurrentPassword(event.target.value)}
              disabled={isSaving}
              required
            />
          </div>
          <div>
            <Label>Новый пароль</Label>
            <Input
              type="password"
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
              disabled={isSaving}
              required
            />
          </div>
          <div>
            <Label>Подтверждение нового пароля</Label>
            <Input
              type="password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              disabled={isSaving}
              required
            />
          </div>
        </div>

        <div className="mt-6 flex justify-end">
          <Button type="submit" size="sm" disabled={isSaving}>
            {isSaving ? 'Сохранение...' : 'Сменить пароль'}
          </Button>
        </div>
      </form>
    </div>
  );
}
