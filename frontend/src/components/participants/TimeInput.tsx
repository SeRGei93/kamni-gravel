'use client';

import React, { useState } from 'react';
import Input from '../form/input/InputField';
import Label from '../form/Label';
import { secondsToTimeString, timeStringToSeconds } from '@/utils/time';

interface TimeInputProps {
  label: string;
  value: number | undefined;
  onChange: (seconds: number | undefined) => void;
  disabled?: boolean;
  required?: boolean;
}

export default function TimeInput({
  label,
  value,
  onChange,
  disabled = false,
  required = false,
}: TimeInputProps) {
  const [timeString, setTimeString] = useState(
    value ? secondsToTimeString(value) : ''
  );
  const [error, setError] = useState(false);

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
      onChange(undefined);
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
        value={timeString}
        onChange={handleChange}
        disabled={disabled}
        required={required}
        error={error}
        hint={error ? 'Неверный формат. Используйте ЧЧ:ММ:СС' : ''}
      />
    </div>
  );
}
