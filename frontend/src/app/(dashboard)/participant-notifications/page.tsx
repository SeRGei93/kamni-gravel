'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Button from '@/components/ui/button/Button';
import Checkbox from '@/components/form/input/Checkbox';
import TextArea from '@/components/form/input/TextArea';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '@/components/ui/table';
import {
  ParticipantNotificationFilter,
  ParticipantNotificationJob,
  ParticipantNotificationRecipient,
  participantNotificationsApi,
} from '@/api/participantNotifications';

type FilterOption = {
  value: ParticipantNotificationFilter;
  label: string;
};

const FILTER_OPTIONS: FilterOption[] = [
  { value: 'all', label: 'Все' },
  { value: 'finished_without_gift', label: 'Проехал, но не добавил приз' },
  { value: 'gift_without_finish', label: 'Добавил приз, но не проехал' },
  { value: 'pending_manual_gift_owners', label: 'Не распределил призы' },
];

const TELEGRAM_MESSAGE_LIMIT = 4096;

function recipientDetails(recipient: ParticipantNotificationRecipient): string {
  const details = [recipient.status];
  details.push(recipient.has_gift ? 'приз добавлен' : 'приз не добавлен');
  if (recipient.has_pending_manual_gifts) {
    details.push('есть приз с ручным распределением без получателя');
  }
  return details.join(' · ');
}

function notificationJobMessage(job: ParticipantNotificationJob): string {
  switch (job.status) {
    case 'queued':
      return `Рассылка поставлена в очередь: ${job.requested} получателей.`;
    case 'running':
      return `Отправка: доставлено ${job.sent} из ${job.requested}, ошибок: ${job.failed}.`;
    case 'completed':
      return `Рассылка завершена. Отправлено: ${job.sent}, ошибок: ${job.failed}, пропущено: ${job.skipped}.`;
    case 'cancelled':
      return job.error || 'Рассылка остановлена.';
    case 'failed':
      return job.error || 'Не удалось выполнить рассылку.';
  }
}

