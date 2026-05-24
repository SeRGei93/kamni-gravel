'use client';

import React, { useState, useEffect } from 'react';
import Input from '../form/input/InputField';
import Label from '../form/Label';
import { secondsToTimeString, timeStringToSeconds } from '@/utils/time';

interface TimeInputProps {
  label: string;
  value: number | undefined;
  onChange: (seconds: number | undefined) => void;
  disabled?: boolean;
}

export default function TimeInput({
  label,
  value,
  onChange,
  disabled = false,
}: TimeInputProps) {
  const [timeString, setTimeString] = useState(
    value ? secondsToTimeString(value) : ''
  );
  const [error, setError] = useState(false);

  useEffect(() => {
    if (value !== undefined) {
      setTimeString(secondsToTimeString(value));
    } else {
      setTimeString('');
    }
  }, [value]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setTimeString(newValue);

    if (newValue === '') {
      setError(false);
      onChange(undefined);
      return;
    }

    const seconds = timeStringToSeconds(newValue);
    if (seconds === null) {
      setError(true);
    } else {
      setError(false);
      onChange(seconds);
    }
  };

  return (
    <div>
      <Label>{label}</Label>
      <Input
        type="text"
        placeholder="ЧЧ:ММ:СС"
        defaultValue={timeString}
        onChange={handleChange}
        disabled={disabled}
        error={error}
        hint={error ? 'Неверный формат. Используйте ЧЧ:ММ:СС' : ''}
      />
    </div>
  );
}
