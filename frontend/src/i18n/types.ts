// 本文件声明 Win 前端 i18n 层使用的共享消息类型。

export type AppLocale = "en" | "zh-CN"

export type TranslationParams = Record<string, string | number | boolean | null | undefined>

export type LocaleMessages = Record<string, string>
