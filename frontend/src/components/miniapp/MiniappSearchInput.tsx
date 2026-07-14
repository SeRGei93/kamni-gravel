"use client";

import { useRef } from "react";
import { CloseLineIcon } from "@/icons";

interface MiniappSearchInputProps {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
  placeholder: string;
  ariaLabel: string;
}

export default function MiniappSearchInput({
  value,
  onChange,
  onClear,
  placeholder,
  ariaLabel,
}: MiniappSearchInputProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  const clear = () => {
    onClear();
    inputRef.current?.focus();
  };

  return (
    <div className="relative w-full">
      <input
        ref={inputRef}
        type="search"
        autoComplete="off"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        aria-label={ariaLabel}
        className="tg-divider tg-title h-10 w-full rounded-lg border bg-transparent py-2 pl-3 pr-10 text-sm outline-none focus:border-[var(--tg-button-color)] [&::-webkit-search-cancel-button]:appearance-none"
      />
      {value && (
        <button
          type="button"
          onClick={clear}
          aria-label="Очистить поиск"
          className="tg-muted absolute inset-y-0 right-0 flex w-10 items-center justify-center rounded-r-lg outline-none transition hover:text-[var(--tg-button-color)] focus-visible:ring-2 focus-visible:ring-[var(--tg-button-color)]"
        >
          <CloseLineIcon aria-hidden="true" className="size-4" />
        </button>
      )}
    </div>
  );
}
