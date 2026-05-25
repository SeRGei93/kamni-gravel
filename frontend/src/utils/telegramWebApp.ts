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
