'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

// Универсальный хук видимости колонок таблицы с сохранением в localStorage.
// Набор видимых колонок переживает перезагрузку страницы (per-browser).
//
// Hydration-safe: начальное состояние = defaultKeys, чтобы серверный HTML и
// первый клиентский рендер совпадали; localStorage читается в useEffect ПОСЛЕ
// монтирования. Чтение storage во время первого рендера вызвало бы hydration
// mismatch.

export interface ColumnPreferences {
  /** Видимые (переключаемые) ключи колонок. */
  visibleKeys: string[];
  /** Видима ли колонка с данным ключом. */
  isVisible: (key: string) => boolean;
  /** Переключить видимость колонки (только для переключаемых ключей). */
  toggle: (key: string) => void;
  /** Сбросить к набору по умолчанию. */
  reset: () => void;
}

interface StoredPrefs {
  v: number;
  visible: string[];
  known: string[];
}

const STORAGE_VERSION = 1;

/**
 * Согласует сохранённые ключи с текущим реестром:
 * - выбрасывает неизвестные (удалённые) колонки;
 * - сохраняет явный выбор пользователя для уже известных колонок;
 * - для новых колонок (которых не было на момент сохранения) применяет дефолт.
 * Порядок результата следует порядку allKeys (реестра).
 */
export function reconcile(
  stored: { visible: string[]; known: string[] },
  allKeys: string[],
  defaultKeys: string[],
): string[] {
  const knownAtSave = new Set(
    stored.known.length > 0 ? stored.known : stored.visible,
  );
  const storedVisible = new Set(stored.visible);
  return allKeys.filter((key) => {
    if (storedVisible.has(key)) return true; // пользователь оставил видимой
    if (!knownAtSave.has(key)) return defaultKeys.includes(key); // новая колонка
    return false; // пользователь скрыл ранее известную колонку
  });
}

export function useColumnPreferences(
  storageKey: string,
  allKeys: string[],
  defaultKeys: string[],
): ColumnPreferences {
  const [visibleKeys, setVisibleKeys] = useState<string[]>(defaultKeys);
  const skipNextPersist = useRef(true);

  // Загрузка из localStorage после монтирования.
  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      const raw = window.localStorage.getItem(storageKey);
      if (!raw) {
        console.debug('[useColumnPreferences] no stored prefs, using defaults', {
          storageKey,
        });
        setVisibleKeys(defaultKeys);
        return;
      }
      const parsed = JSON.parse(raw) as Partial<StoredPrefs>;
      if (!parsed || !Array.isArray(parsed.visible)) {
        setVisibleKeys(defaultKeys);
        return;
      }
      const next = reconcile(
        { visible: parsed.visible, known: parsed.known ?? [] },
        allKeys,
        defaultKeys,
      );
      console.debug('[useColumnPreferences] loaded', {
        storageKey,
        count: next.length,
      });
      setVisibleKeys(next);
    } catch (err) {
      console.error('[useColumnPreferences] failed to load prefs', {
        storageKey,
        error: err,
      });
      setVisibleKeys(defaultKeys);
    }
    // allKeys/defaultKeys — стабильные ссылки из реестра; перезагрузка по storageKey.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storageKey]);

  // Сохранение при изменении (пропускаем самый первый прогон до загрузки).
  useEffect(() => {
    if (skipNextPersist.current) {
      skipNextPersist.current = false;
      return;
    }
    if (typeof window === 'undefined') return;
    try {
      const payload: StoredPrefs = {
        v: STORAGE_VERSION,
        visible: visibleKeys,
        known: allKeys,
      };
      window.localStorage.setItem(storageKey, JSON.stringify(payload));
      console.debug('[useColumnPreferences] persisted', {
        storageKey,
        count: visibleKeys.length,
      });
    } catch (err) {
      console.error('[useColumnPreferences] failed to persist prefs', {
        storageKey,
        error: err,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visibleKeys, storageKey]);

  const toggle = useCallback(
    (key: string) => {
      if (!allKeys.includes(key)) return; // нельзя переключать всегда-видимые/неизвестные
      setVisibleKeys((prev) => {
        const isOn = prev.includes(key);
        console.debug('[useColumnPreferences] toggle', { key, visible: !isOn });
        // Сохраняем порядок реестра.
        const nextSet = new Set(prev);
        if (isOn) nextSet.delete(key);
        else nextSet.add(key);
        return allKeys.filter((k) => nextSet.has(k));
      });
    },
    [allKeys],
  );

  const reset = useCallback(() => {
    console.debug('[useColumnPreferences] reset to defaults');
    setVisibleKeys(defaultKeys);
  }, [defaultKeys]);

  const isVisible = useCallback(
    (key: string) => visibleKeys.includes(key),
    [visibleKeys],
  );

  return { visibleKeys, isVisible, toggle, reset };
}
