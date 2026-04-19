import type { AppSettings, Locale, ThemeMode } from "@/lib/contracts";

export function resolveLocale(settings: AppSettings, candidates?: ReadonlyArray<string>): Locale {
  if (settings.localeMode === "manual") {
    return settings.locale;
  }

  const locales = candidates && candidates.length > 0 ? candidates : navigator.languages;
  const primary = locales[0]?.toLowerCase() ?? "en";
  if (primary.startsWith("zh")) {
    return "zh-CN";
  }
  if (primary.startsWith("ja")) {
    return "ja";
  }
  return "en";
}

export function resolveTheme(mode: ThemeMode, prefersDark: boolean) {
  if (mode === "light") {
    return "light" as const;
  }
  if (mode === "dark") {
    return "dark" as const;
  }
  return prefersDark ? ("dark" as const) : ("light" as const);
}
