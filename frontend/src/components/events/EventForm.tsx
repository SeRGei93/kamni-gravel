'use client';

import React, { useState, useEffect } from 'react';
import Input from '../form/input/InputField';
import Label from '../form/Label';
import TextArea from '../form/input/TextArea';
import FileInput from '../form/input/FileInput';
import Switch from '../form/switch/Switch';
import Button from '../ui/button/Button';
import DatePicker from '../form/date-picker';
import type { Event, CreateEventRequest, UpdateEventRequest } from '@/types';

interface EventFormProps {
  event?: Event;
  onSubmit: (
    data: CreateEventRequest | UpdateEventRequest,
    gpxFile?: File
  ) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
}

export default function EventForm({
  event,
  onSubmit,
  onCancel,
  isLoading = false,
}: EventFormProps) {
  const [name, setName] = useState(event?.name || '');
  const [description, setDescription] = useState(event?.description || '');
  const [active, setActive] = useState(event?.active ?? true);
  const [startDate, setStartDate] = useState<string>(
    event?.start_date ? new Date(event.start_date).toISOString().split('T')[0] : ''
  );
  const [endDate, setEndDate] = useState<string>(
    event?.end_date ? new Date(event.end_date).toISOString().split('T')[0] : ''
  );
  const [gpxFile, setGpxFile] = useState<File | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const data: CreateEventRequest | UpdateEventRequest = {
      name,
      description,
      active,
      start_date: startDate || undefined,
      end_date: endDate || undefined,
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
        />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <Label>Дата начала</Label>
          <DatePicker
            id="start-date"
            mode="single"
            placeholder="Выберите дату начала"
            defaultDate={startDate || undefined}
            onChange={(selectedDates) => {
              if (selectedDates.length > 0) {
                setStartDate(selectedDates[0].toISOString().split('T')[0]);
              }
            }}
          />
        </div>

        <div>
          <Label>Дата окончания</Label>
          <DatePicker
            id="end-date"
            mode="single"
            placeholder="Выберите дату окончания"
            defaultDate={endDate || undefined}
            onChange={(selectedDates) => {
              if (selectedDates.length > 0) {
                setEndDate(selectedDates[0].toISOString().split('T')[0]);
              }
            }}
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
