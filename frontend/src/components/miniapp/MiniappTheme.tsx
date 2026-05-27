"use client";

import { useEffect } from "react";
import { getTelegramWebApp } from "@/utils/telegramWebApp";
import {
  DEFAULT_TELEGRAM_DARK_THEME,
  defaultTelegramDarkThemeStyle,
} from "./telegramTheme";

const CSS_VARIABLES = {
  bg_color: "--tg-bg-color",
  text_color: "--tg-text-color",
  hint_color: "--tg-hint-color",
  link_color: "--tg-link-color",
  button_color: "--tg-button-color",
  button_text_color: "--tg-button-text-color",
  secondary_bg_color: "--tg-secondary-bg-color",
  header_bg_color: "--tg-header-bg-color",
  accent_text_color: "--tg-accent-text-color",
  section_bg_color: "--tg-section-bg-color",
  section_header_text_color: "--tg-section-header-text-color",
  subtitle_text_color: "--tg-subtitle-text-color",
  destructive_text_color: "--tg-destructive-text-color",
} as const;

export default function MiniappTheme() {
  useEffect(() => {
    const webApp = getTelegramWebApp();
    const themeParams = webApp?.themeParams ?? {};
    const root = document.documentElement;

    Object.entries(defaultTelegramDarkThemeStyle).forEach(([name, value]) => {
      root.style.setProperty(name, String(value));
    });

    Object.entries(CSS_VARIABLES).forEach(([telegramKey, cssVariable]) => {
      const value = themeParams[telegramKey as keyof TelegramWebAppThemeParams];
      if (value) {
        root.style.setProperty(cssVariable, value);
      }
    });

    root.style.setProperty("--tg-separator-color", "rgba(255, 255, 255, 0.1)");
    root.style.setProperty(
      "--tg-hover-bg-color",
      mixTelegramColor(themeParams.section_bg_color, "#2b2b2b")
    );
    root.style.setProperty(
      "--tg-accent-soft-color",
      alphaTelegramColor(
        themeParams.button_color ?? DEFAULT_TELEGRAM_DARK_THEME.buttonColor,
        0.16
      )
    );
    root.style.setProperty(
      "--tg-skeleton-color",
      mixTelegramColor(themeParams.section_bg_color, DEFAULT_TELEGRAM_DARK_THEME.skeletonColor)
    );

    webApp?.setHeaderColor?.(themeParams.header_bg_color ?? DEFAULT_TELEGRAM_DARK_THEME.headerBgColor);
    webApp?.setBackgroundColor?.(
      themeParams.secondary_bg_color ?? DEFAULT_TELEGRAM_DARK_THEME.secondaryBgColor
    );
  }, []);

  return null;
}

function alphaTelegramColor(color: string, alpha: number): string {
  const rgb = hexToRgb(color);
  if (!rgb) {
    return DEFAULT_TELEGRAM_DARK_THEME.accentSoftColor;
  }

  return `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${alpha})`;
}

function mixTelegramColor(color: string | undefined, fallback: string): string {
  const rgb = hexToRgb(color ?? "");
  if (!rgb) {
    return fallback;
  }

  const lighter = 18;
  return `rgb(${Math.min(rgb.r + lighter, 255)}, ${Math.min(rgb.g + lighter, 255)}, ${Math.min(
    rgb.b + lighter,
    255
  )})`;
}

function hexToRgb(color: string): { r: number; g: number; b: number } | null {
  const normalized = color.trim().replace("#", "");
  if (!/^[0-9a-fA-F]{6}$/.test(normalized)) {
    return null;
  }

  return {
    r: Number.parseInt(normalized.slice(0, 2), 16),
    g: Number.parseInt(normalized.slice(2, 4), 16),
    b: Number.parseInt(normalized.slice(4, 6), 16),
  };
}
