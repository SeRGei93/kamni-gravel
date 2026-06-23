import type { LockStatus } from '@/types';

// Чистые помощники для логики блокировки редактирования участника.
// Вынесены в отдельный модуль без рантайм-импортов (@/api/*), чтобы их можно
// было покрыть модульными тестами в node-окружении vitest (как остальные тесты
// проекта). Сам хук useParticipantLock переиспользует эти функции.

// Запись заблокирована другим администратором (а не нами).
export function isStatusLockedByOther(status: LockStatus | null): boolean {
  return Boolean(status?.locked && !status.is_mine);
}

// Имя владельца чужого лока (undefined, если лок наш или его нет).
export function ownerNameFromStatus(status: LockStatus | null): string | undefined {
  return isStatusLockedByOther(status) ? status?.locked_by_username : undefined;
}

// releaseOnEnd определяет, нужно ли снимать лок после выхода одной секции из
// редактирования. Один лок разделяют все секции страницы, поэтому снимаем его
// только когда закрылась последняя (счётчик дошёл до нуля).
export function releaseOnEnd(activeEditsBefore: number): {
  activeEditsAfter: number;
  release: boolean;
} {
  const activeEditsAfter = Math.max(0, activeEditsBefore - 1);
  return {
    activeEditsAfter,
    release: activeEditsBefore > 0 && activeEditsAfter === 0,
  };
}
