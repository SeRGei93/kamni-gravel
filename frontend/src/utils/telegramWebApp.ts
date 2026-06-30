let missingRuntimeWarned = false;

export type TelegramColorScheme = 'light' | 'dark';

export function getTelegramWebApp(): TelegramWebApp | null {
  if (typeof window === 'undefined') {
    return null;
  }

  return window.Telegram?.WebApp ?? null;
}

export function isTelegramWebAppAvailable(): boolean {
  return getTelegramWebApp() !== null;
}

export function isFallbackBrowserMode(): boolean {
  return !isTelegramWebAppAvailable();
}

export function getTelegramInitData(): string {
  const webApp = getTelegramWebApp();
  if (!webApp?.initData) {
    warnMissingTelegramRuntime();
    return '';
  }

  return webApp.initData;
}

// waitForTelegramInitData ждёт, пока telegram-web-app.js инициализирует initData.
// Скрипт грузится асинхронно (Script strategy="afterInteractive"), поэтому первый
// эффект страницы может выстрелить раньше — тогда запрос уходит с пустым initData
// и бэкенд отвечает 401 "Missing Telegram init data". Возвращает initData как
// только он появился, либо '' по таймауту (браузерный фолбэк — без Telegram).
export async function waitForTelegramInitData(
  timeoutMs = 3000,
  intervalMs = 50
): Promise<string> {
  if (typeof window === 'undefined') {
    return '';
  }

  const deadline = Date.now() + timeoutMs;
  for (;;) {
    const initData = getTelegramWebApp()?.initData ?? '';
    if (initData) {
      return initData;
    }
    if (Date.now() >= deadline) {
      return '';
    }
    await new Promise((resolve) => {
      window.setTimeout(resolve, intervalMs);
    });
  }
}

export function readyTelegramWebApp(): void {
  const webApp = getTelegramWebApp();
  if (!webApp) {
    warnMissingTelegramRuntime();
    return;
  }

  webApp.ready();
}

export function expandTelegramWebApp(): void {
  const webApp = getTelegramWebApp();
  if (!webApp) {
    warnMissingTelegramRuntime();
    return;
  }

  webApp.expand();
}

// openTelegramProfile открывает профиль участника по его username. Внутри Telegram
// используется нативный openTelegramLink, в браузерном фолбэке — обычное окно.
export function openTelegramProfile(username: string): void {
  const handle = username.replace(/^@+/, '').trim();
  if (!handle) {
    return;
  }

  const url = `https://t.me/${handle}`;
  const webApp = getTelegramWebApp();
  if (webApp?.openTelegramLink) {
    webApp.openTelegramLink(url);
    return;
  }

  if (typeof window !== 'undefined') {
    window.open(url, '_blank', 'noopener,noreferrer');
  }
}

export function getTelegramThemeParams(): TelegramWebAppThemeParams {
  return getTelegramWebApp()?.themeParams ?? {};
}

export function getTelegramColorScheme(): TelegramColorScheme {
  return getTelegramWebApp()?.colorScheme ?? 'light';
}

export function getTelegramViewport(): {
  height: number;
  stableHeight: number;
  isExpanded: boolean;
} {
  const webApp = getTelegramWebApp();
  return {
    height: webApp?.viewportHeight ?? 0,
    stableHeight: webApp?.viewportStableHeight ?? 0,
    isExpanded: webApp?.isExpanded ?? false,
  };
}

export function warnMissingTelegramRuntime(): void {
  if (missingRuntimeWarned) {
    return;
  }

  missingRuntimeWarned = true;
  console.warn('[miniapp] Telegram WebApp runtime is unavailable; using browser fallback mode');
}
