'use client';

import React, { useState, useEffect } from 'react';
import Input from '../form/input/InputField';
import Label from '../form/Label';
import TextArea from '../form/input/TextArea';
import FileInput from '../form/input/FileInput';
import Switch from '../form/switch/Switch';
import Button from '../ui/button/Button';
import type { Event, CreateEventRequest, UpdateEventRequest } from '@/types';
import { fromMinskDateTimeInput, MINSK_OFFSET_LABEL, toMinskDateTimeInput } from '@/utils/minskTime';

interface EventFormProps {
  event?: Event;
  onSubmit: (
    data: CreateEventRequest | UpdateEventRequest,
    gpxFile?: File
  ) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
}

const DEFAULT_PARTICIPATION_CONDITIONS = `УСЛОВИЯ УЧАСТИЯ (ДИСКЛЕЙМЕР) ‼️

Для обычного человека гравийная поездка на 200-300 км не является лёгкой прогулкой и требует хорошей физической и моральной подготовки, планирования питания и питья, а также наличие всего необходимого для ремонта велосипеда, оказания медпомощи и эвакуации себя.

Участие в КАМНЯХ означает полное принятие следующих условий:

1. Участниками КАМНЕЙ 200 признаются люди старше 18 лет, кто прошёл регистрацию, сделал взнос в призовой фонд и проехал хотя бы часть маршрута, что подтверждается Страва-ссылкой. Таковые попадают в лидерборд, при этом приоритет имеют участники, проехавшие маршрут целиком.

2. Самостоятельное обеспечение питанием и питьём 🍼🍔: участник обязан самостоятельно обеспечивать себя питанием и питьём на протяжении всей поездки. Рекомендуется употреблять 50-100 г углеводов каждый час (батончики, гели, фрукты, еда, изотон), не дожидаясь чувства голода, а также небольшое кол-во белков (орехи, сыр, мясо и т.п.). Питьё небольшими глотками каждые 15-20 минут, не дожидаясь жажды. Общий литраж индивидуален, от 0,5 до 1 литра в час, в идеале изотонические напитки (водный раствор с полезными минералами, углеводами: соль, мёд, сок и т.п.). Всегда иметь запас воды и питания, пополнять его при первой возможности.

3. Безопасность 🤌: участник несёт полную ответственность за свою безопасность. Обязателен шлем, исправный велосипед, фонарики, запас денег, внимание к самочувствию. Выход на маршрут только в здоровом состоянии. Рекомендуется иметь при себе аптечку первой помощи: бинты, пластырь, дезинфицирующий раствор, обезболивающие препараты. При плохом самочувствии — остановить прохождение дистанции, сойти с дороги в безопасное место, привести себя в чувство, а при невозможности – сойти с дистанции. При жаре не допускать перегрева, охлаждаться водой, тенью.

4. Выезд на дороги, соблюдение ПДД 🚗 и законов РБ: участник несёт полную ответственность за соблюдение правил дорожного движения, законов Республики Беларусь, и безопасность при выезде на дороги общего пользования.

5. Проблемы на маршруте 🧚‍♀️: при возникновении любого рода проблем на маршруте участник надеется на себя. Помощь другим участникам приветствуется, однако, надеяться на неё не стоит. Обязательно иметь при себе заряженный телефон. Телефоны экстренных служб: 103 скорая медпомощь, 112 МЧС, 102 Милиция. При возникновении непреодолимых трудностей пишите в данный чат. Возможно, кто-то сможет вам помочь.

6. Сход с дистанции ⛔️: в случае схода с дистанции участник самостоятельно добирается до дома. Транспорт не предоставляется. Если вы осознали, что неспособны доехать маршрут до конца, вызывайте эвакуацию (друзья, такси, попутки) либо двигайтесь в направлении ближайшей ЖД-станции.

7. Ремонт и техобслуживание 🚴‍♀️: участник выходит на маршрут на полностью исправном велосипеде с работающей тормозной системой, имеет при себе необходимые инструменты и запчасти для ремонта. Особенно рекомендуется иметь несколько запасных камер, латки, червяки для бескамерки, т.к. вероятность проколов высока.

8. Навигация и маршрут 🏞: участники обязаны самостоятельно ориентироваться на маршруте и иметь при себе навигационные средства с достаточным зарядом до завершения маршрута. Разметка на треке отсутствует.

9. Неопытным велосипедистам 🍬: тем, кто ещё не ездил в длительные поездки, рекомендуется прохождение трека в компании. Так веселее, безопаснее и проще преодолевать трудности.

10. Риски и ответственность 🤌: участие в гонке сопряжено с определенными рисками, включая травмы и аварии. Участники принимают участие на свой страх и риск, создатели маршрута не несут ответственности за возможные инциденты, любые происшествия, связанные с участием в заезде. Принимая участие в заезде, каждый участник подтверждает свое согласие с данными условиями и принимает на себя все связанные риски.`;

