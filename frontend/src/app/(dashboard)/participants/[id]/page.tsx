'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { participantsApi } from '@/api/participants';
import { resultsApi } from '@/api/results';
import type { ParticipantDetail, Gift, Result } from '@/types';
import Badge from '@/components/ui/badge/Badge';
import Button from '@/components/ui/button/Button';
import TextArea from '@/components/form/input/TextArea';
import TimeInput from '@/components/participants/TimeInput';
import ResultCriteriaManager from '@/components/participants/ResultCriteriaManager';
import { getCriteriaColor } from '@/utils/criteria';
import { ChevronLeftIcon, PencilIcon, CheckLineIcon, CloseLineIcon } from '@/icons';
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

export default function ParticipantDetailPage() {
  const params = useParams();
  const participantId = Number(params.id);

  const [participant, setParticipant] = useState<ParticipantDetail | null>(null);
  const [gifts, setGifts] = useState<Gift[]>([]);
  const [currentResult, setCurrentResult] = useState<Result | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isEditingNotes, setIsEditingNotes] = useState(false);
  const [isSavingNotes, setIsSavingNotes] = useState(false);

  // Редактируемые поля участника
  const [notes, setNotes] = useState('');
  
  // Редактируемые поля результата
  const [elapsedTimeSec, setElapsedTimeSec] = useState<number | undefined>();
  const [movingTimeSec, setMovingTimeSec] = useState<number | undefined>();
  const [isEditingResult, setIsEditingResult] = useState(false);
  const [isSavingResult, setIsSavingResult] = useState(false);

  useEffect(() => {
    loadParticipant();
  }, [participantId]);

  const loadParticipant = async () => {
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
      const current = resultsData.results.find((r) => r.is_current);
      setCurrentResult(current || null);
      
      setElapsedTimeSec(participantData.elapsed_time_sec);
      setMovingTimeSec(participantData.moving_time_sec);
      setNotes(participantData.notes || '');
    } catch (err) {
      setError('Ошибка загрузки данных участника');
      console.error('Failed to load participant:', err);
    } finally {
      setIsLoading(false);
    }
  };

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

  const handleSaveResult = async () => {
    if (!participant) return;

    try {
      setIsSavingResult(true);
      const resultId = await getCurrentResultId();
      
      if (resultId) {
        await resultsApi.update(resultId, {
          elapsed_time_sec: elapsedTimeSec,
          moving_time_sec: movingTimeSec,
        });
      }
      
      setIsEditingResult(false);
      await loadParticipant();
    } catch (err) {
      setError('Ошибка сохранения времени');
      console.error('Failed to update result:', err);
    } finally {
      setIsSavingResult(false);
    }
  };

  const handleCancelResultEdit = () => {
    if (!participant) return;
    setElapsedTimeSec(participant.elapsed_time_sec);
    setMovingTimeSec(participant.moving_time_sec);
    setIsEditingResult(false);
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
          <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-800 dark:text-white">
                Результат
              </h3>
              {participant.is_finished && !isEditingResult && (
                <Button
                  size="sm"
                  variant="outline"
                  startIcon={<PencilIcon />}
                  onClick={() => setIsEditingResult(true)}
                >
                  Изменить время
                </Button>
              )}
              {isEditingResult && (
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCancelResultEdit}
                    disabled={isSavingResult}
                  >
                    Отмена
                  </Button>
                  <Button
                    size="sm"
                    onClick={handleSaveResult}
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
              {participant.is_finished && (
                <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
                  {isEditingResult ? (
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                      <TimeInput
                        label="Общее время"
                        value={elapsedTimeSec}
                        onChange={setElapsedTimeSec}
                      />
                      <TimeInput
                        label="Время в пути"
                        value={movingTimeSec}
                        onChange={setMovingTimeSec}
                      />
                    </div>
                  ) : (
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                      <div>
                        <p className="mb-1 text-xs text-gray-500 dark:text-gray-400">
                          Общее время
                        </p>
                        <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                          {participant.elapsed_time || '-'}
                        </p>
                      </div>
                      <div>
                        <p className="mb-1 text-xs text-gray-500 dark:text-gray-400">
                          Время в пути
                        </p>
                        <p className="text-sm font-medium text-gray-800 dark:text-white/90">
                          {participant.moving_time || '-'}
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Критерии результата */}
          <div className="rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-white/[0.03] lg:p-6">
            <h3 className="mb-4 text-lg font-semibold text-gray-800 dark:text-white">
              Критерии результата
            </h3>
            <ResultCriteriaManager
              result={currentResult}
              onUpdate={loadParticipant}
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
            {participant.matched_gifts && participant.matched_gifts.length > 0 ? (
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