export default function ParticipantNotificationsPage() {
  const [filter, setFilter] = useState<ParticipantNotificationFilter>('all');
  const [eventName, setEventName] = useState('');
  const [recipients, setRecipients] = useState<ParticipantNotificationRecipient[]>([]);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [text, setText] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [job, setJob] = useState<ParticipantNotificationJob | null>(null);
  const isSending = job?.status === 'queued' || job?.status === 'running';
  const activeJobID = job?.id;

  const loadRecipients = useCallback(async (nextFilter: ParticipantNotificationFilter) => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await participantNotificationsApi.getRecipients(nextFilter);
      setEventName(response.event_name);
      setRecipients(response.recipients);
      setSelected(new Set());
    } catch (err) {
      setRecipients([]);
      setSelected(new Set());
      setError('Не удалось загрузить участников для рассылки. Проверьте активное событие.');
      console.error('Failed to load participant notification recipients:', {
        operation: 'load_recipients',
        filter: nextFilter,
        error: err,
      });
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadRecipients(filter);
  }, [filter, loadRecipients]);

  useEffect(() => {
    if (!activeJobID || !isSending) {
      return;
    }

    let active = true;
    const poll = async () => {
      try {
        const nextJob = await participantNotificationsApi.getJob(activeJobID);
        if (active) {
          setJob(nextJob);
        }
      } catch (err) {
        if (active) {
          setError('Не удалось получить статус рассылки. Обновите страницу через несколько секунд.');
        }
        console.error('Failed to load participant notification job:', {
          operation: 'load_notification_job',
          job_id: activeJobID,
          error: err,
        });
      }
    };

    void poll();
    const interval = window.setInterval(() => {
      void poll();
    }, 1000);

    return () => {
      active = false;
      window.clearInterval(interval);
    };
  }, [activeJobID, isSending]);

  const characterCount = useMemo(() => Array.from(text).length, [text]);
  const selectedCount = selected.size;
  const allSelected = recipients.length > 0 && selectedCount === recipients.length;
  const canSend = !isSending && selectedCount > 0 && text.trim().length > 0 && characterCount <= TELEGRAM_MESSAGE_LIMIT;

  const changeFilter = (nextFilter: ParticipantNotificationFilter) => {
    setFilter(nextFilter);
  };

  const toggleRecipient = (userId: number, checked: boolean) => {
    setSelected((previous) => {
      const next = new Set(previous);
      if (checked) {
        next.add(userId);
      } else {
        next.delete(userId);
      }
      return next;
    });
  };

  const toggleAll = (checked: boolean) => {
    setSelected(checked ? new Set(recipients.map((recipient) => recipient.user_id)) : new Set());
  };

  const handleSend = async () => {
    const userIds = Array.from(selected);
    if (!canSend || !window.confirm(`Отправить сообщение ${userIds.length} участникам?`)) {
      return;
    }

    try {
      setError(null);
      const notificationJob = await participantNotificationsApi.send(userIds, text.trim());
      setJob(notificationJob);
      setSelected(new Set());
    } catch (err) {
      setError('Не удалось отправить уведомления. Проверьте текст и настройки Telegram-бота.');
      console.error('Failed to send participant notifications:', {
        operation: 'send_notifications',
        selected_count: userIds.length,
        text_length: characterCount,
        error: err,
      });
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-800 dark:text-white/90">Уведомления участникам</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Выберите участников активного события{eventName ? ` «${eventName}»` : ''} и отправьте им личное сообщение в Telegram.
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-error-300 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-800 dark:bg-error-500/10 dark:text-error-400">
          {error}
        </div>
      )}

      {job && (
        <div className={`rounded-lg border px-4 py-3 text-sm ${job.status === 'completed'
          ? 'border-success-300 bg-success-50 text-success-700 dark:border-success-800 dark:bg-success-500/10 dark:text-success-400'
          : job.status === 'failed' || job.status === 'cancelled'
            ? 'border-error-300 bg-error-50 text-error-700 dark:border-error-800 dark:bg-error-500/10 dark:text-error-400'
            : 'border-warning-300 bg-warning-50 text-warning-700 dark:border-warning-800 dark:bg-warning-500/10 dark:text-warning-400'}`}>
          {notificationJobMessage(job)}
        </div>
      )}

      <div className="grid gap-6 xl:grid-cols-2">
        <section className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03]">
          <div className="mb-4 flex items-start justify-between gap-4">
            <div>
              <h2 className="text-lg font-semibold text-gray-800 dark:text-white/90">Текст уведомления</h2>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                Сообщение придёт в личный чат от бота.
              </p>
            </div>
            <span className={`shrink-0 text-sm ${characterCount > TELEGRAM_MESSAGE_LIMIT ? 'text-error-600 dark:text-error-400' : 'text-gray-500 dark:text-gray-400'}`}>
              {characterCount}/{TELEGRAM_MESSAGE_LIMIT}
            </span>
          </div>

          <TextArea
            value={text}
            onChange={setText}
            rows={14}
            error={characterCount > TELEGRAM_MESSAGE_LIMIT}
            placeholder="Напишите сообщение участникам…"
            hint={characterCount > TELEGRAM_MESSAGE_LIMIT ? 'Telegram принимает не более 4096 символов.' : ''}
            disabled={isSending}
            required
          />

          <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Выбрано: <b>{selectedCount}</b>
            </p>
            <Button
              size="sm"
              variant="primary"
              onClick={handleSend}
              disabled={!canSend}
              startIcon={<span aria-hidden="true">✉️</span>}
            >
              {isSending ? 'Рассылка выполняется…' : `Отправить выбранным (${selectedCount})`}
            </Button>
          </div>
          {isSending && (
            <p className="mt-3 text-sm text-warning-600 dark:text-warning-400">
              Рассылка выполняется в фоне: страницу можно не держать открытой.
            </p>
          )}
        </section>

        <section className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03]">
          <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold text-gray-800 dark:text-white/90">Получатели</h2>
              <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                Отметьте только тех, кому нужно отправить сообщение.
              </p>
            </div>
            <label className="text-sm text-gray-700 dark:text-gray-300">
              <span className="sr-only">Фильтр участников</span>
              <select
                value={filter}
                onChange={(event) => changeFilter(event.target.value as ParticipantNotificationFilter)}
                disabled={isLoading || isSending}
                className="rounded-lg border border-gray-300 bg-transparent px-3 py-2 text-sm text-gray-800 outline-hidden focus:border-brand-300 focus:ring-3 focus:ring-brand-500/10 disabled:cursor-not-allowed disabled:opacity-60 dark:border-gray-700 dark:bg-gray-900 dark:text-white/90 dark:focus:border-brand-800"
              >
                {FILTER_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </label>
          </div>

          {isLoading ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">Загрузка…</p>
          ) : recipients.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">Участников по этому фильтру нет.</p>
          ) : (
            <div className="max-h-[34rem] overflow-auto rounded-lg border border-gray-100 dark:border-gray-800">
              <Table>
                <TableHeader className="sticky top-0 border-b border-gray-100 bg-white dark:border-gray-800 dark:bg-gray-900">
                  <TableRow>
                    <TableCell isHeader className="w-12 px-4 py-3">
                      <Checkbox checked={allSelected} onChange={toggleAll} disabled={isSending} />
                    </TableCell>
                    <TableCell isHeader className="px-4 py-3 text-left text-sm font-medium text-gray-500 dark:text-gray-400">
                      Участник
                    </TableCell>
                    <TableCell isHeader className="px-4 py-3 text-left text-sm font-medium text-gray-500 dark:text-gray-400">
                      Статус
                    </TableCell>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {recipients.map((recipient) => (
                    <TableRow key={recipient.user_id} className="border-b border-gray-100 last:border-b-0 dark:border-gray-800">
                      <TableCell className="px-4 py-3">
                        <Checkbox
                          checked={selected.has(recipient.user_id)}
                          onChange={(checked) => toggleRecipient(recipient.user_id, checked)}
                          disabled={isSending}
                        />
                      </TableCell>
                      <TableCell className="px-4 py-3 text-sm font-medium text-gray-700 dark:text-gray-300">
                        {recipient.label}
                      </TableCell>
                      <TableCell className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                        {recipientDetails(recipient)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