export default function EventForm({
  event,
  onSubmit,
  onCancel,
  isLoading = false,
}: EventFormProps) {
  const [name, setName] = useState(event?.name || '');
  const [description, setDescription] = useState(event?.description || '');
  const [participationConditions, setParticipationConditions] = useState(
    event?.participation_conditions || DEFAULT_PARTICIPATION_CONDITIONS
  );
  const [active, setActive] = useState(event?.active ?? true);
  const [stopResults, setStopResults] = useState(event?.stop_results ?? false);
  const [stopGifts, setStopGifts] = useState(event?.stop_gifts ?? false);
  const [startDate, setStartDate] = useState<string>(
    toMinskDateTimeInput(event?.start_date)
  );
  const [endDate, setEndDate] = useState<string>(
    toMinskDateTimeInput(event?.end_date)
  );
  const [gpxFile, setGpxFile] = useState<File | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const data: CreateEventRequest | UpdateEventRequest = {
      name,
      description,
      participation_conditions: participationConditions,
      active,
      stop_results: stopResults,
      stop_gifts: stopGifts,
      start_date: fromMinskDateTimeInput(startDate),
      end_date: fromMinskDateTimeInput(endDate),
    };

    await onSubmit(data, gpxFile || undefined);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div>
        <Label>
          Название <span className="text-error-500">*</span>
        </Label>
        <Input
          type="text"
          placeholder="Название события"
          defaultValue={name}
          onChange={(e) => setName(e.target.value)}
          required
          disabled={isLoading}
        />
      </div>

      <div>
        <Label>Описание</Label>
        <TextArea
          placeholder="Описание события"
          value={description}
          onChange={setDescription}
          rows={4}
          disabled={isLoading}
        />
      </div>

      <div>
        <Label>Условия участия</Label>
        <TextArea
          placeholder="Условия участия"
          value={participationConditions}
          onChange={setParticipationConditions}
          rows={14}
          disabled={isLoading}
        />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <Label>Дата и время начала ({MINSK_OFFSET_LABEL})</Label>
          <Input
            id="start-date"
            type="datetime-local"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            disabled={isLoading}
          />
        </div>

        <div>
          <Label>Дата и время окончания ({MINSK_OFFSET_LABEL})</Label>
          <Input
            id="end-date"
            type="datetime-local"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            disabled={isLoading}
          />
        </div>
      </div>

      <div>
        <Label>GPX файл</Label>
        <FileInput
          accept=".gpx"
          onChange={(e) => setGpxFile(e.target.files?.[0] || null)}
          disabled={isLoading}
        />
        <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {gpxFile ? (
            <span>Будет загружен: {gpxFile.name}</span>
          ) : event?.gpx_file_path ? (
            <span>Текущий файл: {event.gpx_file_path}</span>
          ) : (
            <span>Выберите GPX файл маршрута. Он будет сохранён в общем хранилище событий.</span>
          )}
        </div>
      </div>

      <div>
        <Switch
          label="Активное событие"
          defaultChecked={active}
          onChange={setActive}
        />
      </div>

      <div className="space-y-3">
        <Switch
          label="Остановить добавление результатов"
          defaultChecked={stopResults}
          onChange={setStopResults}
        />
        <Switch
          label="Остановить добавление призов"
          defaultChecked={stopGifts}
          onChange={setStopGifts}
        />
        <div className="text-xs text-gray-500 dark:text-gray-400">
          Бот перестанет принимать результаты/призы и скроет кнопки. Дата
          окончания закрывает только регистрацию — приём призов и результатов
          управляется этими галочками.
        </div>
      </div>

      <div className="flex items-center gap-3 justify-end">
        <Button type="button" variant="outline" onClick={onCancel} disabled={isLoading}>
          Отмена
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading ? 'Сохранение...' : event ? 'Сохранить изменения' : 'Создать событие'}
        </Button>
      </div>
    </form>
  );
}
