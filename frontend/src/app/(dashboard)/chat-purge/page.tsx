'use client';

import { ChangeEvent, useEffect, useState } from 'react';
import {
  chatMembersApi,
  ChatMembersSummary,
  ChatPurgeCandidate,
  ChatPurgeExecuteResult,
} from '@/api/chatMembers';
import Button from '@/components/ui/button/Button';
import Checkbox from '@/components/form/input/Checkbox';
import FileInput from '@/components/form/input/FileInput';
import Label from '@/components/form/Label';
import { Table, TableBody, TableCell, TableHeader, TableRow } from '@/components/ui/table';

export default function ChatPurgePage() {
  const [summary, setSummary] = useState<ChatMembersSummary | null>(null);
  const [eventName, setEventName] = useState('');
  const [candidates, setCandidates] = useState<ChatPurgeCandidate[]>([]);
  const [protectedGiftOwners, setProtectedGiftOwners] = useState(0);
  const [selected, setSelected] = useState<Set<number>>(new Set());

  const [isLoading, setIsLoading] = useState(true);
  const [isImporting, setIsImporting] = useState(false);
  const [isPurging, setIsPurging] = useState(false);
  const [kickingId, setKickingId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [result, setResult] = useState<ChatPurgeExecuteResult | null>(null);

  useEffect(() => {
    loadAll();
  }, []);

  const loadAll = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const [summaryResp, candidatesResp] = await Promise.all([
        chatMembersApi.getSummary(),
        chatMembersApi.getCandidates(),
      ]);
      setSummary(summaryResp);
      setEventName(candidatesResp.event_name);
      setCandidates(candidatesResp.candidates);
      setProtectedGiftOwners(candidatesResp.protected_gift_owners);
      // Все кандидаты предотмечены.
      setSelected(new Set(candidatesResp.candidates.map((c) => c.user_id)));
    } catch (err) {
      setError('Не удалось загрузить данные чистки чата');
      console.error('Failed to load chat purge data:', { operation: 'load', error: err });
    } finally {
      setIsLoading(false);
    }
  };

  const handleImport = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;
    try {
      setIsImporting(true);
      setError(null);
      setNotice(null);
      const importResult = await chatMembersApi.importCsv(file);
      setNotice(
        `Импортировано: ${importResult.imported}, пропущено строк: ${importResult.skipped_rows}, всего в таблице: ${importResult.total_in_table}`,
      );
      await loadAll();
    } catch (err) {
      setError('Не удалось импортировать CSV');
      console.error('Failed to import chat members CSV:', { operation: 'import', error: err });
    } finally {
      setIsImporting(false);
      event.target.value = '';
    }
  };

  const toggle = (userId: number, checked: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) {
        next.add(userId);
      } else {
        next.delete(userId);
      }
      return next;
    });
  };

  const toggleAll = (checked: boolean) => {
    setSelected(checked ? new Set(candidates.map((c) => c.user_id)) : new Set());
  };

  const handlePurge = async () => {
    const userIds = Array.from(selected);
    if (userIds.length === 0) return;
    if (!window.confirm(`Кикнуть ${userIds.length} чел.? Действие необратимо.`)) {
      return;
    }
    try {
      setIsPurging(true);
      setError(null);
      setResult(null);
      const purgeResult = await chatMembersApi.execute(userIds);
      setResult(purgeResult);
      await loadAll();
    } catch (err) {
      setError('Не удалось выполнить чистку. Возможно, функция не настроена (токен бота / публичный чат).');
      console.error('Failed to execute chat purge:', { operation: 'execute', error: err });
    } finally {
      setIsPurging(false);
    }
  };

  const handleKickOne = async (candidate: ChatPurgeCandidate) => {
    if (!window.confirm(`Кикнуть ${candidate.label}? Действие необратимо.`)) {
      return;
    }
    try {
      setKickingId(candidate.user_id);
      setError(null);
      setResult(null);
      const purgeResult = await chatMembersApi.execute([candidate.user_id]);
      setResult(purgeResult);
      await loadAll();
    } catch (err) {
      setError('Не удалось кикнуть участника. Возможно, функция не настроена (токен бота / публичный чат).');
      console.error('Failed to kick chat member:', { operation: 'kick_one', error: err });
    } finally {
      setKickingId(null);
    }
  };

  const busy = isPurging || kickingId !== null;
  const allSelected = candidates.length > 0 && selected.size === candidates.length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-800 dark:text-white/90">Чистка чата</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Удаление участников публичного чата без приза за активное событие
          {eventName ? ` «${eventName}»` : ''}.
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-error-300 bg-error-50 px-4 py-3 text-sm text-error-700 dark:border-error-800 dark:bg-error-500/10 dark:text-error-400">
          {error}
        </div>
      )}
      {notice && (
        <div className="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:border-gray-800 dark:bg-white/[0.03] dark:text-gray-300">
          {notice}
        </div>
      )}

      <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="flex flex-wrap items-center gap-4 text-sm text-gray-600 dark:text-gray-300">
          <span>В чате: <b>{summary?.total ?? '—'}</b></span>
          <span>Админов: <b>{summary?.admins ?? '—'}</b></span>
          <span>Ботов: <b>{summary?.bots ?? '—'}</b></span>
        </div>
        <div className="mt-4">
          <Label htmlFor="chat-members-csv">Обновить список из CSV</Label>
          <FileInput id="chat-members-csv" accept=".csv" onChange={handleImport} disabled={isImporting} />
          {isImporting && (
            <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">Импорт…</p>
          )}
        </div>
      </div>

      {result && (
        <div className="rounded-lg border border-success-300 bg-success-50 px-4 py-3 text-sm text-success-700 dark:border-success-800 dark:bg-success-500/10 dark:text-success-400">
          Кикнуто: {result.kicked}, ошибок: {result.failed}, пропущено: {result.skipped}, защищено: {result.protected}
        </div>
      )}

      <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03]">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div className="text-sm text-gray-600 dark:text-gray-300">
            Кандидатов: <b>{candidates.length}</b> · Защищено обладателей приза: <b>{protectedGiftOwners}</b>
          </div>
          <Button
            size="sm"
            variant="primary"
            onClick={handlePurge}
            disabled={busy || selected.size === 0}
          >
            {isPurging ? 'Выполняется удаление…' : `Кикнуть выбранных (${selected.size})`}
          </Button>
        </div>

        {isPurging && (
          <p className="mb-3 text-sm text-warning-600 dark:text-warning-400">
            Выполняется удаление, не закрывайте вкладку…
          </p>
        )}

        {isLoading ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">Загрузка…</p>
        ) : candidates.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Кандидатов на удаление нет.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader className="border-b border-gray-100 dark:border-gray-800">
                <TableRow>
                  <TableCell isHeader className="w-12 px-4 py-3">
                    <Checkbox checked={allSelected} onChange={toggleAll} />
                  </TableCell>
                  <TableCell isHeader className="px-4 py-3 text-left text-sm font-medium text-gray-500 dark:text-gray-400">
                    Участник
                  </TableCell>
                  <TableCell isHeader className="px-4 py-3 text-left text-sm font-medium text-gray-500 dark:text-gray-400">
                    Причина
                  </TableCell>
                  <TableCell isHeader className="w-28 px-4 py-3 text-right text-sm font-medium text-gray-500 dark:text-gray-400">
                    Действие
                  </TableCell>
                </TableRow>
              </TableHeader>
              <TableBody>
                {candidates.map((candidate) => (
                  <TableRow key={candidate.user_id} className="border-b border-gray-100 dark:border-gray-800">
                    <TableCell className="px-4 py-3">
                      <Checkbox
                        checked={selected.has(candidate.user_id)}
                        onChange={(checked) => toggle(candidate.user_id, checked)}
                      />
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                      {candidate.label}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                      {candidate.reason}
                    </TableCell>
                    <TableCell className="px-4 py-3 text-right">
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={() => handleKickOne(candidate)}
                        disabled={busy}
                      >
                        {kickingId === candidate.user_id ? 'Удаление…' : 'Кикнуть'}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </div>
  );
}
