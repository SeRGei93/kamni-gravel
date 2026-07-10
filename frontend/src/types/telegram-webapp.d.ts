export {};

declare global {
  interface Window {
    Telegram?: {
      WebApp?: TelegramWebApp;
    };
  }

  interface TelegramWebApp {
    initData: string;
    initDataUnsafe?: TelegramWebAppInitDataUnsafe;
    colorScheme: 'light' | 'dark';
    themeParams: TelegramWebAppThemeParams;
    isExpanded: boolean;
    viewportHeight: number;
    viewportStableHeight: number;
    ready: () => void;
    expand: () => void;
    close: () => void;
    openLink?: (url: string, options?: { try_instant_view?: boolean }) => void;
    openTelegramLink?: (url: string) => void;
    setHeaderColor?: (color: string) => void;
    setBackgroundColor?: (color: string) => void;
    BackButton?: TelegramWebAppBackButton;
  }

  interface TelegramWebAppBackButton {
    isVisible: boolean;
    onClick: (callback: () => void) => TelegramWebAppBackButton;
    offClick: (callback: () => void) => TelegramWebAppBackButton;
    show: () => TelegramWebAppBackButton;
    hide: () => TelegramWebAppBackButton;
  }

  interface TelegramWebAppInitDataUnsafe {
    query_id?: string;
    user?: TelegramWebAppUser;
    auth_date?: number;
    hash?: string;
  }

  interface TelegramWebAppUser {
    id: number;
    is_bot?: boolean;
    first_name?: string;
    last_name?: string;
    username?: string;
    language_code?: string;
    is_premium?: boolean;
    photo_url?: string;
  }

  interface TelegramWebAppThemeParams {
    bg_color?: string;
    text_color?: string;
    hint_color?: string;
    link_color?: string;
    button_color?: string;
    button_text_color?: string;
    secondary_bg_color?: string;
    header_bg_color?: string;
    accent_text_color?: string;
    section_bg_color?: string;
    section_header_text_color?: string;
    subtitle_text_color?: string;
    destructive_text_color?: string;
  }
}
