'use client';

import { FormEvent, useEffect, useState } from 'react';
import { userBlacklistApi } from '@/api/userBlacklist';
import type { UserBlacklistEntry } from '@/types';
import Button from '@/components/ui/button/Button';
import Input from '@/components/form/input/InputField';
import TextArea from '@/components/form/input/TextArea';
import Label from '@/components/form/Label';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '@/components/ui/table';
import { CheckLineIcon, CloseLineIcon, PencilIcon, PlusIcon, TrashIcon } from '@/icons';

export default function UserBlacklistPage() {
  const [entries, setEntries] = useState<UserBlacklistEntry[]>([]);
  const [telegramUserId, setTelegramUserId] = useState('');
  const [reason, setReason] = useState('');
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editingReason, setEditingReason] = useState('');
  const [pendingId, setPendingId] = useState<number | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadEntries();
  }, []);

  const loadEntries = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await userBlacklistApi.getAll();
      setEntries(response.entries);
    } catch (err) {
      setError('Ошибка загрузки blacklist');
      console.error('Failed to load user blacklist:', {
        operation: 'list',
        error: err,
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleAdd = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const parsedTelegramUserId = Number(telegramUserId);
    if (!Number.isInteger(parsedTelegramUserId) || parsedTelegramUserId <= 0) {
      setError('Telegram ID должен быть положительным числом');
      return;
    }

    try {
      setIsSaving(true);
      setError(null);
      await userBlacklistApi.add({
        telegram_user_id: parsedTelegramUserId,
        reason,
      });
      setTelegramUserId('');
      setReason('');
      await loadEntries();
    } catch (err) {
      setError('Ошибка добавления в blacklist');
      console.error('Failed to add user blacklist entry:', {
        operation: 'add',
        telegram_user_id: parsedTelegramUserId,
        error: err,
      });
    } finally {
      setIsSaving(false);
    }
  };

  const startEdit = (entry: UserBlacklistEntry) => {
    setEditingId(entry.telegram_user_id);
    setEditingReason(entry.reason || '');
    setError(null);
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditingReason('');
  };

  const handleUpdate = async (telegramUserId: number) => {
    try {
      setPendingId(telegramUserId);
      setError(null);
      await userBlacklistApi.updateReason(telegramUserId, {
        reason: editingReason,
      });
      setEditingId(null);
      setEditingReason('');
      await loadEntries();
    } catch (err) {
      setError('Ошибка обновления причины блокировки');
      console.error('Failed to update user blacklist entry:', {
        operation: 'update_reason',
        telegram_user_id: telegramUserId,
        error: err,
      });
    } finally {
      setPendingId(null);
    }
  };

  const handleRemove = async (telegramUserId: number) => {
    if (!window.confirm(`Удалить Telegram ID ${telegramUserId} из blacklist?`)) {
      return;
    }

    try {
      setPendingId(telegramUserId);
      setError(null);
      await userBlacklistApi.remove(telegramUserId);
      await loadEntries();
    } catch (err) {
      setError('Ошибка удаления из blacklist');
      console.error('Failed to remove user blacklist entry:', {
        operation: 'remove',
        telegram_user_id: telegramUserId,
        error: err,
      });
    } finally {
      setPendingId(null);
    }
  };

  const displayName = (entry: UserBlacklistEntry) => {
    const name = [entry.first_name, entry.last_name].filter(Boolean).join(' ');
    if (name) return name;
    if (entry.username) return `@${entry.username}`;
    return 'Профиль ещё не создан';
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="mb-2 text-2xl font-semibold text-gray-800 dark:text-white">
          Blacklist пользователей
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Telegram ID, которым запрещены регистрация на заезд и отправка призов
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
          <p className="text-error-600 dark:text-error-400">{error}</p>
        </div>
      )}

      <form
        onSubmit={handleAdd}
        className="rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-white/[0.03]"
      >
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-[260px_1fr_auto] lg:items-end">
          <div>
            <Label>Telegram ID</Label>
            <Input
              type="number"
              min="1"
              placeholder="123456789"
              value={telegramUserId}
              onChange={(event) => setTelegramUserId(event.target.value)}
              disabled={isSaving}
            />
          </div>
          <div>
            <Label>Причина</Label>
            <Input
              type="text"
              placeholder="Причина блокировки"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              disabled={isSaving}
            />
          </div>
          <Button
            type="submit"
            size="sm"
            startIcon={<PlusIcon />}
            disabled={isSaving}
            className="h-11"
          >
            {isSaving ? 'Добавление...' : 'Добавить'}
          </Button>
        </div>
      </form>

      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Записей: {entries.length}
        </p>
        <Button size="sm" variant="outline" onClick={loadEntries} disabled={isLoading}>
          Обновить
        </Button>
      </div>

      <div className="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-white/[0.05] dark:bg-white/[0.03]">
        <div className="max-w-full overflow-x-auto">
          <div className="min-w-[980px]">
            <Table>
              <TableHeader className="border-b border-gray-100 dark:border-white/[0.05]">
                <TableRow>
                  <TableCell isHeader className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400">
                    Telegram ID
                  </TableCell>
                  <TableCell isHeader className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400">
                    Профиль
                  </TableCell>
                  <TableCell isHeader className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400">
                    Причина
                  </TableCell>
                  <TableCell isHeader className="px-5 py-3 font-medium text-gray-500 text-start text-theme-xs dark:text-gray-400">
                    Добавлен
                  </TableCell>
                  <TableCell isHeader className="px-5 py-3 font-medium text-gray-500 text-end text-theme-xs dark:text-gray-400">
                    Действия
                  </TableCell>
                </TableRow>
              </TableHeader>
              <TableBody className="divide-y divide-gray-100 dark:divide-white/[0.05]">
                {isLoading ? (
                  <TableRow>
                    <td colSpan={5} className="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                      Загрузка...
                    </td>
                  </TableRow>
                ) : entries.length === 0 ? (
                  <TableRow>
                    <td colSpan={5} className="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                      Blacklist пуст
                    </td>
                  </TableRow>
                ) : (
                  entries.map((entry) => (
                    <TableRow key={entry.telegram_user_id} className="hover:bg-gray-50 dark:hover:bg-white/5">
                      <TableCell className="px-5 py-4 text-start">
                        <span className="font-medium text-gray-800 text-theme-sm dark:text-white/90">
                          {entry.telegram_user_id}
                        </span>
                      </TableCell>
                      <TableCell className="px-5 py-4 text-start">
                        <div>
                          <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                            {displayName(entry)}
                          </p>
                          {entry.username && (
                            <p className="text-xs text-gray-500 dark:text-gray-400">
                              @{entry.username}
                            </p>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="px-5 py-4 text-start">
                        {editingId === entry.telegram_user_id ? (
                          <TextArea
                            value={editingReason}
                            onChange={setEditingReason}
                            rows={2}
                            disabled={pendingId === entry.telegram_user_id}
                          />
                        ) : (
                          <span className="text-sm text-gray-800 dark:text-white/90">
                            {entry.reason || '-'}
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="px-5 py-4 text-start">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          {new Date(entry.created_at).toLocaleDateString('ru-RU')}
                        </span>
                      </TableCell>
                      <TableCell className="px-5 py-4 text-end">
                        {editingId === entry.telegram_user_id ? (
                          <div className="flex justify-end gap-2">
                            <Button
                              size="xs"
                              variant="outline"
                              startIcon={<CloseLineIcon />}
                              onClick={cancelEdit}
                              disabled={pendingId === entry.telegram_user_id}
                            >
                              Отмена
                            </Button>
                            <Button
                              size="xs"
                              startIcon={<CheckLineIcon />}
                              onClick={() => handleUpdate(entry.telegram_user_id)}
                              disabled={pendingId === entry.telegram_user_id}
                            >
                              Сохранить
                            </Button>
                          </div>
                        ) : (
                          <div className="flex justify-end gap-2">
                            <Button
                              size="xs"
                              variant="outline"
                              startIcon={<PencilIcon />}
                              onClick={() => startEdit(entry)}
                              disabled={pendingId === entry.telegram_user_id}
                            >
                              Изменить
                            </Button>
                            <Button
                              size="xs"
                              variant="outline"
                              startIcon={<TrashIcon />}
                              onClick={() => handleRemove(entry.telegram_user_id)}
                              disabled={pendingId === entry.telegram_user_id}
                            >
                              Удалить
                            </Button>
                          </div>
                        )}
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
