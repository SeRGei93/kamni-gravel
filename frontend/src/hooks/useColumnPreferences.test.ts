import { describe, expect, it } from 'vitest';
import { reconcile } from './useColumnPreferences';

// Реестр: 4 переключаемые колонки, 3 из них видимы по умолчанию.
const ALL = ['a', 'b', 'c', 'd'];
const DEFAULTS = ['a', 'b', 'c'];

describe('reconcile', () => {
  it('keeps the default set when stored snapshot matches the registry', () => {
    const result = reconcile({ visible: DEFAULTS, known: ALL }, ALL, DEFAULTS);
    expect(result).toEqual(['a', 'b', 'c']);
  });

  it('respects an explicit user choice (hidden default stays hidden)', () => {
    // Пользователь скрыл дефолтную колонку "b".
    const result = reconcile(
      { visible: ['a', 'c'], known: ALL },
      ALL,
      DEFAULTS,
    );
    expect(result).toEqual(['a', 'c']);
  });

  it('keeps a non-default column the user explicitly enabled', () => {
    const result = reconcile(
      { visible: ['a', 'b', 'c', 'd'], known: ALL },
      ALL,
      DEFAULTS,
    );
    expect(result).toEqual(['a', 'b', 'c', 'd']);
  });

  it('drops keys that no longer exist in the registry', () => {
    // Старое хранилище знало про удалённую колонку "x".
    const result = reconcile(
      { visible: ['a', 'x'], known: ['a', 'b', 'x'] },
      ALL,
      DEFAULTS,
    );
    expect(result).not.toContain('x');
    expect(result).toContain('a');
  });

  it('applies defaults for newly added columns, leaves non-default new columns off', () => {
    // На момент сохранения реестр знал только про "a" и "b"; пользователь скрыл "b".
    // Колонки "c" (дефолтная) и "d" (не дефолтная) добавлены позже.
    const result = reconcile(
      { visible: ['a'], known: ['a', 'b'] },
      ALL,
      DEFAULTS,
    );
    expect(result).toContain('a'); // оставлена пользователем
    expect(result).not.toContain('b'); // скрыта пользователем, остаётся скрытой
    expect(result).toContain('c'); // новая дефолтная — показана
    expect(result).not.toContain('d'); // новая не дефолтная — скрыта
  });

  it('preserves registry order regardless of stored order', () => {
    const result = reconcile(
      { visible: ['c', 'a'], known: ALL },
      ALL,
      DEFAULTS,
    );
    expect(result).toEqual(['a', 'c']);
  });

  it('round-trips the persisted JSON payload shape', () => {
    const visible = ['a', 'c'];
    const raw = JSON.stringify({ v: 1, visible, known: ALL });
    const parsed = JSON.parse(raw) as { visible: string[]; known: string[] };
    const result = reconcile(
      { visible: parsed.visible, known: parsed.known },
      ALL,
      DEFAULTS,
    );
    expect(result).toEqual(['a', 'c']);
  });
});
