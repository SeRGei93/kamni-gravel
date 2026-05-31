'use client';

import { FormEvent, useEffect, useState } from 'react';
import { adminUsersApi } from '@/api/adminUsers';
import type { AdminUser } from '@/types';
import Button from '@/components/ui/button/Button';
import Input from '@/components/form/input/InputField';
import Label from '@/components/form/Label';
import {
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { PlusIcon, UserIcon } from '@/icons';

const MIN_PASSWORD_LENGTH = 8;

export default function AdminUsersPage() {
  const [admins, setAdmins] = useState<AdminUser[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAdmins();
  }, []);

  const loadAdmins = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await adminUsersApi.getAll();
      setAdmins(response.admins);
    } catch (err) {
      setError('Ошибка загрузки администраторов');
      console.error('Admin users API failed:', {
        operation: 'list_admin_users',
        error: err,
      });
    } finally {
      setIsLoading(false);
    }
  };

  const resetForm = () => {
    setUsername('');
    setPassword('');
    setConfirmPassword('');
  };

  const handleCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const normalizedUsername = username.trim();

    if (!normalizedUsername) {
      setError('Укажите имя администратора');
      return;
    }
    if (!password) {
      setError('Укажите пароль');
      return;
    }
    if (password.length < MIN_PASSWORD_LENGTH) {
      setError(`Пароль должен быть не короче ${MIN_PASSWORD_LENGTH} символов`);
      return;
    }
    if (password !== confirmPassword) {
      setError('Пароли не совпадают');
      return;
    }

    try {
      setIsSaving(true);
      setError(null);
      await adminUsersApi.create({
        username: normalizedUsername,
        password,
      });
      resetForm();
      setIsFormOpen(false);
      await loadAdmins();
    } catch (err) {
      setError('Ошибка создания администратора');
      console.error('Admin users API failed:', {
        operation: 'create_admin_user',
        username: normalizedUsername,
        error: err,
      });
    } finally {
      setIsSaving(false);
    }
  };

  const formatDate = (value?: string | null) => {
    if (!value) return '-';
    return new Date(value).toLocaleString('ru-RU', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
            Администраторы
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Аккаунты с доступом к панели управления
          </p>
        </div>
        <Button
          size="sm"
          startIcon={<PlusIcon />}
          onClick={() => {
            setIsFormOpen((value) => !value);
            setError(null);
          }}
        >
          Добавить админа
        </Button>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      {isFormOpen && (
        <form
          onSubmit={handleCreate}
          className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]"
        >
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(200px,1fr)_minmax(180px,1fr)_minmax(180px,1fr)_auto] xl:items-end">
            <div>
              <Label>Имя пользователя</Label>
              <Input
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                placeholder="admin"
                disabled={isSaving}
                required
              />
            </div>
            <div>
              <Label>Пароль</Label>
              <Input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="Минимум 8 символов"
                disabled={isSaving}
                required
              />
            </div>
            <div>
              <Label>Подтверждение</Label>
              <Input
                type="password"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                placeholder="Повторите пароль"
                disabled={isSaving}
                required
              />
            </div>
            <div className="flex gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => {
                  resetForm();
                  setIsFormOpen(false);
                }}
                disabled={isSaving}
                className="h-11"
              >
                Отмена
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={isSaving}
                className="h-11"
              >
                {isSaving ? 'Создание...' : 'Создать'}
              </Button>
            </div>
          </div>
        </form>
      )}

      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Администраторов: {admins.length}
        </p>
        <Button
          size="sm"
          variant="outline"
          onClick={loadAdmins}
          disabled={isLoading}
        >
          Обновить
        </Button>
      </div>

      <div className="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-white/[0.05] dark:bg-white/[0.03]">
        <div className="max-w-full overflow-x-auto">
          <div className="min-w-[860px]">
            <Table>
              <TableHeader className="border-b border-gray-100 dark:border-white/[0.05]">
                <TableRow>
                  <TableCell
                    isHeader
                    className="px-5 py-3 text-start text-theme-xs font-medium text-gray-500 dark:text-gray-400"
                  >
                    Пользователь
                  </TableCell>
                  <TableCell
                    isHeader
                    className="px-5 py-3 text-start text-theme-xs font-medium text-gray-500 dark:text-gray-400"
                  >
                    Роль
                  </TableCell>
                  <TableCell
                    isHeader
                    className="px-5 py-3 text-start text-theme-xs font-medium text-gray-500 dark:text-gray-400"
                  >
                    Создан
                  </TableCell>
                  <TableCell
                    isHeader
                    className="px-5 py-3 text-start text-theme-xs font-medium text-gray-500 dark:text-gray-400"
                  >
                    Последний вход
                  </TableCell>
                </TableRow>
              </TableHeader>
              <TableBody className="divide-y divide-gray-100 dark:divide-white/[0.05]">
                {isLoading ? (
                  <TableRow>
                    <td
                      colSpan={4}
                      className="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400"
                    >
                      Загрузка...
                    </td>
                  </TableRow>
                ) : admins.length === 0 ? (
                  <TableRow>
                    <td
                      colSpan={4}
                      className="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400"
                    >
                      Администраторы не найдены
                    </td>
                  </TableRow>
                ) : (
                  admins.map((admin) => (
                    <TableRow
                      key={admin.id}
                      className="hover:bg-gray-50 dark:hover:bg-white/5"
                    >
                      <TableCell className="px-5 py-4 text-start">
                        <div className="flex items-center gap-3">
                          <span className="inline-flex h-9 w-9 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-white/[0.06] dark:text-gray-300">
                            <UserIcon />
                          </span>
                          <div>
                            <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                              {admin.username}
                            </p>
                            <p className="text-xs text-gray-500 dark:text-gray-400">
                              ID {admin.id}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="px-5 py-4 text-start">
                        <span className="rounded-full bg-brand-50 px-2.5 py-1 text-xs font-medium text-brand-700 dark:bg-brand-500/15 dark:text-brand-300">
                          {admin.role}
                        </span>
                      </TableCell>
                      <TableCell className="px-5 py-4 text-start text-sm text-gray-600 dark:text-gray-400">
                        {formatDate(admin.created_at)}
                      </TableCell>
                      <TableCell className="px-5 py-4 text-start text-sm text-gray-600 dark:text-gray-400">
                        {formatDate(admin.last_login)}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </div>
  );
}
