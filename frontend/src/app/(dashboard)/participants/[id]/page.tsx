'use client';

import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { participantsApi } from '@/api/participants';
import { resultsApi } from '@/api/results';
import type { ParticipantDetail, Gift, Result, PrizeGiftAssignment } from '@/types';
import Badge from '@/components/ui/badge/Badge';
import Button from '@/components/ui/button/Button';
import Input from '@/components/form/input/InputField';
import TextArea from '@/components/form/input/TextArea';
import TimeInput from '@/components/participants/TimeInput';
import Label from '@/components/form/Label';
import ResultCriteriaManager from '@/components/participants/ResultCriteriaManager';
import { getCriteriaColor } from '@/utils/criteria';
import { fromMinskDateTimeInput, toMinskDateTimeInput } from '@/utils/minskTime';
import { secondsToTimeString } from '@/utils/time';
import { formatSpeed, formatDistanceKm, metersToKm, kmToMeters } from '@/utils/format';
import { ChevronLeftIcon, PencilIcon, CheckLineIcon, CloseLineIcon, PlusIcon, TrashIcon } from '@/icons';
import Link from 'next/link';

const GENDER_LABELS: Record<string, string> = {
  male: 'Мужской',
  female: 'Женский',
};

const BIKE_TYPE_LABELS: Record<string, string> = {
  gravel: 'Гравийник',
  mtb: 'МТБ',
  road: 'Шоссе',
  single_speed: 'Фикс',
  tandem: 'Тандем',
};

function formatPrizeAssignment(assignment: PrizeGiftAssignment): string {
  const target = assignment.target_rank
    ? `место ${assignment.target_rank}`
    : 'без привязки к месту';
  const assigned = `выдано месту ${assignment.assigned_rank}`;

  if (assignment.is_fallback) {
    return `${target} -> ${assigned}`;
  }

  return target === 'без привязки к месту' ? assigned : target;
}

// Ячейка «подпись + значение» для блока результата. Скрывается, если значение пустое.
function Metric({ label, value }: { label: string; value?: string | null }) {
  if (!value) return null;
  return (
    <div>
      <p className="mb-1 text-xs text-gray-500 dark:text-gray-400">{label}</p>
      <p className="text-sm font-medium text-gray-800 dark:text-white/90">{value}</p>
    </div>
  );
}

// Парсит необязательное числовое поле формы (поддерживает запятую как десятичный разделитель).
function parseOptionalNumber(value: string): number | undefined {
  const normalized = value.trim().replace(',', '.');
  if (normalized === '') return undefined;
  const parsed = Number(normalized);
  return Number.isFinite(parsed) ? parsed : undefined;
}

// То же, но округляет до целого — для полей, которые на бэкенде имеют тип int
// (пульс, каденс, калории): дробное значение иначе вызвало бы 400 при декодировании.
function parseOptionalInt(value: string): number | undefined {
  const parsed = parseOptionalNumber(value);
  return parsed === undefined ? undefined : Math.round(parsed);
}

