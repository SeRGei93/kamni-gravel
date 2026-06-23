import { useCallback, useEffect, useRef, useState } from 'react';
import { participantLockApi } from '@/api/participants';
import { ApiError } from '@/api/client';
import type { LockStatus } from '@/types';
import {
  isStatusLockedByOther,
  ownerNameFromStatus,
  releaseOnEnd,
} from './participantLock.helpers';

// Реэкспорт чистых помощников для удобного импорта из этого модуля.
export {
  isStatusLockedByOther,
  ownerNameFromStatus,
  releaseOnEnd,
} from './participantLock.helpers';

// Захват лока продлевается heartbeat'ом заметно чаще, чем TTL на бэкенде (90с),
// чтобы лок не протух во время активного редактирования.
const HEARTBEAT_MS = 30_000;
// Пока показывается чужой лок и мы не редактируем — опрашиваем статус, чтобы
// автоматически снять баннер, когда другой администратор закончит.
const FOREIGN_POLL_MS = 15_000;
const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface UseParticipantLock {
  /** Текущее известное состояние лока (или null до первой загрузки). */
  lockStatus: LockStatus | null;
  /** true, когда запись держит другой администратор. */
  isLockedByOther: boolean;
  /** Имя администратора, удерживающего лок (если это не мы). */
  lockOwnerName?: string;
  /**
   * Пытается захватить лок перед входом в режим редактирования.
   * Возвращает true при успехе (вход разрешён) и false, если запись занята
   * другим администратором (или захват не удался).
   */
  beginEdit: () => Promise<boolean>;
  /** Сообщает, что секция вышла из редактирования; при закрытии последней — снимает лок. */
  endEdit: () => void;
}

/**
 * useParticipantLock управляет пессимистичной блокировкой редактирования участника:
 * захват по входу в редактирование, heartbeat, снятие при выходе/уходе со страницы,
 * а также индикация чужого лока. Один лок разделяют все секции страницы (счётчик
 * активных редактирований); снятие происходит при закрытии последней секции.
 */
export function useParticipantLock(participantId: number): UseParticipantLock {
  const [lockStatus, setLockStatus] = useState<LockStatus | null>(null);

  // Счётчик одновременно открытых на редактирование секций и зеркало "редактируем
  // ли сейчас" для обработчиков событий (которые читают актуальное значение из ref).
  const activeEditsRef = useRef(0);
  const editingRef = useRef(false);
  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const isLockedByOther = isStatusLockedByOther(lockStatus);
  const lockOwnerName = ownerNameFromStatus(lockStatus);

  const stopHeartbeat = useCallback(() => {
    if (heartbeatRef.current) {
      clearInterval(heartbeatRef.current);
      heartbeatRef.current = null;
    }
  }, []);

  // heartbeat повторно захватывает лок (POST идемпотентен): продлевает TTL, а если
  // лок успел протухнуть и его никто не занял — заново берёт его. 409 означает,
  // что лок перехватил другой администратор — прекращаем локальное редактирование.
  const heartbeat = useCallback(async () => {
    try {
      const status = await participantLockApi.acquire(participantId);
      setLockStatus(status);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        console.debug('[lock] heartbeat lost the lock (409)');
        setLockStatus((err.data as LockStatus) ?? null);
        stopHeartbeat();
        activeEditsRef.current = 0;
        editingRef.current = false;
      } else {
        console.debug('[lock] heartbeat failed (ignored)', err);
      }
    }
  }, [participantId, stopHeartbeat]);

  const startHeartbeat = useCallback(() => {
    if (heartbeatRef.current) return;
    heartbeatRef.current = setInterval(() => {
      void heartbeat();
    }, HEARTBEAT_MS);
  }, [heartbeat]);

  const beginEdit = useCallback(async (): Promise<boolean> => {
    try {
      const status = await participantLockApi.acquire(participantId);
      setLockStatus(status);
      activeEditsRef.current += 1;
      editingRef.current = true;
      startHeartbeat();
      return true;
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        console.debug('[lock] acquire denied (held by another admin)');
        setLockStatus((err.data as LockStatus) ?? null);
      } else {
        console.debug('[lock] acquire failed', err);
      }
      return false;
    }
  }, [participantId, startHeartbeat]);

  const endEdit = useCallback(() => {
    const { activeEditsAfter, release } = releaseOnEnd(activeEditsRef.current);
    activeEditsRef.current = activeEditsAfter;
    if (!release) {
      return; // другие секции ещё редактируются — лок держим
    }
    editingRef.current = false;
    stopHeartbeat();
    void participantLockApi.release(participantId).then(() => {
      setLockStatus((prev) =>
        prev && prev.is_mine
          ? { ...prev, locked: false, is_mine: false }
          : prev
      );
    });
  }, [participantId, stopHeartbeat]);

  // Первичная загрузка состояния лока.
  useEffect(() => {
    let cancelled = false;
    participantLockApi
      .get(participantId)
      .then((status) => {
        if (!cancelled) setLockStatus(status);
      })
      .catch((err) => console.debug('[lock] initial status fetch failed', err));
    return () => {
      cancelled = true;
    };
  }, [participantId]);

  // Опрос состояния, пока показывается чужой лок и мы не редактируем.
  useEffect(() => {
    if (!isLockedByOther) return;
    const timer = setInterval(() => {
      participantLockApi
        .get(participantId)
        .then(setLockStatus)
        .catch(() => {});
    }, FOREIGN_POLL_MS);
    return () => clearInterval(timer);
  }, [isLockedByOther, participantId]);

  // Снятие лока при закрытии вкладки/навигации и при размонтировании.
  useEffect(() => {
    // keepalive-DELETE переживает выгрузку страницы; sendBeacon не подходит —
    // он только POST и без заголовков, а эндпоинту нужен метод DELETE + Bearer.
    const releaseViaKeepalive = () => {
      if (!editingRef.current || typeof window === 'undefined') return;
      const token = localStorage.getItem('access_token');
      try {
        void fetch(`${API_URL}/api/participants/${participantId}/lock`, {
          method: 'DELETE',
          keepalive: true,
          headers: token ? { Authorization: `Bearer ${token}` } : undefined,
        });
      } catch {
        // best-effort
      }
    };

    // Возврат на вкладку во время редактирования — сразу освежаем/перезахватываем лок.
    const onVisibility = () => {
      if (document.visibilityState === 'visible' && editingRef.current) {
        void heartbeat();
      }
    };

    window.addEventListener('pagehide', releaseViaKeepalive);
    window.addEventListener('beforeunload', releaseViaKeepalive);
    document.addEventListener('visibilitychange', onVisibility);

    return () => {
      window.removeEventListener('pagehide', releaseViaKeepalive);
      window.removeEventListener('beforeunload', releaseViaKeepalive);
      document.removeEventListener('visibilitychange', onVisibility);
      stopHeartbeat();
      if (editingRef.current) {
        editingRef.current = false;
        activeEditsRef.current = 0;
        void participantLockApi.release(participantId);
      }
    };
  }, [participantId, heartbeat, stopHeartbeat]);

  return { lockStatus, isLockedByOther, lockOwnerName, beginEdit, endEdit };
}
