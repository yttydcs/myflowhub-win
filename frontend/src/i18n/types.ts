// Context: declares shared message-key typing for the Win frontend i18n layer.

export type AppLocale = "en" | "zh-CN"

export type TranslationParams = Record<string, string | number | boolean | null | undefined>

export type LocaleMessages = Record<string, string>