export default function ParticipantDetailPage() {
  const params = useParams();
  const router = useRouter();
  const participantId = Number(params.id);

  const [participant, setParticipant] = useState<ParticipantDetail | null>(null);
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [currentResult, setCurrentResult] = useState<Result | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isEditingNotes, setIsEditingNotes] = useState(false);
  const [isSavingNotes, setIsSavingNotes] = useState(false);
  const [isDeletingParticipant, setIsDeletingParticipant] = useState(false);

  // Редактируемые поля участника
  const [notes, setNotes] = useState('');
  
  // Редактируемые поля результата
  const [elapsedTimeSec, setElapsedTimeSec] = useState<number | undefined>();
  const [movingTimeSec, setMovingTimeSec] = useState<number | undefined>();
  const [resultLink, setResultLink] = useState('');
  const [isEditingResult, setIsEditingResult] = useState(false);
  const [isSavingResult, setIsSavingResult] = useState(false);

  // Метрики заезда из Стравы (строки полей формы)
  const [startedAt, setStartedAt] = useState('');
  const [finishedAt, setFinishedAt] = useState('');
  const [distanceKm, setDistanceKm] = useState('');
  const [avgHeartRate, setAvgHeartRate] = useState('');
  const [maxHeartRate, setMaxHeartRate] = useState('');
  const [avgCadence, setAvgCadence] = useState('');
  const [calories, setCalories] = useState('');
  const [peakSpeedKmh, setPeakSpeedKmh] = useState('');

  // Инициализирует поля метрик из текущего результата (или очищает их).
  // Новые поля берутся из результата (resultsApi), а не из участника —
  // ParticipantDTO их не зеркалит.
  const fillMetricFieldsFromResult = useCallback((result: Result | null) => {
    setStartedAt(result?.started_at ? toMinskDateTimeInput(result.started_at) : '');
    setFinishedAt(result?.finished_at ? toMinskDateTimeInput(result.finished_at) : '');
    const km = metersToKm(result?.distance_meters);
    setDistanceKm(km !== undefined ? String(km) : '');
    setAvgHeartRate(result?.avg_heart_rate !== undefined ? String(result.avg_heart_rate) : '');
    setMaxHeartRate(result?.max_heart_rate !== undefined ? String(result.max_heart_rate) : '');
    setAvgCadence(result?.avg_cadence !== undefined ? String(result.avg_cadence) : '');
    setCalories(result?.calories !== undefined ? String(result.calories) : '');
    setPeakSpeedKmh(result?.peak_speed_kmh !== undefined ? String(result.peak_speed_kmh) : '');
  }, []);

  const loadParticipant = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);

      // Загружаем данные параллельно
      const [participantData, giftsData, resultsData] = await Promise.all([
        participantsApi.getById(participantId),
        participantsApi.getGifts(participantId),
        resultsApi.getByParticipant(participantId),
      ]);

      setParticipant(participantData);
      setGifts(giftsData.gifts);
      
      // Находим текущий результат
      const current = resultsData.results.find((r) => r.is_current) || null;
      setCurrentResult(current);

      setElapsedTimeSec(participantData.elapsed_time_sec);
      setMovingTimeSec(participantData.moving_time_sec);
      setResultLink(participantData.result_link || '');
      fillMetricFieldsFromResult(current);
      setNotes(participantData.notes || '');
    } catch (err) {
      setError('Ошибка загрузки данных участника');
      console.error('Failed to load participant:', err);
    } finally {
      setIsLoading(false);
    }
  }, [participantId, fillMetricFieldsFromResult]);

  useEffect(() => {
    loadParticipant();
  }, [loadParticipant]);

  // Точечное обновление только блока «Подобранные призы».
  // Привязка/отвязка критерия пересчитывает подобранные призы на сервере,
  // поэтому достаточно перезапросить участника и обновить только matched-поля —
  // без полноэкранного спиннера и без сброса редактируемых полей результата/заметок.
  const refreshMatchedGifts = useCallback(async () => {
    try {
      const data = await participantsApi.getById(participantId);
      setParticipant((prev) =>
        prev
          ? {
              ...prev,
              matched_gifts: data.matched_gifts,
              matched_gift_assignments: data.matched_gift_assignments,
            }
          : data
      );
    } catch (err) {
      console.error('Failed to refresh matched gifts after criteria change:', err);
    }
  }, [participantId]);

  const handleSaveNotes = async () => {
    if (!participant) return;

    try {
      setIsSavingNotes(true);
      await participantsApi.update(participantId, {
        notes: notes || undefined,
      });
      setIsEditingNotes(false);
      await loadParticipant(); // Перезагружаем данные
    } catch (err) {
      setError('Ошибка сохранения заметок');
      console.error('Failed to update participant notes:', err);
    } finally {
      setIsSavingNotes(false);
    }
  };

  const handleCancelNotesEdit = () => {
    if (!participant) return;
    setNotes(participant.notes || '');
    setIsEditingNotes(false);
  };

  // Получаем ID текущего результата (для обновления времени)
  const getCurrentResultId = async (): Promise<number | null> => {
    try {
      const response = await resultsApi.getByParticipant(participantId);
      const currentResult = response.results.find((r) => r.is_current);
      return currentResult?.id || null;
    } catch {
      return null;
    }
  };

  const handleStartResultEdit = () => {
    if (!participant) return;
    setElapsedTimeSec(participant.elapsed_time_sec);
    setMovingTimeSec(participant.moving_time_sec);
    setResultLink(participant.result_link || '');
    fillMetricFieldsFromResult(currentResult);
    setIsEditingResult(true);
  };

  const handleStartResultCreate = () => {
    setElapsedTimeSec(undefined);
    setMovingTimeSec(undefined);
    setResultLink('');
    fillMetricFieldsFromResult(null);
    setIsEditingResult(true);
  };

  const handleSaveResult = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!participant) return;

    const startedAtIso = startedAt ? fromMinskDateTimeInput(startedAt) : undefined;
    const finishedAtIso = finishedAt ? fromMinskDateTimeInput(finishedAt) : undefined;
    const hasStartFinish = Boolean(startedAtIso && finishedAtIso);

    // Общее время: либо из старт/финиш, либо из «Общего времени».
    if (!hasStartFinish && (elapsedTimeSec === undefined || elapsedTimeSec <= 0)) {
      setError('Укажите общее время или обе метки: время старта и финиша');
      return;
    }

    let totalSec = elapsedTimeSec;
    if (hasStartFinish) {
      const diffSec = Math.round(
        (new Date(finishedAtIso as string).getTime() -
          new Date(startedAtIso as string).getTime()) /
          1000
      );
      if (diffSec <= 0) {
        setError('Время финиша должно быть позже времени старта');
        return;
      }
      totalSec = diffSec;
    }

    if (
      movingTimeSec !== undefined &&
      totalSec !== undefined &&
      movingTimeSec > totalSec
    ) {
      setError('Время в движении не может быть больше общего времени');
      return;
    }

    const metrics = {
      elapsed_time_sec: hasStartFinish ? undefined : elapsedTimeSec,
      moving_time_sec: movingTimeSec,
      started_at: startedAtIso,
      finished_at: finishedAtIso,
      distance_meters: kmToMeters(parseOptionalNumber(distanceKm)),
      avg_heart_rate: parseOptionalInt(avgHeartRate),
      max_heart_rate: parseOptionalInt(maxHeartRate),
      peak_speed_kmh: parseOptionalNumber(peakSpeedKmh),
      avg_cadence: parseOptionalInt(avgCadence),
      calories: parseOptionalInt(calories),
    };

    try {
      setIsSavingResult(true);
      setError(null);
      const resultId = await getCurrentResultId();

      if (resultId) {
        await resultsApi.update(resultId, metrics);
      } else {
        await resultsApi.create(participant.id, {
          ...metrics,
          result_link: resultLink.trim() || undefined,
        });
      }

      setIsEditingResult(false);
      await loadParticipant();
    } catch (err) {
      setError('Ошибка сохранения результата');
      console.error('Failed to save result:', {
        operation: currentResult ? 'update_result' : 'create_result',
        participant_id: participant.id,
        error: err,
      });
    } finally {
      setIsSavingResult(false);
    }
  };

  const handleCancelResultEdit = () => {
    if (!participant) return;
    setElapsedTimeSec(participant.elapsed_time_sec);
    setMovingTimeSec(participant.moving_time_sec);
    setResultLink(participant.result_link || '');
    fillMetricFieldsFromResult(currentResult);
    setIsEditingResult(false);
  };

  const handleDeleteParticipant = async () => {
    if (!participant) return;
    if (
      !window.confirm(
        `Удалить участника ${participant.first_name || participant.username || participant.user_id}?`
      )
    ) {
      return;
    }

    try {
      setIsDeletingParticipant(true);
      setError(null);
      await participantsApi.delete(participant.id);
      router.push('/participants');
    } catch (err) {
      setError('Ошибка удаления участника');
      console.error('Failed to delete participant:', {
        operation: 'delete_participant',
        participant_id: participant.id,
        event_id: participant.event_id,
        error: err,
      });
    } finally {
      setIsDeletingParticipant(false);
    }
  };


  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-gray-500 dark:text-gray-400">Загрузка...</div>
      </div>
    );
  }

  if (error || !participant) {
    return (
      <div className="rounded-lg border border-error-200 bg-error-50 p-4 dark:border-error-800 dark:bg-error-900/20">
        <p className="text-error-600 dark:text-error-400">
          {error || 'Участник не найден'}
        </p>
        <Link
          href="/participants"
          className="mt-2 inline-flex items-center gap-2 text-sm text-error-600 underline dark:text-error-400"
        >
          <ChevronLeftIcon />
          Вернуться к списку
        </Link>
      </div>
    );
  }

  // Определяем категории участника
  const categories: string[] = [];
  if (participant.gender === 'male') categories.push('Мужской зачёт');
  if (participant.gender === 'female') categories.push('Женский зачёт');
  categories.push(BIKE_TYPE_LABELS[participant.bike_type] || participant.bike_type);

  // Живой предпросмотр вычисляемых полей в форме редактирования.
  const previewStartedIso = startedAt ? fromMinskDateTimeInput(startedAt) : undefined;
  const previewFinishedIso = finishedAt ? fromMinskDateTimeInput(finishedAt) : undefined;
  let previewTotalSec: number | undefined = elapsedTimeSec;
  if (previewStartedIso && previewFinishedIso) {
    const diff = Math.round(
      (new Date(previewFinishedIso).getTime() - new Date(previewStartedIso).getTime()) / 1000
    );
    previewTotalSec = diff > 0 ? diff : undefined;
  }
  const previewDistanceMeters = kmToMeters(parseOptionalNumber(distanceKm));
  const previewIdleSec =
    previewTotalSec !== undefined && movingTimeSec !== undefined && previewTotalSec >= movingTimeSec
      ? previewTotalSec - movingTimeSec
      : undefined;
  const computeSpeed = (meters?: number, sec?: number) =>
    meters && meters > 0 && sec && sec > 0 ? (meters / 1000) / (sec / 3600) : undefined;
  const previewAvgSpeed = computeSpeed(previewDistanceMeters, previewTotalSec);
  const previewAvgMovingSpeed = computeSpeed(previewDistanceMeters, movingTimeSec);

  return (
    <div className="space-y-6">
      {/* Заголовок */}
      <div className="flex items-center justify-between">
        <div>
          <Link
            href="/participants"
            className="mb-2 inline-flex items-center gap-2 text-sm text-gray-600 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200"
          >
            <ChevronLeftIcon />
            Назад к списку участников
          </Link>
          <h1 className="text-2xl font-semibold text-gray-800 dark:text-white">
            {participant.first_name} {participant.last_name}
          </h1>
          <p className="mt-1 text-gray-600 dark:text-gray-400">
            @{participant.username || `user${participant.user_id}`}
          </p>
        </div>
        <Button
          size="sm"
          variant="outline"
          startIcon={<TrashIcon />}
          onClick={handleDeleteParticipant}
          disabled={isDeletingParticipant}
        >
          {isDeletingParticipant ? 'Удаление...' : 'Удалить'}
        </Button>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Основная информация */}
        <div className="lg:col-span-2 space-y-6">
          {/* Информация об участнике */}
          <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
            <h3 className="mb-4 text-lg font-semibold text-gray-800 dark:text-white">
              Информация
            </h3>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <p className="mb-1 text-xs text-gray-500 dark:text-gray-400">
                  Telegram ID
                </p>
                <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                  {participant.user_id}
                </p>
              </div>
              <div>
                <p className="mb-1 text-xs text-gray-500 dark:text-gray-400">
                  Пол
                </p>
                <Badge
                  color={participant.gender === 'male' ? 'info' : 'warning'}
                  size="sm"
                >
                  {GENDER_LABELS[participant.gender]}
                </Badge>
              </div>
              <div>
                <p className="mb-1 text-xs text-gray-500 dark:text-gray-400">
                  Тип велосипеда
                </p>
                <Badge color="light" size="sm">
                  {BIKE_TYPE_LABELS[participant.bike_type]}
                </Badge>
              </div>
              <div>
                <p className="mb-1 text-xs text-gray-500 dark:text-gray-400">
                  Дата регистрации
                </p>
                <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                  {new Date(participant.registered_at).toLocaleDateString('ru-RU')}
                </p>
              </div>
            </div>

            {/* Категории */}
            <div className="mt-4">
              <p className="mb-2 text-xs text-gray-500 dark:text-gray-400">
                Категории
              </p>
              <div className="flex flex-wrap gap-2">
                {categories.map((cat, idx) => (
                  <Badge key={idx} color="info" size="sm">
                    {cat}
                  </Badge>
                ))}
              </div>
            </div>

            {/* Места */}
            {(participant.place_absolute || participant.place_by_gender || participant.place_by_gender_bike) && (
              <div className="mt-4">
                <p className="mb-2 text-xs text-gray-500 dark:text-gray-400">
                  Места в зачётах
                </p>
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                  {participant.place_absolute && (
                    <div>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        Абсолютный
                      </p>
                      <p className="text-lg font-semibold text-gray-800 dark:text-white">
                        {participant.place_absolute}
                      </p>
                    </div>
                  )}
                  {participant.place_by_gender && (
                    <div>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        По гендеру
                      </p>
                      <p className="text-lg font-semibold text-gray-800 dark:text-white">
                        {participant.place_by_gender}
                      </p>
                    </div>
                  )}
                  {participant.place_by_gender_bike && (
                    <div>
                      <p className="text-xs text-gray-500 dark:text-gray-400">
                        Гендер+тип
                      </p>
                      <p className="text-lg font-semibold text-gray-800 dark:text-white">
                        {participant.place_by_gender_bike}
                      </p>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Заметки администратора */}
          <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-800 dark:text-white">
                Заметки администратора
              </h3>
              {!isEditingNotes ? (
                <Button
                  size="sm"
                  variant="outline"
                  startIcon={<PencilIcon />}
                  onClick={() => setIsEditingNotes(true)}
                >
                  Редактировать
                </Button>
              ) : (
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    startIcon={<CloseLineIcon />}
                    onClick={handleCancelNotesEdit}
                    disabled={isSavingNotes}
                  >
                    Отмена
                  </Button>
                  <Button
                    size="sm"
                    startIcon={<CheckLineIcon />}
                    onClick={handleSaveNotes}
                    disabled={isSavingNotes}
                  >
                    {isSavingNotes ? 'Сохранение...' : 'Сохранить'}
                  </Button>
                </div>
              )}
            </div>
            {isEditingNotes ? (
              <TextArea
                placeholder="Введите заметки..."
                value={notes}
                onChange={setNotes}
                rows={4}
              />
            ) : (
              <p className="text-sm text-gray-800 dark:text-white/90">
                {participant.notes || 'Нет заметок'}
              </p>
            )}
          </div>

          {/* Результат и Время */}
          <form
            onSubmit={handleSaveResult}
            className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6"
          >
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-800 dark:text-white">
                Результат
              </h3>
              {participant.is_finished && !isEditingResult && (
                <Button
                  size="sm"
                  variant="outline"
                  startIcon={<PencilIcon />}
                  onClick={handleStartResultEdit}
                >
                  Изменить время
                </Button>
              )}
              {!participant.is_finished && !isEditingResult && (
                <Button
                  size="sm"
                  variant="outline"
                  startIcon={<PlusIcon />}
                  onClick={handleStartResultCreate}
                >
                  Добавить результат
                </Button>
              )}
              {isEditingResult && (
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleCancelResultEdit}
                    disabled={isSavingResult}
                  >
                    Отмена
                  </Button>
                  <Button
                    type="submit"
                    size="sm"
                    disabled={isSavingResult}
                  >
                    {isSavingResult ? 'Сохранение...' : 'Сохранить'}
                  </Button>
                </div>
              )}
            </div>
            
            <div className="space-y-4">
              {/* Статус */}
              <div className="flex items-center gap-3">
                <Badge color={participant.is_finished ? 'success' : 'light'}>
                  {participant.is_finished ? 'Проехал' : 'Не проехал'}
                </Badge>
              </div>

              {/* Ссылка на результат */}
              {participant.result_link && (
                <div>
                  <p className="mb-1 text-xs text-gray-500 dark:text-gray-400">
                    Ссылка на результат
                  </p>
                  <a
                    href={participant.result_link}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-brand-500 hover:text-brand-600 dark:text-brand-400"
                  >
                    {participant.result_link}
                  </a>
                </div>
              )}

              {/* Дата отправки */}
              {participant.finished_at && (
                <div>
                  <p className="mb-1 text-xs text-gray-500 dark:text-gray-400">
                    Дата отправки результата
                  </p>
                  <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                    {new Date(participant.finished_at).toLocaleString('ru-RU')}
                  </p>
                </div>
              )}

              {/* Время */}
              {(participant.is_finished || isEditingResult) && (
                <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
                  {isEditingResult ? (
                    <div className="space-y-4">
                      {/* Старт / финиш заезда */}
                      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <div>
                          <Label>Время старта</Label>
                          <Input
                            type="datetime-local"
                            value={startedAt}
                            onChange={(event) => setStartedAt(event.target.value)}
                          />
                        </div>
                        <div>
                          <Label>Время финиша</Label>
                          <Input
                            type="datetime-local"
                            value={finishedAt}
                            onChange={(event) => setFinishedAt(event.target.value)}
                          />
                        </div>
                      </div>

                      {/* Время: общее (если не заданы старт/финиш) и в движении */}
                      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <TimeInput
                          label="Общее время (если не заданы старт/финиш)"
                          value={elapsedTimeSec}
                          onChange={setElapsedTimeSec}
                        />
                        <TimeInput
                          label="Время в движении"
                          value={movingTimeSec}
                          onChange={setMovingTimeSec}
                        />
                      </div>

                      {/* Метрики из Стравы */}
                      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                        <div>
                          <Label>Дистанция, км</Label>
                          <Input
                            type="number"
                            step={0.1}
                            min="0"
                            placeholder="например 202.3"
                            value={distanceKm}
                            onChange={(event) => setDistanceKm(event.target.value)}
                          />
                        </div>
                        <div>
                          <Label>Пиковая скорость, км/ч</Label>
                          <Input
                            type="number"
                            step={0.1}
                            min="0"
                            value={peakSpeedKmh}
                            onChange={(event) => setPeakSpeedKmh(event.target.value)}
                          />
                        </div>
                        <div>
                          <Label>Калории, ккал</Label>
                          <Input
                            type="number"
                            min="0"
                            value={calories}
                            onChange={(event) => setCalories(event.target.value)}
                          />
                        </div>
                        <div>
                          <Label>Средний пульс, уд/мин</Label>
                          <Input
                            type="number"
                            min="0"
                            value={avgHeartRate}
                            onChange={(event) => setAvgHeartRate(event.target.value)}
                          />
                        </div>
                        <div>
                          <Label>Максимальный пульс, уд/мин</Label>
                          <Input
                            type="number"
                            min="0"
                            value={maxHeartRate}
                            onChange={(event) => setMaxHeartRate(event.target.value)}
                          />
                        </div>
                        <div>
                          <Label>Средний каденс, об/мин</Label>
                          <Input
                            type="number"
                            min="0"
                            value={avgCadence}
                            onChange={(event) => setAvgCadence(event.target.value)}
                          />
                        </div>
                      </div>

                      {/* Живой предпросмотр расчётов */}
                      <div className="rounded-lg bg-gray-50 p-3 dark:bg-white/[0.03]">
                        <p className="mb-2 text-xs font-medium text-gray-500 dark:text-gray-400">
                          Рассчитывается автоматически
                        </p>
                        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                          <Metric
                            label="Общее время"
                            value={previewTotalSec !== undefined ? secondsToTimeString(previewTotalSec) : '-'}
                          />
                          <Metric
                            label="Простой"
                            value={previewIdleSec !== undefined ? secondsToTimeString(previewIdleSec) : '-'}
                          />
                          <Metric label="Ср. скорость" value={formatSpeed(previewAvgSpeed) || '-'} />
                          <Metric
                            label="Ср. скорость в движении"
                            value={formatSpeed(previewAvgMovingSpeed) || '-'}
                          />
                        </div>
                      </div>

                      {!currentResult && (
                        <div>
                          <p className="mb-1.5 text-sm font-medium text-gray-700 dark:text-gray-400">
                            Ссылка Strava
                          </p>
                          <Input
                            type="url"
                            placeholder="https://www.strava.com/activities/..."
                            value={resultLink}
                            onChange={(event) => setResultLink(event.target.value)}
                          />
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                      <Metric
                        label="Общее время"
                        value={currentResult?.elapsed_time || participant.elapsed_time || '-'}
                      />
                      <Metric
                        label="Время в движении"
                        value={currentResult?.moving_time || participant.moving_time || '-'}
                      />
                      <Metric label="Простой" value={currentResult?.idle_time || '-'} />
                      <Metric label="Ср. скорость" value={formatSpeed(currentResult?.avg_speed_kmh) || '-'} />
                      <Metric
                        label="Ср. скорость в движении"
                        value={formatSpeed(currentResult?.avg_moving_speed_kmh) || '-'}
                      />
                      <Metric label="Дата проезда" value={currentResult?.ride_date} />
                      <Metric label="Дистанция" value={formatDistanceKm(currentResult?.distance_meters)} />
                      <Metric label="Пиковая скорость" value={formatSpeed(currentResult?.peak_speed_kmh)} />
                      <Metric
                        label="Средний пульс"
                        value={currentResult?.avg_heart_rate !== undefined ? `${currentResult.avg_heart_rate} уд/мин` : undefined}
                      />
                      <Metric
                        label="Максимальный пульс"
                        value={currentResult?.max_heart_rate !== undefined ? `${currentResult.max_heart_rate} уд/мин` : undefined}
                      />
                      <Metric
                        label="Средний каденс"
                        value={currentResult?.avg_cadence !== undefined ? `${currentResult.avg_cadence} об/мин` : undefined}
                      />
                      <Metric
                        label="Калории"
                        value={currentResult?.calories !== undefined ? `${currentResult.calories} ккал` : undefined}
                      />
                    </div>
                  )}
                </div>
              )}
            </div>
          </form>

          {/* Критерии результата */}
          <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
            <h3 className="mb-4 text-lg font-semibold text-gray-800 dark:text-white">
              Критерии результата
            </h3>
            <ResultCriteriaManager
              result={currentResult}
              onUpdate={refreshMatchedGifts}
            />
          </div>

        </div>

        {/* Боковая панель */}
        <div className="space-y-6">
          {/* Призы от участника */}
          <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
            <h3 className="mb-4 text-lg font-semibold text-gray-800 dark:text-white">
              Призы от участника
            </h3>
            {gifts.length > 0 ? (
              <div className="space-y-3">
                {gifts.map((gift) => (
                  <div
                    key={gift.id}
                    className="rounded-lg border border-gray-200 p-3 dark:border-gray-700"
                  >
                    <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                      {gift.description}
                    </p>
                    {gift.criteria && gift.criteria.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1">
                        {gift.criteria.map((c) => (
                          <Badge
                            key={c.id}
                            color={getCriteriaColor(c.criteria_type)}
                            size="sm"
                          >
                            {c.name}
                          </Badge>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Нет призов
              </p>
            )}
          </div>

          {/* Подобранные призы (автоматически) */}
          <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
            <h3 className="mb-4 text-lg font-semibold text-gray-800 dark:text-white">
              Подобранные призы
            </h3>
            {participant.matched_gift_assignments && participant.matched_gift_assignments.length > 0 ? (
              <div className="space-y-3">
                {participant.matched_gift_assignments.map((assignment, index) => (
                  <div key={`${assignment.gift_id}-${assignment.target_rank || 'none'}-${index}`} className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                      {assignment.gift.description}
                    </p>
                    {assignment.gift.criteria && assignment.gift.criteria.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1">
                        {assignment.gift.criteria.map((c) => (
                          <Badge
                            key={c.id}
                            color={getCriteriaColor(c.criteria_type)}
                            size="sm"
                          >
                            {c.name}
                          </Badge>
                        ))}
                      </div>
                    )}
                    <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                      {formatPrizeAssignment(assignment)}
                    </p>
                  </div>
                ))}
              </div>
            ) : participant.matched_gifts && participant.matched_gifts.length > 0 ? (
              <div className="space-y-3">
                {participant.matched_gifts.map((gift, index) => (
                  <div key={gift.id || index} className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                    <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                      {gift.description}
                    </p>
                    {gift.criteria && gift.criteria.length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1">
                        {gift.criteria.map((c) => (
                          <Badge
                            key={c.id}
                            color={getCriteriaColor(c.criteria_type)}
                            size="sm"
                          >
                            {c.name}
                          </Badge>
                        ))}
                      </div>
                    )}
                    <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                      Подобран автоматически
                    </p>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-gray-500 dark:text-gray-400">
                Призы не подобраны
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
